package bundle

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

const maxSignals = 30

var (
	annotationPattern = regexp.MustCompile(`::error(?:\s+([^:]+))?::(.*)$`)
	fileLinePattern   = regexp.MustCompile(`^(.+\.(?:go|ts|tsx|js|jsx|py|rb|rs|java|kt|cs|php|c|cc|cpp|h|hpp|sql|yaml|yml)):(\d+)(?::\d+)?:\s*(.+)$`)
	failPattern       = regexp.MustCompile(`^--- FAIL:\s+([A-Za-z0-9_./-]+)`)
	missingEnvPattern = regexp.MustCompile(`(?i)(?:missing|required|undefined|not set).*\b([A-Z][A-Z0-9_]{2,})\b|\b([A-Z][A-Z0-9_]{2,})\b.*(?:missing|required|undefined|not set)`)
	timestampPattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T[^\s]+\s+`)
)

type Normalizer struct {
	Now func() time.Time
}

type evidenceInput struct {
	Name          string
	Source        string
	ParseMetadata bool
}

var evidenceInputs = []evidenceInput{
	{Name: project.GitHubActionsLogName, Source: "github_actions", ParseMetadata: true},
	{Name: project.GoTestLogName, Source: "local_go_test"},
	{Name: project.ShellCommandLogName, Source: "local_shell"},
}

func (n Normalizer) NormalizeRun(run project.Run) (NormalizedEvidence, error) {
	if n.Now == nil {
		n.Now = time.Now
	}

	normalized := NormalizedEvidence{
		Version:     SchemaVersion,
		GeneratedAt: n.Now().UTC(),
		Run: RunMetadata{
			RunID: run.ID,
		},
	}
	seen := map[string]bool{}
	truncatedSignals := false
	foundEvidence := false

	for _, input := range evidenceInputs {
		evidencePath := run.EvidencePath(input.Name)
		file, err := os.Open(evidencePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return NormalizedEvidence{}, fmt.Errorf("open evidence log %s: %w", run.RelativePath(evidencePath), err)
		}
		foundEvidence = true
		setRunSource(&normalized.Run, input.Source)
		normalized.Sources = append(normalized.Sources, EvidenceSource{
			Source: input.Source,
			Path:   run.RelativePath(evidencePath),
		})
		wasTruncated, err := scanEvidenceFile(file, input, run, &normalized, seen, run.RelativePath(evidencePath))
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return NormalizedEvidence{}, err
		}
		if wasTruncated {
			truncatedSignals = true
		}
	}
	foundReports, err := scanJUnitReports(run, &normalized, seen)
	if err != nil {
		return NormalizedEvidence{}, err
	}
	if foundReports {
		foundEvidence = true
		setRunSource(&normalized.Run, "junit_report")
	}
	if !foundEvidence {
		return NormalizedEvidence{}, fmt.Errorf("no evidence logs found for run %s", run.ID)
	}
	if normalized.Run.Source == "" {
		normalized.Run.Source = "local"
	}
	if normalized.Run.RunID == "" {
		normalized.Run.RunID = run.ID
	}
	if truncatedSignals {
		normalized.Warnings = append(normalized.Warnings, fmt.Sprintf("signal list capped at %d entries", maxSignals))
	}
	if len(normalized.Signals) == 0 {
		normalized.Warnings = append(normalized.Warnings, "no recognizable failure signals were extracted")
	}
	return normalized, nil
}

func scanEvidenceFile(file *os.File, input evidenceInput, run project.Run, normalized *NormalizedEvidence, seen map[string]bool, rawPath string) (bool, error) {
	var currentJob jobContext
	truncatedSignals := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if input.ParseMetadata {
			parseEvidenceMetadata(line, &normalized.Run)
		}
		if job, ok := parseJobHeader(line); ok {
			currentJob = job
			normalized.Sources = append(normalized.Sources, EvidenceSource{
				Source: input.Source,
				Path:   rawPath,
				Job:    job.Name,
				JobID:  job.ID,
			})
			continue
		}
		if strings.HasPrefix(line, "--- tailchase-end-job ") {
			currentJob = jobContext{}
			continue
		}

		for _, signal := range extractSignals(cleanLogLine(line), currentJob, rawPath, input.Source) {
			key := signalKey(signal)
			if seen[key] {
				continue
			}
			seen[key] = true
			if len(normalized.Signals) >= maxSignals {
				truncatedSignals = true
				continue
			}
			normalized.Signals = append(normalized.Signals, signal)
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan evidence log: %w", err)
	}
	return truncatedSignals, nil
}

func scanJUnitReports(run project.Run, normalized *NormalizedEvidence, seen map[string]bool) (bool, error) {
	pattern := filepath.Join(run.EvidenceDir(), project.TestReportsDirName, "*.xml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return false, err
	}
	for _, path := range paths {
		if err := scanJUnitReport(path, run, normalized, seen); err != nil {
			return false, err
		}
	}
	return len(paths) > 0, nil
}

func scanJUnitReport(path string, run project.Run, normalized *NormalizedEvidence, seen map[string]bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc junitDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse junit report %s: %w", run.RelativePath(path), err)
	}
	rawPath := run.RelativePath(path)
	normalized.Sources = append(normalized.Sources, EvidenceSource{Source: "junit_report", Path: rawPath})
	for _, testCase := range doc.allCases() {
		failure := testCase.failure()
		if failure == nil {
			continue
		}
		signal := Signal{
			Type:           "test_report_failure",
			Source:         "junit_report",
			Message:        strings.TrimSpace(firstNonEmpty(failure.Message, failure.Type, "failing test: "+testCase.displayName())),
			File:           strings.TrimSpace(testCase.File),
			Confidence:     "high",
			RawExcerpt:     strings.TrimSpace(firstNonEmpty(failure.Text, failure.Message)),
			RawExcerptPath: rawPath,
		}
		if signal.Message == "" {
			signal.Message = "failing test: " + testCase.displayName()
		}
		key := signalKey(signal)
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(normalized.Signals) >= maxSignals {
			normalized.Warnings = append(normalized.Warnings, fmt.Sprintf("signal list capped at %d entries", maxSignals))
			return nil
		}
		normalized.Signals = append(normalized.Signals, signal)
	}
	return nil
}

type junitDocument struct {
	XMLName xml.Name
	Suites  []junitSuite `xml:"testsuite"`
	Cases   []junitCase  `xml:"testcase"`
}

type junitSuite struct {
	Cases []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	File      string        `xml:"file,attr"`
	Failure   *junitFailure `xml:"failure"`
	Error     *junitFailure `xml:"error"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

func (d junitDocument) allCases() []junitCase {
	cases := append([]junitCase{}, d.Cases...)
	for _, suite := range d.Suites {
		cases = append(cases, suite.Cases...)
	}
	return cases
}

func (c junitCase) failure() *junitFailure {
	if c.Failure != nil {
		return c.Failure
	}
	return c.Error
}

func (c junitCase) displayName() string {
	if c.Classname == "" {
		return c.Name
	}
	if c.Name == "" {
		return c.Classname
	}
	return c.Classname + "." + c.Name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func setRunSource(run *RunMetadata, source string) {
	if run.Source == "" {
		run.Source = source
		return
	}
	if run.Source != source {
		run.Source = "mixed"
	}
}

func parseEvidenceMetadata(line string, run *RunMetadata) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(key) {
	case "repository":
		run.Repository = value
	case "run_id":
		run.RunID = value
	case "collected_at":
		run.CollectedAt = value
	}
}

func WriteNormalizedEvidence(run project.Run, normalized NormalizedEvidence) error {
	if normalized.Version == 0 {
		normalized.Version = SchemaVersion
	}
	data, err := yaml.Marshal(normalized)
	if err != nil {
		return err
	}
	return run.WriteArtifactFile(project.NormalizedEvidenceName, project.ArtifactNormalizedEvidence, "normalized_evidence", data)
}

func ReadNormalizedEvidence(run project.Run) (NormalizedEvidence, error) {
	data, err := run.ReadArtifactFile(project.NormalizedEvidenceName)
	if err != nil {
		return NormalizedEvidence{}, fmt.Errorf("read normalized evidence: %w", err)
	}
	var normalized NormalizedEvidence
	if err := yaml.Unmarshal(data, &normalized); err != nil {
		return NormalizedEvidence{}, fmt.Errorf("parse normalized evidence: %w", err)
	}
	if normalized.Version == 0 {
		normalized.Version = SchemaVersion
	}
	if normalized.Version != SchemaVersion {
		return NormalizedEvidence{}, fmt.Errorf("unsupported normalized evidence version %d", normalized.Version)
	}
	return normalized, nil
}

type jobContext struct {
	ID   int64
	Name string
}

func parseJobHeader(line string) (jobContext, bool) {
	if !strings.HasPrefix(line, "--- tailchase-job ") {
		return jobContext{}, false
	}
	idValue := headerField(line, "id")
	nameValue := headerField(line, "name")
	id, _ := strconv.ParseInt(idValue, 10, 64)
	return jobContext{ID: id, Name: nameValue}, true
}

func headerField(line string, field string) string {
	key := field + "="
	start := strings.Index(line, key)
	if start < 0 {
		return ""
	}
	start += len(key)
	if start >= len(line) {
		return ""
	}
	if line[start] != '"' {
		end := strings.IndexByte(line[start:], ' ')
		if end < 0 {
			return strings.Trim(line[start:], "- ")
		}
		return strings.Trim(line[start:start+end], "- ")
	}
	rest := line[start:]
	end := strings.Index(rest[1:], `"`)
	if end < 0 {
		return strings.Trim(rest, `"`)
	}
	unquoted, err := strconv.Unquote(rest[:end+2])
	if err != nil {
		return strings.Trim(rest[:end+2], `"`)
	}
	return unquoted
}

func extractSignals(line string, job jobContext, rawPath string, source string) []Signal {
	if line == "" || strings.HasPrefix(line, "[tailchase]") {
		return nil
	}

	if match := annotationPattern.FindStringSubmatch(line); match != nil {
		file, lineNumber := parseAnnotationProperties(match[1])
		return []Signal{newSignal("github_annotation", source, job, strings.TrimSpace(match[2]), file, lineNumber, "high", line, rawPath)}
	}
	if match := fileLinePattern.FindStringSubmatch(line); match != nil {
		lineNumber, _ := strconv.Atoi(match[2])
		return []Signal{newSignal("file_error", source, job, strings.TrimSpace(match[3]), match[1], lineNumber, "high", line, rawPath)}
	}
	if match := failPattern.FindStringSubmatch(line); match != nil {
		return []Signal{newSignal("test_failure", source, job, "failing test: "+match[1], "", 0, "high", line, rawPath)}
	}
	if strings.Contains(strings.ToLower(line), "panic:") {
		return []Signal{newSignal("runtime_panic", source, job, line, "", 0, "high", line, rawPath)}
	}
	if envName := missingEnvName(line); envName != "" {
		return []Signal{newSignal("missing_environment", source, job, line, "", 0, "high", line, rawPath)}
	}
	if looksLikeGenericFailure(line) {
		return []Signal{newSignal("generic_failure", source, job, line, "", 0, "medium", line, rawPath)}
	}
	return nil
}

func newSignal(signalType string, source string, job jobContext, message string, file string, line int, confidence string, rawExcerpt string, rawPath string) Signal {
	return Signal{
		Type:           signalType,
		Source:         source,
		Job:            job.Name,
		Message:        strings.TrimSpace(message),
		File:           strings.TrimSpace(file),
		Line:           line,
		Confidence:     confidence,
		RawExcerpt:     strings.TrimSpace(rawExcerpt),
		RawExcerptPath: rawPath,
	}
}

func parseAnnotationProperties(props string) (string, int) {
	var file string
	var lineNumber int
	for _, part := range strings.Split(props, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "file":
			file = value
		case "line":
			lineNumber, _ = strconv.Atoi(value)
		}
	}
	return file, lineNumber
}

func cleanLogLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "##[error]")
	line = strings.TrimSpace(timestampPattern.ReplaceAllString(line, ""))
	return line
}

func missingEnvName(line string) string {
	match := missingEnvPattern.FindStringSubmatch(line)
	if match == nil {
		return ""
	}
	if match[1] != "" {
		return match[1]
	}
	return match[2]
}

func looksLikeGenericFailure(line string) bool {
	lower := strings.ToLower(line)
	if strings.HasPrefix(line, "--- tailchase-") {
		return false
	}
	return strings.Contains(lower, "error:") ||
		strings.Contains(lower, "fatal:") ||
		strings.Contains(lower, "exit status") ||
		strings.Contains(lower, " failed") ||
		strings.HasPrefix(lower, "failed ") ||
		strings.HasPrefix(lower, "fail ") ||
		strings.HasPrefix(lower, "fail\t") ||
		strings.Contains(lower, "exception")
}

func signalKey(signal Signal) string {
	return strings.ToLower(strings.Join([]string{
		signal.Type,
		signal.Source,
		signal.Job,
		signal.File,
		strconv.Itoa(signal.Line),
		signal.Message,
	}, "\x00"))
}

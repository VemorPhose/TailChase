package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

type Generator struct {
	Template string
}

type templateData struct {
	Bundle        bundle.FailureBundle
	Delta         deltaContext
	NextActions   []string
	Commands      []string
	StopCondition string
}

func (g Generator) Generate(failureBundle bundle.FailureBundle, opts Options) (Result, error) {
	tmplText := g.Template
	if tmplText == "" {
		if opts.Delta {
			tmplText = defaultDeltaRepairPromptTemplate
		} else {
			tmplText = defaultRepairPromptTemplate
		}
	}

	tmpl, err := template.New("repair_prompt.md").Funcs(template.FuncMap{
		"fallback":      fallback,
		"signalSummary": signalSummary,
		"excerpt":       excerpt,
	}).Parse(tmplText)
	if err != nil {
		return Result{}, err
	}

	data := templateData{
		Bundle:        failureBundle,
		Delta:         buildDeltaContext(failureBundle, opts.AttemptHistory),
		NextActions:   nextActions(failureBundle),
		Commands:      commandsToRun(failureBundle),
		StopCondition: stopCondition(failureBundle),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return Result{}, err
	}

	content, truncated := applySizeLimit(strings.TrimSpace(buf.String())+"\n", opts.SizeLimit)
	return Result{Content: content, Truncated: truncated}, nil
}

type deltaContext struct {
	HasHistory              bool
	AttemptCount            int
	LatestAttempt           project.Attempt
	SameRootErrorSeenBefore bool
	MatchingAttemptNumbers  string
	NewRootCandidates       []bundle.Signal
	RepeatedRootCandidates  []bundle.Signal
}

func buildDeltaContext(failureBundle bundle.FailureBundle, history project.AttemptHistory) deltaContext {
	context := deltaContext{
		HasHistory:              len(history.Attempts) > 0,
		AttemptCount:            len(history.Attempts),
		SameRootErrorSeenBefore: failureBundle.AttemptContext.SameRootErrorSeenBefore,
		MatchingAttemptNumbers:  joinInts(failureBundle.AttemptContext.MatchingAttemptNumbers),
	}
	if context.HasHistory {
		context.LatestAttempt = history.Attempts[len(history.Attempts)-1]
	}

	prior := map[string][]int{}
	for _, attempt := range history.Attempts {
		for _, candidate := range attempt.RootErrorCandidates {
			fingerprint := bundle.RootErrorFingerprint(candidate)
			if fingerprint != "" {
				prior[fingerprint] = append(prior[fingerprint], attempt.Number)
			}
		}
	}
	matchingAttempts := map[int]bool{}
	for _, signal := range failureBundle.RootErrorCandidates {
		if numbers := prior[bundle.RootErrorFingerprint(signal.Message)]; len(numbers) > 0 {
			for _, number := range numbers {
				matchingAttempts[number] = true
			}
			context.RepeatedRootCandidates = append(context.RepeatedRootCandidates, signal)
			continue
		}
		context.NewRootCandidates = append(context.NewRootCandidates, signal)
	}
	if len(context.RepeatedRootCandidates) > 0 {
		context.SameRootErrorSeenBefore = true
	}
	if context.MatchingAttemptNumbers == "" && len(matchingAttempts) > 0 {
		context.MatchingAttemptNumbers = joinInts(sortedAttemptNumbers(matchingAttempts))
	}
	return context
}

func WriteRepairPrompt(run project.Run, result Result) error {
	if err := run.WriteArtifactFile(project.RepairPromptName, project.ArtifactRepairPrompt, "repair_prompt", []byte(result.Content)); err != nil {
		return err
	}
	if result.ModelMetadata == nil {
		return nil
	}
	if result.ModelMetadata.Version == 0 {
		result.ModelMetadata.Version = project.SchemaVersion
	}
	data, err := yaml.Marshal(result.ModelMetadata)
	if err != nil {
		return err
	}
	return run.WriteArtifactFile(project.ModelMetadataName, project.ArtifactModelMetadata, "model_metadata", data)
}

func applySizeLimit(content string, sizeLimit int) (string, bool) {
	if sizeLimit <= 0 || len(content) <= sizeLimit {
		return content, false
	}
	content = content[:sizeLimit]
	content = strings.TrimSpace(content) + "\n\n[tailchase] Prompt truncated to configured size limit.\n"
	return content, true
}

func LoadTemplateFromFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func nextActions(failureBundle bundle.FailureBundle) []string {
	var actions []string
	for _, signal := range failureBundle.RootErrorCandidates {
		if len(actions) >= 4 {
			break
		}
		location := signalLocation(signal)
		if location != "" {
			actions = append(actions, fmt.Sprintf("Start at %s and fix: %s.", location, signal.Message))
			continue
		}
		if signal.Job != "" {
			actions = append(actions, fmt.Sprintf("Inspect the %q job around: %s.", signal.Job, signal.Message))
			continue
		}
		actions = append(actions, "Inspect the earliest high-confidence failure signal before editing unrelated code.")
	}
	if len(actions) == 0 {
		actions = append(actions, "Open the raw GitHub Actions log and identify the first meaningful error before editing.")
	}
	actions = append(actions, "Make the smallest focused change that satisfies the original goal contract.")
	actions = append(actions, "Re-run the listed commands and compare any new failure against the same goal/non-goal boundaries.")
	return actions
}

func commandsToRun(failureBundle bundle.FailureBundle) []string {
	seen := map[string]bool{}
	var commands []string
	add := func(command string) {
		if command == "" || seen[command] {
			return
		}
		seen[command] = true
		commands = append(commands, command)
	}

	for _, condition := range failureBundle.Goal.DoneConditions {
		lower := strings.ToLower(condition)
		switch {
		case strings.Contains(lower, "go test"):
			add("go test ./...")
		case strings.Contains(lower, "npm test"):
			add("npm test")
		case strings.Contains(lower, "npm run build"):
			add("npm run build")
		case strings.Contains(lower, "pytest"):
			add("pytest")
		case strings.Contains(lower, "cargo test"):
			add("cargo test")
		}
	}

	for _, signal := range append(failureBundle.RootErrorCandidates, failureBundle.DownstreamSymptoms...) {
		switch strings.ToLower(filepath.Ext(signal.File)) {
		case ".go":
			add("go test ./...")
		case ".ts", ".tsx", ".js", ".jsx":
			add("npm test")
			add("npm run build")
		case ".py":
			add("pytest")
		case ".rs":
			add("cargo test")
		}
	}

	if len(commands) == 0 {
		add("run the failing GitHub Actions job or its closest local equivalent")
	}
	return commands
}

func stopCondition(failureBundle bundle.FailureBundle) string {
	if len(failureBundle.Goal.StopRules) > 0 {
		return "Stop and ask for human guidance if any stop rule applies: " + strings.Join(failureBundle.Goal.StopRules, "; ")
	}
	if len(failureBundle.Goal.NonGoals) > 0 {
		return "Stop and ask for human guidance if the apparent fix requires violating a non-goal, weakening tests, or changing behavior outside the original task."
	}
	return "Stop and ask for human guidance if the apparent fix requires weakening tests or changing behavior outside the original task."
}

func signalSummary(signal bundle.Signal) string {
	parts := []string{fmt.Sprintf("[%s] %s", signal.Confidence, signal.Message)}
	if location := signalLocation(signal); location != "" {
		parts = append(parts, "at "+location)
	}
	if signal.Job != "" {
		parts = append(parts, "in job "+strconvQuote(signal.Job))
	}
	return strings.Join(parts, " ")
}

func signalLocation(signal bundle.Signal) string {
	if signal.File == "" {
		return ""
	}
	if signal.Line > 0 {
		return fmt.Sprintf("%s:%d", signal.File, signal.Line)
	}
	return signal.File
}

func excerpt(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "`", "'")
	if len(value) <= 220 {
		return value
	}
	return strings.TrimSpace(value[:220]) + "..."
}

func fallback(value string, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func strconvQuote(value string) string {
	return fmt.Sprintf("%q", value)
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func sortedAttemptNumbers(values map[int]bool) []int {
	numbers := make([]int, 0, len(values))
	for value := range values {
		numbers = append(numbers, value)
	}
	sort.Ints(numbers)
	return numbers
}

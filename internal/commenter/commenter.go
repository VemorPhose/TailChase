package commenter

import (
	"bytes"
	"strconv"
	"strings"
	"text/template"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

const maxListedItems = 5

type BodyOptions struct {
	Run          project.Run
	Bundle       bundle.FailureBundle
	RepairPrompt string
}

type bodyData struct {
	Goal           string
	RunID          string
	PromptPath     string
	BundlePath     string
	RootSignals    []string
	SafetyFindings []bundle.SafetyFinding
	Budget         bundle.BudgetMetadata
	Artifacts      []string
	PromptSummary  string
}

func BuildBody(opts BodyOptions) (string, error) {
	repairPrompt := strings.TrimSpace(opts.RepairPrompt)
	data := bodyData{
		Goal:           opts.Bundle.Goal.Goal,
		RunID:          opts.Run.ID,
		PromptPath:     opts.Run.RelativePath(opts.Run.ArtifactPath(project.RepairPromptName)),
		BundlePath:     opts.Run.RelativePath(opts.Run.ArtifactPath(project.FailureBundleName)),
		RootSignals:    signalSummaries(opts.Bundle.RootErrorCandidates),
		SafetyFindings: firstSafetyFindings(opts.Bundle.SafetyFindings),
		Budget:         opts.Bundle.Budget,
		Artifacts:      artifactPaths(opts.Bundle),
		PromptSummary:  firstParagraph(repairPrompt),
	}
	if data.Goal == "" {
		data.Goal = "(goal not specified)"
	}
	if data.PromptSummary == "" {
		data.PromptSummary = "(repair prompt is empty)"
	}

	var buf bytes.Buffer
	if err := commentTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func signalSummaries(signals []bundle.Signal) []string {
	limit := min(len(signals), maxListedItems)
	out := make([]string, 0, limit)
	for _, signal := range signals[:limit] {
		message := strings.TrimSpace(signal.Message)
		if message == "" {
			continue
		}
		location := strings.TrimSpace(signal.File)
		if signal.Line > 0 && signal.File != "" {
			location = signal.File + ":" + strconv.Itoa(signal.Line)
		}
		if location != "" {
			out = append(out, message+" ("+location+")")
			continue
		}
		out = append(out, message)
	}
	return out
}

func firstSafetyFindings(findings []bundle.SafetyFinding) []bundle.SafetyFinding {
	if len(findings) <= maxListedItems {
		return findings
	}
	return findings[:maxListedItems]
}

func artifactPaths(failureBundle bundle.FailureBundle) []string {
	seen := map[string]bool{}
	var paths []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}
	for _, source := range failureBundle.Sources {
		add(source.Path)
	}
	for _, artifact := range failureBundle.Artifacts {
		add(artifact.Path)
	}
	for _, signal := range failureBundle.RootErrorCandidates {
		add(signal.RawExcerptPath)
	}
	for _, signal := range failureBundle.DownstreamSymptoms {
		add(signal.RawExcerptPath)
	}
	if len(paths) > maxListedItems {
		return paths[:maxListedItems]
	}
	return paths
}

func firstParagraph(text string) string {
	for _, block := range strings.Split(text, "\n\n") {
		block = strings.TrimSpace(block)
		if block != "" {
			return block
		}
	}
	return ""
}

var commentTemplate = template.Must(template.New("github_pr_comment.md").Parse(`## Tailchase Repair Context

Goal: {{ .Goal }}
Run: {{ .RunID }}

Repair prompt: ` + "`{{ .PromptPath }}`" + `
Failure bundle: ` + "`{{ .BundlePath }}`" + `

{{- if .RootSignals }}

Likely root signals:
{{- range .RootSignals }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .SafetyFindings }}

Safety findings:
{{- range .SafetyFindings }}
- {{ .Decision }}: {{ .Rule }} - {{ .Message }}{{ if .Path }} (` + "`{{ .Path }}`" + `){{ end }}
{{- end }}
{{- end }}

Context budget: raw {{ .Budget.RawEvidenceBytes }} bytes, included {{ .Budget.IncludedExcerptBytes }} bytes, collapsed repeated blocks {{ .Budget.RepeatedBlocksCollapsed }}.

{{- if .Artifacts }}

Artifact references:
{{- range .Artifacts }}
- ` + "`{{ . }}`" + `
{{- end }}
{{- end }}

Raw logs are intentionally omitted from this comment; use the local artifact links above for full evidence.

Prompt summary:
{{ .PromptSummary }}
`))

package exporter

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

const (
	TargetCodex      = "codex"
	TargetClaudeCode = "claude-code"
	TargetCopilot    = "copilot"
)

type Result struct {
	Target string
	Path   string
}

type targetSpec struct {
	Target      string
	FileName    string
	Title       string
	Instruction string
}

type renderData struct {
	Spec             targetSpec
	RunID            string
	RepairPromptPath string
	BundlePath       string
	Sources          []string
	SafetyFindings   []bundle.SafetyFinding
	RepairPrompt     string
}

var targetSpecs = map[string]targetSpec{
	TargetCodex: {
		Target:      TargetCodex,
		FileName:    "codex-prompt.md",
		Title:       "Codex Repair Context",
		Instruction: "Use this file as the next Codex task prompt. Keep the work scoped to the Tailchase repair prompt and stop if any safety finding says stop.",
	},
	TargetClaudeCode: {
		Target:      TargetClaudeCode,
		FileName:    "claude-code-prompt.md",
		Title:       "Claude Code Repair Context",
		Instruction: "Paste this into Claude Code as repair context. Preserve the listed goal, non-goals, artifact links, and safety findings.",
	},
	TargetCopilot: {
		Target:      TargetCopilot,
		FileName:    "copilot-instructions.md",
		Title:       "GitHub Copilot Repair Context",
		Instruction: "Use this in Copilot Chat or agent mode as a focused repair brief. Reference the local artifacts instead of pasting raw logs.",
	},
}

func Targets() []string {
	targets := make([]string, 0, len(targetSpecs))
	for target := range targetSpecs {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func Write(run project.Run, target string, failureBundle bundle.FailureBundle, repairPrompt string) (Result, error) {
	spec, err := lookupTarget(target)
	if err != nil {
		return Result{}, err
	}
	content, err := Render(run, spec.Target, failureBundle, repairPrompt)
	if err != nil {
		return Result{}, err
	}

	fileName := filepath.Join(project.ExportsDirName, spec.FileName)
	if err := run.WriteArtifactFile(fileName, exportArtifactName(spec.Target), project.ArtifactTargetExport, []byte(content)); err != nil {
		return Result{}, err
	}
	return Result{
		Target: spec.Target,
		Path:   run.RelativePath(run.ArtifactPath(fileName)),
	}, nil
}

func Render(run project.Run, target string, failureBundle bundle.FailureBundle, repairPrompt string) (string, error) {
	spec, err := lookupTarget(target)
	if err != nil {
		return "", err
	}
	repairPrompt = strings.TrimSpace(repairPrompt)
	if repairPrompt == "" {
		return "", fmt.Errorf("repair prompt is empty")
	}

	data := renderData{
		Spec:             spec,
		RunID:            run.ID,
		RepairPromptPath: run.RelativePath(run.ArtifactPath(project.RepairPromptName)),
		BundlePath:       run.RelativePath(run.ArtifactPath(project.FailureBundleName)),
		Sources:          exportSources(failureBundle),
		SafetyFindings:   failureBundle.SafetyFindings,
		RepairPrompt:     repairPrompt,
	}

	var buf bytes.Buffer
	if err := exportTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func lookupTarget(target string) (targetSpec, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	spec, ok := targetSpecs[target]
	if !ok {
		return targetSpec{}, fmt.Errorf("unsupported export target %q; supported targets: %s", target, strings.Join(Targets(), ", "))
	}
	return spec, nil
}

func exportArtifactName(target string) string {
	return strings.ReplaceAll(target, "-", "_") + "_export"
}

func exportSources(failureBundle bundle.FailureBundle) []string {
	seen := map[string]bool{}
	var sources []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		sources = append(sources, path)
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
	sort.Strings(sources)
	return sources
}

var exportTemplate = template.Must(template.New("target_export.md").Parse(`# {{ .Spec.Title }}

{{ .Spec.Instruction }}

Run ID: {{ .RunID }}

## Source Artifacts

- Repair prompt: {{ .RepairPromptPath }}
- Failure bundle: {{ .BundlePath }}
{{- range .Sources }}
- Raw evidence: {{ . }}
{{- end }}

{{- if .SafetyFindings }}

## Safety Findings
{{- range .SafetyFindings }}

- {{ .Decision }}: {{ .Rule }} - {{ .Message }}{{ if .Path }} ({{ .Path }}){{ end }}
{{- end }}
{{- end }}

## Repair Prompt

{{ .RepairPrompt }}
`))

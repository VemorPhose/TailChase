package prompt

const defaultRepairPromptTemplate = `# Repair Prompt

You are continuing a coding task after GitHub Actions failed. Use the goal contract and CI evidence below. Keep the repair focused on the original goal, preserve listed behavior, and do not weaken tests to make CI pass.

## Original Goal

{{ .Bundle.Goal.Goal }}

{{- if .Bundle.Goal.NonGoals }}

## Non-Goals
{{- range .Bundle.Goal.NonGoals }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.MustPreserve }}

## Must Preserve
{{- range .Bundle.Goal.MustPreserve }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.DoneConditions }}

## Done Conditions
{{- range .Bundle.Goal.DoneConditions }}
- {{ . }}
{{- end }}
{{- end }}

## CI Evidence Summary

- Repository: {{ fallback .Bundle.Run.Repository "unknown" }}
- Run ID: {{ fallback .Bundle.Run.RunID "unknown" }}
- Source: {{ fallback .Bundle.Run.Source "github_actions" }}

## Context Budget

- Raw evidence bytes: {{ .Bundle.Budget.RawEvidenceBytes }}
- Included excerpt bytes: {{ .Bundle.Budget.IncludedExcerptBytes }}
- Repeated blocks collapsed: {{ .Bundle.Budget.RepeatedBlocksCollapsed }}
- Estimated prompt bytes: {{ .Bundle.Budget.EstimatedPromptBytes }}

{{- if .Bundle.RootErrorCandidates }}

## Likely Root Cause Candidates
{{- range .Bundle.RootErrorCandidates }}
- {{ signalSummary . }}
  - Evidence: ` + "`{{ excerpt .RawExcerpt }}`" + `
  - Raw log: {{ .RawExcerptPath }}
{{- end }}
{{- else }}

## Likely Root Cause Candidates

- Tailchase did not extract a clear root-error candidate. Inspect the raw log artifact before editing.
{{- end }}

{{- if .Bundle.DownstreamSymptoms }}

## Downstream Symptoms
{{- range .Bundle.DownstreamSymptoms }}
- {{ signalSummary . }}
{{- end }}
{{- end }}

## Next Actions
{{- range .NextActions }}
- {{ . }}
{{- end }}

## Commands To Run
{{- range .Commands }}
- ` + "`{{ . }}`" + `
{{- end }}

{{- if .Bundle.Warnings }}

## Warnings
{{- range .Bundle.Warnings }}
- {{ . }}
{{- end }}
{{- end }}

## Stop Condition

{{ .StopCondition }}

## Local Artifacts
{{- range .Bundle.Artifacts }}
- {{ .Name }}: {{ .Path }}
{{- end }}
`

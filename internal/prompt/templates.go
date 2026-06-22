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

{{- if .Bundle.Goal.ExpectedPaths }}

## Expected Paths
{{- range .Bundle.Goal.ExpectedPaths }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.SuspiciousPaths }}

## Suspicious Paths
{{- range .Bundle.Goal.SuspiciousPaths }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.StopRules }}

## Stop Rules
{{- range .Bundle.Goal.StopRules }}
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

{{- if .Bundle.SafetyFindings }}

## Safety Findings
{{- range .Bundle.SafetyFindings }}
- [{{ .Decision }}] {{ .Rule }}: {{ .Message }}
{{- end }}
{{- end }}

## Stop Condition

{{ .StopCondition }}

## Local Artifacts
{{- range .Bundle.Artifacts }}
- {{ .Name }}: {{ .Path }}
{{- end }}

{{- if .Bundle.Sources }}

## Evidence Sources
{{- range .Bundle.Sources }}
- {{ .Source }}: {{ .Path }}
{{- end }}
{{- end }}
`

const defaultDeltaRepairPromptTemplate = `# Delta Repair Prompt

You are continuing a coding task after at least one repair attempt may already have happened. Focus on what changed, avoid re-reading repeated context unless needed, and keep the repair inside the original goal contract.

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

{{- if .Bundle.Goal.ExpectedPaths }}

## Expected Paths
{{- range .Bundle.Goal.ExpectedPaths }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.SuspiciousPaths }}

## Suspicious Paths
{{- range .Bundle.Goal.SuspiciousPaths }}
- {{ . }}
{{- end }}
{{- end }}

{{- if .Bundle.Goal.StopRules }}

## Stop Rules
{{- range .Bundle.Goal.StopRules }}
- {{ . }}
{{- end }}
{{- end }}

## Attempt Memory

{{- if .Delta.HasHistory }}
- Prior attempts recorded: {{ .Delta.AttemptCount }}
- Latest prior attempt: #{{ .Delta.LatestAttempt.Number }} ({{ .Delta.LatestAttempt.Outcome }})
- Previous bundle: {{ .Delta.LatestAttempt.BundlePath }}
- Previous prompt: {{ .Delta.LatestAttempt.PromptPath }}
{{- else }}
- No prior attempts recorded; use this as a normal repair prompt with delta-ready structure.
{{- end }}
{{- if .Delta.SameRootErrorSeenBefore }}
- Same root error seen before: yes{{ if .Delta.MatchingAttemptNumbers }} (attempts: {{ .Delta.MatchingAttemptNumbers }}){{ end }}
{{- else }}
- Same root error seen before: no
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

{{- if .Delta.NewRootCandidates }}

## New Root Evidence
{{- range .Delta.NewRootCandidates }}
- {{ signalSummary . }}
  - Evidence: ` + "`{{ excerpt .RawExcerpt }}`" + `
  - Raw log: {{ .RawExcerptPath }}
{{- end }}
{{- else }}

## New Root Evidence

- No new root-error candidate was identified.
{{- end }}

{{- if .Delta.RepeatedRootCandidates }}

## Repeated Root Evidence
{{- range .Delta.RepeatedRootCandidates }}
- {{ signalSummary . }}
  - Evidence excerpt omitted because this root error already appeared in prior attempts.
  - Raw log: {{ .RawExcerptPath }}
{{- end }}
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

{{- if .Bundle.SafetyFindings }}

## Safety Findings
{{- range .Bundle.SafetyFindings }}
- [{{ .Decision }}] {{ .Rule }}: {{ .Message }}
{{- end }}
{{- end }}

## Stop Condition

{{ .StopCondition }}

## Local Artifacts
{{- range .Bundle.Artifacts }}
- {{ .Name }}: {{ .Path }}
{{- end }}

{{- if .Bundle.Sources }}

## Evidence Sources
{{- range .Bundle.Sources }}
- {{ .Source }}: {{ .Path }}
{{- end }}
{{- end }}
`

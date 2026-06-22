package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestNormalizerExtractsSignals(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.EvidencePath(project.GitHubActionsLogName), `# Tailchase GitHub Actions evidence
--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="" ---
2026-06-20T10:00:00Z ::error file=internal/app/app.go,line=42::undefined: Handler
internal/app/app.go:42:10: undefined: Handler
--- FAIL: TestHandler
panic: missing required environment variable API_TOKEN
--- tailchase-end-job id=11 ---
`)

	normalized, err := (bundlepkg.Normalizer{
		Now: func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) },
	}).NormalizeRun(run)
	if err != nil {
		t.Fatalf("NormalizeRun() error = %v", err)
	}

	if len(normalized.Signals) != 4 {
		t.Fatalf("signals = %d, want 4: %#v", len(normalized.Signals), normalized.Signals)
	}
	if normalized.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", normalized.Version, bundlepkg.SchemaVersion)
	}
	if normalized.Signals[0].Type != "github_annotation" {
		t.Fatalf("first signal type = %q, want github_annotation", normalized.Signals[0].Type)
	}
	if normalized.Signals[0].File != "internal/app/app.go" || normalized.Signals[0].Line != 42 {
		t.Fatalf("annotation location = %s:%d, want internal/app/app.go:42", normalized.Signals[0].File, normalized.Signals[0].Line)
	}
	if normalized.Signals[0].Job != "unit tests" {
		t.Fatalf("job = %q, want unit tests", normalized.Signals[0].Job)
	}
}

func TestWriteAndReadNormalizedEvidence(t *testing.T) {
	run := mustRun(t)
	normalized := bundlepkg.NormalizedEvidence{
		GeneratedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		Signals: []bundlepkg.Signal{
			{Type: "generic_failure", Source: "github_actions", Message: "build failed", Confidence: "medium"},
		},
	}

	if err := bundlepkg.WriteNormalizedEvidence(run, normalized); err != nil {
		t.Fatalf("WriteNormalizedEvidence() error = %v", err)
	}
	got, err := bundlepkg.ReadNormalizedEvidence(run)
	if err != nil {
		t.Fatalf("ReadNormalizedEvidence() error = %v", err)
	}
	if got.Signals[0].Message != "build failed" {
		t.Fatalf("message = %q, want build failed", got.Signals[0].Message)
	}
	if got.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", got.Version, bundlepkg.SchemaVersion)
	}

	data, err := os.ReadFile(run.ArtifactPath(project.NormalizedEvidenceName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("normalized YAML missing version: %s", string(data))
	}
	if !strings.Contains(string(data), "generic_failure") {
		t.Fatalf("normalized YAML did not contain signal: %s", string(data))
	}
}

func TestReadNormalizedEvidenceDefaultsMissingVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.ArtifactPath(project.NormalizedEvidenceName), `generated_at: 2026-06-20T12:00:00Z
signals:
  - type: generic_failure
    source: github_actions
    message: build failed
    confidence: medium
`)

	got, err := bundlepkg.ReadNormalizedEvidence(run)
	if err != nil {
		t.Fatalf("ReadNormalizedEvidence() error = %v", err)
	}
	if got.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", got.Version, bundlepkg.SchemaVersion)
	}
}

func TestReadNormalizedEvidenceRejectsUnsupportedVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.ArtifactPath(project.NormalizedEvidenceName), "version: 99\n")

	if _, err := bundlepkg.ReadNormalizedEvidence(run); err == nil {
		t.Fatal("ReadNormalizedEvidence() error = nil, want unsupported version error")
	}
}

func TestCompilerBuildsFailureBundle(t *testing.T) {
	run := mustRun(t)
	normalized := bundlepkg.NormalizedEvidence{
		Version: 1,
		Run: bundlepkg.RunMetadata{
			Source:     "github_actions",
			Repository: "owner/repo",
			RunID:      "12345",
		},
		Sources: []bundlepkg.EvidenceSource{{Source: "github_actions", Path: ".tailchase/runs/12345/evidence/github-actions.log"}},
		Signals: []bundlepkg.Signal{
			{Type: "file_error", Source: "github_actions", Job: "test", Message: "undefined: Handler", File: "internal/app/app.go", Line: 42, Confidence: "high"},
			{Type: "generic_failure", Source: "github_actions", Job: "test", Message: "exit status 1", Confidence: "medium"},
		},
	}
	goal := project.Goal{
		Goal:            "Fix the handler compile error",
		NonGoals:        []string{"Do not change API routes"},
		DoneConditions:  []string{"go test ./... passes"},
		SuspiciousPaths: []string{"internal/app"},
	}

	got, err := (bundlepkg.Compiler{
		Now: func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) },
	}).Compile(run, goal, normalized)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if got.Run.Repository != "owner/repo" {
		t.Fatalf("repository = %q, want owner/repo", got.Run.Repository)
	}
	if got.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", got.Version, bundlepkg.SchemaVersion)
	}
	if len(got.RootErrorCandidates) != 1 {
		t.Fatalf("root candidates = %d, want 1", len(got.RootErrorCandidates))
	}
	if len(got.DownstreamSymptoms) != 1 {
		t.Fatalf("downstream symptoms = %d, want 1", len(got.DownstreamSymptoms))
	}
	if !hasSubstring(got.Warnings, "suspicious path") {
		t.Fatalf("warnings = %#v, want suspicious path warning", got.Warnings)
	}
}

func TestCompilerDetectsRepeatedRootFailure(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		current  string
	}{
		{
			name:     "exact",
			previous: "undefined: Handler",
			current:  "undefined: Handler",
		},
		{
			name:     "near identical",
			previous: "internal/app/app.go:41:5: undefined: Handler",
			current:  "internal/app/app.go:88:2: undefined: Handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := mustRun(t)
			if _, err := run.AppendAttempt(project.Attempt{
				RootErrorCandidates: []string{tt.previous},
				Outcome:             "failed",
			}); err != nil {
				t.Fatalf("AppendAttempt() error = %v", err)
			}

			normalized := bundlepkg.NormalizedEvidence{
				Version: 1,
				Signals: []bundlepkg.Signal{
					{Type: "file_error", Source: "github_actions", Message: tt.current, Confidence: "high"},
				},
			}
			got, err := (bundlepkg.Compiler{}).Compile(run, project.Goal{
				Goal:           "Fix the compile error",
				NonGoals:       []string{"Do not weaken tests"},
				DoneConditions: []string{"go test ./... passes"},
			}, normalized)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			if !got.AttemptContext.SameRootErrorSeenBefore {
				t.Fatalf("same root flag = false, want true")
			}
			if len(got.AttemptContext.MatchingAttemptNumbers) != 1 || got.AttemptContext.MatchingAttemptNumbers[0] != 1 {
				t.Fatalf("matching attempts = %#v, want [1]", got.AttemptContext.MatchingAttemptNumbers)
			}
			if !hasSubstring(got.Warnings, "same root error seen before") {
				t.Fatalf("warnings = %#v, want repeated root warning", got.Warnings)
			}
		})
	}
}

func TestCompilerIgnoresRepeatedDownstreamSymptoms(t *testing.T) {
	run := mustRun(t)
	if _, err := run.AppendAttempt(project.Attempt{
		RootErrorCandidates: []string{"exit status 1"},
		Outcome:             "failed",
	}); err != nil {
		t.Fatalf("AppendAttempt() error = %v", err)
	}

	normalized := bundlepkg.NormalizedEvidence{
		Version: 1,
		Signals: []bundlepkg.Signal{
			{Type: "file_error", Source: "github_actions", Message: "undefined: Handler", Confidence: "high"},
			{Type: "generic_failure", Source: "github_actions", Message: "exit status 1", Confidence: "medium"},
		},
	}
	got, err := (bundlepkg.Compiler{}).Compile(run, project.Goal{
		Goal:           "Fix the compile error",
		NonGoals:       []string{"Do not weaken tests"},
		DoneConditions: []string{"go test ./... passes"},
	}, normalized)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if got.AttemptContext.SameRootErrorSeenBefore {
		t.Fatalf("same root flag = true, want false")
	}
	if hasSubstring(got.Warnings, "same root error seen before") {
		t.Fatalf("warnings = %#v, did not want repeated root warning", got.Warnings)
	}
}

func TestCompilerRecordsBudgetAndCollapsesRepeatedExcerpts(t *testing.T) {
	run := mustRun(t)
	rawLog := "Run go test ./...\n" + strings.Repeat("panic: boom\ninternal/app/app.go:42\ncreated by test\n", 6)
	writeFile(t, run.EvidencePath(project.GitHubActionsLogName), rawLog)

	repeatedExcerpt := strings.Repeat("panic: boom\ninternal/app/app.go:42\ncreated by test\n", 6)
	normalized := bundlepkg.NormalizedEvidence{
		Version: 1,
		Sources: []bundlepkg.EvidenceSource{
			{Source: "github_actions", Path: run.RelativePath(run.EvidencePath(project.GitHubActionsLogName))},
			{Source: "github_actions", Path: run.RelativePath(run.EvidencePath(project.GitHubActionsLogName)), Job: "unit"},
		},
		Signals: []bundlepkg.Signal{
			{
				Type:           "runtime_panic",
				Source:         "github_actions",
				Message:        "panic: boom",
				Confidence:     "high",
				RawExcerpt:     repeatedExcerpt,
				RawExcerptPath: run.RelativePath(run.EvidencePath(project.GitHubActionsLogName)),
			},
		},
	}
	got, err := (bundlepkg.Compiler{}).Compile(run, project.Goal{
		Goal:           "Fix the panic",
		NonGoals:       []string{"Do not weaken tests"},
		DoneConditions: []string{"go test ./... passes"},
	}, normalized)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if got.Budget.RawEvidenceBytes != int64(len(rawLog)) {
		t.Fatalf("raw evidence bytes = %d, want %d", got.Budget.RawEvidenceBytes, len(rawLog))
	}
	if got.Budget.IncludedExcerptBytes != int64(len(got.RootErrorCandidates[0].RawExcerpt)) {
		t.Fatalf("included excerpt bytes = %d, want compacted excerpt length %d", got.Budget.IncludedExcerptBytes, len(got.RootErrorCandidates[0].RawExcerpt))
	}
	if got.Budget.RepeatedBlocksCollapsed != 5 {
		t.Fatalf("collapsed blocks = %d, want 5", got.Budget.RepeatedBlocksCollapsed)
	}
	if got.Budget.EstimatedPromptBytes <= got.Budget.IncludedExcerptBytes {
		t.Fatalf("estimated prompt bytes = %d, want more than included excerpts %d", got.Budget.EstimatedPromptBytes, got.Budget.IncludedExcerptBytes)
	}
	if len(got.RootErrorCandidates[0].RawExcerpt) >= len(repeatedExcerpt) {
		t.Fatalf("excerpt was not compacted:\n%s", got.RootErrorCandidates[0].RawExcerpt)
	}
	if !strings.Contains(got.RootErrorCandidates[0].RawExcerpt, "repeated previous 3-line block 5 more time(s)") {
		t.Fatalf("excerpt missing collapse marker:\n%s", got.RootErrorCandidates[0].RawExcerpt)
	}
}

func TestWriteAndReadFailureBundle(t *testing.T) {
	run := mustRun(t)
	want := bundlepkg.FailureBundle{
		Version: 1,
		Run:     bundlepkg.RunMetadata{Source: "github_actions", RunID: "12345"},
		Goal:    bundlepkg.GoalContract{Goal: "Fix CI"},
		RootErrorCandidates: []bundlepkg.Signal{
			{Type: "generic_failure", Source: "github_actions", Message: "failed", Confidence: "medium"},
		},
	}

	if err := bundlepkg.WriteFailureBundle(run, want); err != nil {
		t.Fatalf("WriteFailureBundle() error = %v", err)
	}
	got, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		t.Fatalf("ReadFailureBundle() error = %v", err)
	}
	if got.RootErrorCandidates[0].Message != "failed" {
		t.Fatalf("message = %q, want failed", got.RootErrorCandidates[0].Message)
	}
	if got.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", got.Version, bundlepkg.SchemaVersion)
	}

	data, err := os.ReadFile(run.ArtifactPath(project.FailureBundleName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("failure bundle YAML missing version: %s", string(data))
	}
	if !strings.Contains(string(data), "root_error_candidates") {
		t.Fatalf("failure bundle YAML missing candidates: %s", string(data))
	}
}

func TestReadFailureBundleDefaultsMissingVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.ArtifactPath(project.FailureBundleName), `run:
  source: github_actions
goal:
  goal: Fix CI
root_error_candidates:
  - type: generic_failure
    source: github_actions
    message: failed
    confidence: medium
`)

	got, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		t.Fatalf("ReadFailureBundle() error = %v", err)
	}
	if got.Version != bundlepkg.SchemaVersion {
		t.Fatalf("version = %d, want %d", got.Version, bundlepkg.SchemaVersion)
	}
}

func TestReadFailureBundleRejectsUnsupportedVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.ArtifactPath(project.FailureBundleName), "version: 99\n")

	if _, err := bundlepkg.ReadFailureBundle(run); err == nil {
		t.Fatal("ReadFailureBundle() error = nil, want unsupported version error")
	}
}

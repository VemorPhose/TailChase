package bundle

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
)

func TestCompilerBuildsFailureBundle(t *testing.T) {
	run := mustBundleRun(t)
	normalized := NormalizedEvidence{
		Version: schemaVersion,
		Run: RunMetadata{
			Source:     "github_actions",
			Repository: "owner/repo",
			RunID:      "12345",
		},
		Sources: []EvidenceSource{{Source: "github_actions", Path: ".tailchase/runs/12345/evidence/github-actions.log"}},
		Signals: []Signal{
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

	got, err := (Compiler{
		Now: func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) },
	}).Compile(run, goal, normalized)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if got.Run.Repository != "owner/repo" {
		t.Fatalf("repository = %q, want owner/repo", got.Run.Repository)
	}
	if len(got.RootErrorCandidates) != 1 {
		t.Fatalf("root candidates = %d, want 1", len(got.RootErrorCandidates))
	}
	if len(got.DownstreamSymptoms) != 1 {
		t.Fatalf("downstream symptoms = %d, want 1", len(got.DownstreamSymptoms))
	}
	if !containsWarning(got.Warnings, "suspicious path") {
		t.Fatalf("warnings = %#v, want suspicious path warning", got.Warnings)
	}
}

func TestWriteAndReadFailureBundle(t *testing.T) {
	run := mustBundleRun(t)
	want := FailureBundle{
		Version: schemaVersion,
		Run:     RunMetadata{Source: "github_actions", RunID: "12345"},
		Goal:    GoalContract{Goal: "Fix CI"},
		RootErrorCandidates: []Signal{
			{Type: "generic_failure", Source: "github_actions", Message: "failed", Confidence: "medium"},
		},
	}

	if err := WriteFailureBundle(run, want); err != nil {
		t.Fatalf("WriteFailureBundle() error = %v", err)
	}
	got, err := ReadFailureBundle(run)
	if err != nil {
		t.Fatalf("ReadFailureBundle() error = %v", err)
	}
	if got.RootErrorCandidates[0].Message != "failed" {
		t.Fatalf("message = %q, want failed", got.RootErrorCandidates[0].Message)
	}

	data, err := os.ReadFile(run.ArtifactPath(project.FailureBundleName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "root_error_candidates") {
		t.Fatalf("failure bundle YAML missing candidates: %s", string(data))
	}
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

package tests

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	modelpkg "github.com/VemorPhose/TailChase/internal/model"
	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
)

func TestGeneratorRendersRepairPrompt(t *testing.T) {
	failureBundle := bundlepkg.FailureBundle{
		Run: bundlepkg.RunMetadata{Source: "github_actions", Repository: "owner/repo", RunID: "12345"},
		Goal: bundlepkg.GoalContract{
			Goal:           "Fix CI",
			NonGoals:       []string{"Do not weaken tests"},
			DoneConditions: []string{"go test ./... passes"},
			ExpectedPaths:  []string{"internal/app"},
			StopRules:      []string{"Stop before changing tests"},
		},
		Budget: bundlepkg.BudgetMetadata{
			RawEvidenceBytes:        1200,
			IncludedExcerptBytes:    120,
			RepeatedBlocksCollapsed: 3,
			EstimatedPromptBytes:    2400,
		},
		RootErrorCandidates: []bundlepkg.Signal{
			{
				Type:           "file_error",
				Source:         "github_actions",
				Job:            "unit tests",
				Message:        "undefined: Handler",
				File:           "internal/app/app.go",
				Line:           42,
				Confidence:     "high",
				RawExcerpt:     "internal/app/app.go:42: undefined: Handler",
				RawExcerptPath: ".tailchase/runs/12345/evidence/github-actions.log",
			},
		},
		Artifacts: []bundlepkg.Artifact{{Name: "failure_bundle", Path: ".tailchase/runs/12345/failure-bundle.yml"}},
	}

	result, err := (promptpkg.Generator{}).Generate(failureBundle, promptpkg.Options{SizeLimit: 12000})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	for _, want := range []string{"Fix CI", "undefined: Handler", "internal/app/app.go:42", "go test ./...", "Context Budget", "Raw evidence bytes: 1200", "Repeated blocks collapsed: 3", "Expected Paths", "Stop Rules", "Stop before changing tests"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("prompt missing %q:\n%s", want, result.Content)
		}
	}
}

func TestGeneratorRendersDeltaPromptWithPriorAttempt(t *testing.T) {
	repeatedExcerpt := strings.Repeat("internal/app/app.go:42: undefined: Handler\n", 8)
	failureBundle := bundlepkg.FailureBundle{
		Run: bundlepkg.RunMetadata{Source: "github_actions", Repository: "owner/repo", RunID: "12345"},
		Goal: bundlepkg.GoalContract{
			Goal:           "Fix CI",
			NonGoals:       []string{"Do not weaken tests"},
			DoneConditions: []string{"go test ./... passes"},
		},
		AttemptContext: bundlepkg.AttemptContext{SameRootErrorSeenBefore: true, MatchingAttemptNumbers: []int{1}},
		Budget:         bundlepkg.BudgetMetadata{RawEvidenceBytes: 3000, IncludedExcerptBytes: 200, RepeatedBlocksCollapsed: 5, EstimatedPromptBytes: 1800},
		RootErrorCandidates: []bundlepkg.Signal{
			{
				Type:           "file_error",
				Source:         "github_actions",
				Message:        "undefined: Handler",
				File:           "internal/app/app.go",
				Line:           42,
				Confidence:     "high",
				RawExcerpt:     repeatedExcerpt,
				RawExcerptPath: ".tailchase/runs/12345/evidence/github-actions.log",
			},
		},
		Artifacts: []bundlepkg.Artifact{{Name: "failure_bundle", Path: ".tailchase/runs/12345/failure-bundle.yml"}},
	}

	result, err := (promptpkg.Generator{}).Generate(failureBundle, promptpkg.Options{
		SizeLimit: 12000,
		Delta:     true,
		AttemptHistory: project.AttemptHistory{Attempts: []project.Attempt{
			{Number: 1, RunID: "12345", BundlePath: ".tailchase/runs/12345/failure-bundle.yml", PromptPath: ".tailchase/runs/12345/repair-prompt.md", RootErrorCandidates: []string{"undefined: Handler"}, Outcome: "failed"},
		}},
	})
	if err != nil {
		t.Fatalf("Generate(delta) error = %v", err)
	}

	for _, want := range []string{"# Delta Repair Prompt", "Prior attempts recorded: 1", "Latest prior attempt: #1 (failed)", "Same root error seen before: yes (attempts: 1)", "Repeated Root Evidence", "Evidence excerpt omitted", ".tailchase/runs/12345/evidence/github-actions.log", "Stop Condition"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("delta prompt missing %q:\n%s", want, result.Content)
		}
	}
	if strings.Contains(result.Content, repeatedExcerpt) {
		t.Fatalf("delta prompt duplicated repeated excerpt:\n%s", result.Content)
	}
}

func TestGeneratorDeltaPromptFallsBackWithoutHistory(t *testing.T) {
	failureBundle := bundlepkg.FailureBundle{
		Goal: bundlepkg.GoalContract{
			Goal:           "Fix CI",
			NonGoals:       []string{"Do not weaken tests"},
			DoneConditions: []string{"go test ./... passes"},
		},
		RootErrorCandidates: []bundlepkg.Signal{
			{Type: "file_error", Source: "github_actions", Message: "undefined: Handler", Confidence: "high", RawExcerpt: "internal/app/app.go:42: undefined: Handler", RawExcerptPath: ".tailchase/runs/12345/evidence/github-actions.log"},
		},
	}

	result, err := (promptpkg.Generator{}).Generate(failureBundle, promptpkg.Options{SizeLimit: 12000, Delta: true})
	if err != nil {
		t.Fatalf("Generate(delta without history) error = %v", err)
	}

	for _, want := range []string{"No prior attempts recorded", "Same root error seen before: no", "New Root Evidence", "internal/app/app.go:42: undefined: Handler"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("delta fallback prompt missing %q:\n%s", want, result.Content)
		}
	}
}

func TestGeneratorTruncatesToSizeLimit(t *testing.T) {
	failureBundle := bundlepkg.FailureBundle{
		Goal: bundlepkg.GoalContract{Goal: strings.Repeat("long ", 100)},
		RootErrorCandidates: []bundlepkg.Signal{
			{Type: "generic_failure", Source: "github_actions", Message: strings.Repeat("failure ", 100), Confidence: "medium"},
		},
	}

	result, err := (promptpkg.Generator{}).Generate(failureBundle, promptpkg.Options{SizeLimit: 200})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !result.Truncated {
		t.Fatal("result was not marked truncated")
	}
	if !strings.Contains(result.Content, "Prompt truncated") {
		t.Fatalf("prompt missing truncation marker:\n%s", result.Content)
	}
}

func TestWriteRepairPrompt(t *testing.T) {
	run := mustRun(t)

	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}
	data, err := os.ReadFile(run.ArtifactPath(project.RepairPromptName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "# Repair Prompt\n" {
		t.Fatalf("prompt = %q, want # Repair Prompt", string(data))
	}
}

func TestModelGeneratorUsesProviderAndRecordsMetadata(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 30, 0, 0, time.UTC)
	var got modelpkg.Request
	provider := modelpkg.ProviderFunc(func(ctx context.Context, request modelpkg.Request) (modelpkg.Response, error) {
		got = request
		return modelpkg.Response{
			Content: "# Model Repair Prompt\nUse the raw artifact links and stop before weakening tests.",
			Metadata: map[string]string{
				"response_id": "resp_123",
			},
		}, nil
	})
	failureBundle := bundlepkg.FailureBundle{
		Goal: bundlepkg.GoalContract{
			Goal:          "Fix CI",
			NonGoals:      []string{"Do not weaken tests"},
			ExpectedPaths: []string{"internal/app"},
			StopRules:     []string{"Stop before weakening tests"},
		},
		Sources: []bundlepkg.EvidenceSource{
			{Source: "github_actions", Path: ".tailchase/runs/12345/evidence/github-actions.log"},
		},
		Budget: bundlepkg.BudgetMetadata{RawEvidenceBytes: 4000, IncludedExcerptBytes: 500, RepeatedBlocksCollapsed: 2, EstimatedPromptBytes: 1500},
		SafetyFindings: []bundlepkg.SafetyFinding{
			{Rule: "test_weakening", Decision: "stop", Message: "test file edit detected", Path: "internal/app/app_test.go"},
		},
		RootErrorCandidates: []bundlepkg.Signal{
			{Type: "file_error", Source: "github_actions", Message: "undefined: Handler", File: "internal/app/app.go", Line: 42, Confidence: "high", RawExcerptPath: ".tailchase/runs/12345/evidence/github-actions.log"},
		},
		Artifacts: []bundlepkg.Artifact{
			{Name: "failure_bundle", Path: ".tailchase/runs/12345/failure-bundle.yml"},
		},
	}

	result, err := (promptpkg.ModelGenerator{
		Provider: provider,
		Now: func() time.Time {
			return now
		},
	}).Generate(context.Background(), failureBundle, project.ModelConfig{
		Provider: "openai_compatible",
		Model:    "example-model",
	}, promptpkg.Options{
		SizeLimit: 12000,
		Delta:     true,
		AttemptHistory: project.AttemptHistory{Attempts: []project.Attempt{
			{Number: 1, RunID: "12345", BundlePath: ".tailchase/runs/12345/failure-bundle.yml", PromptPath: ".tailchase/runs/12345/repair-prompt.md", RootErrorCandidates: []string{"undefined: Handler"}, Outcome: "failed"},
		}},
	})
	if err != nil {
		t.Fatalf("ModelGenerator.Generate() error = %v", err)
	}

	if got.Model != "example-model" || len(got.Messages) != 2 {
		t.Fatalf("request = %#v, want model and two messages", got)
	}
	requestText := got.Messages[1].Content
	for _, want := range []string{"raw_evidence_links", ".tailchase/runs/12345/evidence/github-actions.log", "safety_findings", "test_weakening", "attempt_history", "delta: true"} {
		if !strings.Contains(requestText, want) {
			t.Fatalf("model request missing %q:\n%s", want, requestText)
		}
	}
	if !strings.Contains(result.Content, "# Model Repair Prompt") {
		t.Fatalf("content = %q, want model prompt", result.Content)
	}
	if result.ModelMetadata == nil {
		t.Fatal("ModelMetadata is nil")
	}
	if result.ModelMetadata.GeneratedAt != now || result.ModelMetadata.Provider != "openai_compatible" || result.ModelMetadata.Model != "example-model" {
		t.Fatalf("metadata = %#v, want provider/model/time", result.ModelMetadata)
	}
	if !result.ModelMetadata.Delta || result.ModelMetadata.ResponseMetadata["response_id"] != "resp_123" {
		t.Fatalf("metadata = %#v, want delta and response metadata", result.ModelMetadata)
	}
}

func TestModelGeneratorFailsSafelyOnProviderError(t *testing.T) {
	provider := modelpkg.ProviderFunc(func(ctx context.Context, request modelpkg.Request) (modelpkg.Response, error) {
		return modelpkg.Response{}, errors.New("provider unavailable")
	})

	_, err := (promptpkg.ModelGenerator{Provider: provider}).Generate(context.Background(), bundlepkg.FailureBundle{}, project.ModelConfig{Model: "example-model"}, promptpkg.Options{SizeLimit: 12000})
	if err == nil {
		t.Fatal("ModelGenerator.Generate() error = nil, want provider error")
	}
	if !strings.Contains(err.Error(), "model prompt generation failed") || !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("error = %v, want safe provider failure", err)
	}
}

func TestWriteRepairPromptWritesModelMetadata(t *testing.T) {
	run := mustRun(t)
	result := promptpkg.Result{
		Content: "# Model Repair Prompt\n",
		ModelMetadata: &promptpkg.ModelMetadata{
			Version:     project.SchemaVersion,
			Provider:    "openai_compatible",
			Model:       "example-model",
			PromptMode:  "model",
			GeneratedAt: time.Date(2026, 6, 22, 10, 30, 0, 0, time.UTC),
			PromptBytes: len("# Model Repair Prompt\n"),
		},
	}

	if err := promptpkg.WriteRepairPrompt(run, result); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}
	data, err := os.ReadFile(run.ArtifactPath(project.ModelMetadataName))
	if err != nil {
		t.Fatalf("ReadFile(model metadata) error = %v", err)
	}
	for _, want := range []string{"version: 1", "provider: openai_compatible", "model: example-model", "prompt_mode: model"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("metadata missing %q:\n%s", want, string(data))
		}
	}
	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if !artifactRecorded(meta.Artifacts, project.ArtifactRepairPrompt) || !artifactRecorded(meta.Artifacts, project.ArtifactModelMetadata) {
		t.Fatalf("artifacts = %#v, want repair prompt and model metadata", meta.Artifacts)
	}
}

func artifactRecorded(artifacts []project.RunArtifact, name string) bool {
	for _, artifact := range artifacts {
		if artifact.Name == name {
			return true
		}
	}
	return false
}

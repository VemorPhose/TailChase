package tests

import (
	"os"
	"strings"
	"testing"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
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

	for _, want := range []string{"Fix CI", "undefined: Handler", "internal/app/app.go:42", "go test ./...", "Context Budget", "Raw evidence bytes: 1200", "Repeated blocks collapsed: 3"} {
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

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

package prompt

import (
	"os"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestGeneratorRendersRepairPrompt(t *testing.T) {
	failureBundle := bundle.FailureBundle{
		Run: bundle.RunMetadata{Source: "github_actions", Repository: "owner/repo", RunID: "12345"},
		Goal: bundle.GoalContract{
			Goal:           "Fix CI",
			NonGoals:       []string{"Do not weaken tests"},
			DoneConditions: []string{"go test ./... passes"},
		},
		RootErrorCandidates: []bundle.Signal{
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
		Artifacts: []bundle.Artifact{{Name: "failure_bundle", Path: ".tailchase/runs/12345/failure-bundle.yml"}},
	}

	result, err := (Generator{}).Generate(failureBundle, Options{SizeLimit: 12000})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	for _, want := range []string{"Fix CI", "undefined: Handler", "internal/app/app.go:42", "go test ./..."} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("prompt missing %q:\n%s", want, result.Content)
		}
	}
}

func TestGeneratorTruncatesToSizeLimit(t *testing.T) {
	failureBundle := bundle.FailureBundle{
		Goal: bundle.GoalContract{Goal: strings.Repeat("long ", 100)},
		RootErrorCandidates: []bundle.Signal{
			{Type: "generic_failure", Source: "github_actions", Message: strings.Repeat("failure ", 100), Confidence: "medium"},
		},
	}

	result, err := (Generator{}).Generate(failureBundle, Options{SizeLimit: 200})
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
	run, err := project.NewStore(t.TempDir()).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	if err := WriteRepairPrompt(run, Result{Content: "# Repair Prompt\n"}); err != nil {
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

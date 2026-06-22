package tests

import (
	"os"
	"strings"
	"testing"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/commenter"
	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
)

func TestBuildPRCommentOmitsRawLogs(t *testing.T) {
	run := mustRun(t)
	rawLog := strings.Repeat("panic: very long raw stack\n", 50)
	failureBundle := commentFailureBundle(rawLog)

	body, err := commenter.BuildBody(commenter.BodyOptions{
		Run:          run,
		Bundle:       failureBundle,
		RepairPrompt: "# Repair Prompt\nFix undefined Handler.\n\nMore detailed local instructions.",
	})
	if err != nil {
		t.Fatalf("BuildBody() error = %v", err)
	}

	for _, want := range []string{"Tailchase Repair Context", "Fix CI", "undefined: Handler", project.RepairPromptName, project.FailureBundleName, "github-actions.log", "test_weakening", "Raw logs are intentionally omitted", "# Repair Prompt"} {
		if !strings.Contains(body, want) {
			t.Fatalf("comment body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, rawLog) || strings.Contains(body, "panic: very long raw stack") {
		t.Fatalf("comment body included raw log excerpt:\n%s", body)
	}
}

func TestCommentCommandDryRunPrintsBody(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeCommentFailureBundle(t, run)
	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\nFix undefined Handler.\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}

	stdout, _, err := runTailchase(t, "comment", "--run", "12345", "--pr", "7", "--dry-run")
	if err != nil {
		t.Fatalf("tailchase comment --dry-run error = %v", err)
	}
	for _, want := range []string{"Tailchase Repair Context", "Fix CI", "undefined: Handler", "Raw logs are intentionally omitted"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, stdout)
		}
	}
}

func TestCommentCommandRequiresPRNumberAndToken(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeCommentFailureBundle(t, run)
	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}

	_, _, err = runTailchase(t, "comment", "--run", "12345", "--pr", "0", "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "PR number must be greater than zero") {
		t.Fatalf("error = %v, want PR number validation", err)
	}

	_, _, err = runTailchase(t, "comment", "--run", "12345", "--pr", "7")
	if err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN or GH_TOKEN is required") {
		t.Fatalf("error = %v, want missing token validation", err)
	}
	if _, err := os.Stat(".tailchase-comment.md"); !os.IsNotExist(err) {
		t.Fatalf("comment command created unexpected local file")
	}
}

func commentFailureBundle(rawExcerpt string) bundlepkg.FailureBundle {
	return bundlepkg.FailureBundle{
		Goal: bundlepkg.GoalContract{Goal: "Fix CI"},
		Sources: []bundlepkg.EvidenceSource{
			{Source: "github_actions", Path: ".tailchase/runs/12345/evidence/github-actions.log"},
		},
		Budget: bundlepkg.BudgetMetadata{RawEvidenceBytes: 9000, IncludedExcerptBytes: 500, RepeatedBlocksCollapsed: 4},
		SafetyFindings: []bundlepkg.SafetyFinding{
			{Rule: "test_weakening", Decision: "stop", Message: "test weakening detected"},
		},
		RootErrorCandidates: []bundlepkg.Signal{
			{Type: "file_error", Source: "github_actions", Message: "undefined: Handler", File: "internal/app/app.go", Line: 42, Confidence: "high", RawExcerpt: rawExcerpt, RawExcerptPath: ".tailchase/runs/12345/evidence/github-actions.log"},
		},
		Artifacts: []bundlepkg.Artifact{
			{Name: "failure_bundle", Path: ".tailchase/runs/12345/failure-bundle.yml"},
		},
	}
}

func writeCommentFailureBundle(t *testing.T, run project.Run) {
	t.Helper()

	writeFile(t, run.ArtifactPath(project.FailureBundleName), `version: 1
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
goal:
  goal: Fix CI
sources:
  - source: github_actions
    path: .tailchase/runs/12345/evidence/github-actions.log
budget:
  raw_evidence_bytes: 9000
  included_excerpt_bytes: 500
  repeated_blocks_collapsed: 4
safety_findings:
  - rule: test_weakening
    decision: stop
    message: test weakening detected
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    file: internal/app/app.go
    line: 42
    confidence: high
    raw_excerpt: "panic: very long raw stack"
    raw_excerpt_path: .tailchase/runs/12345/evidence/github-actions.log
artifacts:
  - name: failure_bundle
    path: .tailchase/runs/12345/failure-bundle.yml
`)
}

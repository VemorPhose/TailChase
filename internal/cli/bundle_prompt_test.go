package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/project"
)

func TestRunBundleAndPrompt(t *testing.T) {
	root := t.TempDir()
	if err := runInit(commandWithOutput(&bytes.Buffer{}), root); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	writeGoal(t, root)

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	log := `# Tailchase GitHub Actions evidence
repository: owner/repo
run_id: 12345
--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="" ---
internal/app/app.go:42:10: undefined: Handler
--- tailchase-end-job id=11 ---
`
	if err := os.WriteFile(run.EvidencePath(project.GitHubActionsLogName), []byte(log), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var bundleOut bytes.Buffer
	if err := runBundle(commandWithOutput(&bundleOut), root, "12345"); err != nil {
		t.Fatalf("runBundle() error = %v", err)
	}
	if !strings.Contains(bundleOut.String(), project.FailureBundleName) {
		t.Fatalf("bundle output = %q, want failure bundle path", bundleOut.String())
	}

	var promptOut bytes.Buffer
	cmd := commandWithOutput(&promptOut)
	cmd.SetErr(&bytes.Buffer{})
	if err := runPrompt(cmd, root, "12345"); err != nil {
		t.Fatalf("runPrompt() error = %v", err)
	}
	if !strings.Contains(promptOut.String(), "undefined: Handler") {
		t.Fatalf("prompt output missing evidence:\n%s", promptOut.String())
	}
	if _, err := os.Stat(run.ArtifactPath(project.RepairPromptName)); err != nil {
		t.Fatalf("repair prompt was not written: %v", err)
	}
}

func TestRunPromptHonorsFileTarget(t *testing.T) {
	root := t.TempDir()
	if err := runInit(commandWithOutput(&bytes.Buffer{}), root); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	writeGoal(t, root)
	writeConfig(t, root, "file")
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFailureBundle(t, run)

	var out bytes.Buffer
	cmd := commandWithOutput(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := runPrompt(cmd, root, "12345"); err != nil {
		t.Fatalf("runPrompt() error = %v", err)
	}
	if strings.Contains(out.String(), "# Repair Prompt") {
		t.Fatalf("file target printed full prompt:\n%s", out.String())
	}
	if !strings.Contains(out.String(), project.RepairPromptName) {
		t.Fatalf("file target did not print prompt path:\n%s", out.String())
	}
}

func writeGoal(t *testing.T, root string) {
	t.Helper()
	data := []byte(`goal: Fix CI
non_goals:
  - Do not weaken tests
must_preserve:
  - Existing behavior
done_conditions:
  - go test ./... passes
`)
	if err := os.WriteFile(filepath.Join(root, project.DirName, project.GoalFileName), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func writeConfig(t *testing.T, root string, promptTarget string) {
	t.Helper()
	data := []byte(`collectors:
  - github_actions
github:
  repo: owner/repo
failed_jobs_only: true
max_log_lines_per_job: 1200
prompt_target: ` + promptTarget + `
prompt_size_limit: 12000
`)
	if err := os.WriteFile(filepath.Join(root, project.DirName, project.ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func writeFailureBundle(t *testing.T, run project.Run) {
	t.Helper()
	data := []byte(`version: 1
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
goal:
  goal: Fix CI
root_error_candidates:
  - type: file_error
    source: github_actions
    job: unit tests
    message: "undefined: Handler"
    file: internal/app/app.go
    line: 42
    confidence: high
artifacts: []
`)
	if err := os.WriteFile(run.ArtifactPath(project.FailureBundleName), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

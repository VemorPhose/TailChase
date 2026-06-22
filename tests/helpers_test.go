package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/cli"
	"github.com/VemorPhose/TailChase/internal/project"
)

func runTailchase(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := cli.NewRootCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func mustRun(t *testing.T) project.Run {
	t.Helper()

	run, err := project.NewStore(t.TempDir()).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	return run
}

func writeFile(t *testing.T, path string, data string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeGoal(t *testing.T, root string) {
	t.Helper()

	writeFile(t, project.GoalPath(root), `version: 1
goal: Fix CI
non_goals:
  - Do not weaken tests
must_preserve:
  - Existing behavior
done_conditions:
  - go test ./... passes
expected_paths:
  - internal/app
stop_rules:
  - Stop if the fix requires weakening tests
`)
}

func writeConfig(t *testing.T, root string, promptTarget string) {
	t.Helper()

	writeFile(t, project.ConfigPath(root), `version: 1
collectors:
  - github_actions
github:
  repo: owner/repo
failed_jobs_only: true
max_log_lines_per_job: 1200
prompt_target: `+promptTarget+`
prompt_size_limit: 12000
`)
}

func writeFailureBundle(t *testing.T, run project.Run) {
	t.Helper()

	writeFile(t, run.ArtifactPath(project.FailureBundleName), `version: 1
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
}

func hasSubstring(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

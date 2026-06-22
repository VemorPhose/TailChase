package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestGuardAnalyzeDetectsSuspiciousPathsAndLoops(t *testing.T) {
	findings := guard.Analyze(guard.Input{
		Goal: project.Goal{
			Goal:            "Fix CI",
			SuspiciousPaths: []string{".github/workflows"},
		},
		FailureBundle: bundlepkg.FailureBundle{
			RootErrorCandidates: []bundlepkg.Signal{{Message: "undefined: Handler", File: "internal/app/app.go"}},
		},
		EditedPaths:    []string{".github/workflows/ci.yml"},
		CommandHistory: []string{"go test ./...", "go test ./...", "go test ./..."},
		CommandOutput:  "internal/app/app.go:42: undefined: Handler",
	})

	for _, want := range []string{"suspicious_path_edit", "repeated_command_loop", "known_failure_repeated"} {
		if !hasGuardFinding(findings, want) {
			t.Fatalf("findings = %#v, want %s", findings, want)
		}
	}
}

func TestGuardCommandWritesSteeringEvents(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeGoal(t, root)
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeGuardFailureBundle(t, run)
	commandLogPath := filepath.Join(root, "commands.log")
	writeFile(t, commandLogPath, "$ go test ./...\n$ go test ./...\n$ go test ./...\ninternal/app/app.go:42: undefined: Handler\n")

	stdout, _, err := runTailchase(t, "guard", "--run", "12345", "--command-log", commandLogPath)
	if err != nil {
		t.Fatalf("tailchase guard error = %v", err)
	}
	if !strings.Contains(stdout, "Recorded 2 guard finding") {
		t.Fatalf("stdout = %q, want finding count", stdout)
	}
	data, err := os.ReadFile(run.ArtifactPath(project.SteeringEventsName))
	if err != nil {
		t.Fatalf("ReadFile(steering events) error = %v", err)
	}
	for _, want := range []string{"version: 1", "repeated_command_loop", "known_failure_repeated"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("steering events missing %q:\n%s", want, string(data))
		}
	}
}

func hasGuardFinding(findings []guard.Finding, rule string) bool {
	for _, finding := range findings {
		if finding.Rule == rule {
			return true
		}
	}
	return false
}

func writeGuardFailureBundle(t *testing.T, run project.Run) {
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
    message: "undefined: Handler"
    file: internal/app/app.go
    line: 42
    confidence: high
artifacts:
  - name: failure_bundle
    path: .tailchase/runs/12345/failure-bundle.yml
`)
}

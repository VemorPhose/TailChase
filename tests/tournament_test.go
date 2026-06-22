package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/tournament"
)

func TestTournamentCommandEvaluatesFixtureBranchesWithoutMutatingThem(t *testing.T) {
	root := writeTournamentRepo(t)
	passCommit := runGitOutput(t, root, "rev-parse", "candidate-pass")
	failCommit := runGitOutput(t, root, "rev-parse", "candidate-fail")
	t.Chdir(root)

	stdout, _, err := runTailchase(t, "tournament", "candidate-pass", "candidate-fail")
	if err != nil {
		t.Fatalf("tailchase tournament error = %v", err)
	}
	if !strings.Contains(stdout, "Winner: candidate-pass") {
		t.Fatalf("stdout = %q, want candidate-pass winner", stdout)
	}
	if got := runGitOutput(t, root, "branch", "--show-current"); got != "main" {
		t.Fatalf("current branch = %q, want main", got)
	}
	if got := runGitOutput(t, root, "rev-parse", "candidate-pass"); got != passCommit {
		t.Fatalf("candidate-pass commit moved: %s -> %s", passCommit, got)
	}
	if got := runGitOutput(t, root, "rev-parse", "candidate-fail"); got != failCommit {
		t.Fatalf("candidate-fail commit moved: %s -> %s", failCommit, got)
	}

	reportPath := tournament.ReportPath(root, "candidate-pass", "candidate-fail")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", reportPath, err)
	}
	report := string(data)
	for _, want := range []string{"# Tailchase Tournament Report", "## Evaluation Criteria", "Tests: passed", "Tests: failed", "Dependency changes: 1"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func writeTournamentRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "checkout", "-B", "main")
	writeTournamentSource(t, root, 1)
	commitAll(t, root, "initial")

	runGit(t, root, "checkout", "-B", "candidate-pass")
	writeFile(t, filepath.Join(root, "fix.txt"), "minimal fix\n")
	writeTournamentBundle(t, root, "100", "warn")
	commitAll(t, root, "candidate pass")

	runGit(t, root, "checkout", "main")
	runGit(t, root, "checkout", "-B", "candidate-fail")
	writeTournamentSource(t, root, 2)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/tournamentfixture\n\ngo 1.21\n\n// dependency file changed\n")
	writeTournamentBundle(t, root, "200", "stop")
	commitAll(t, root, "candidate fail")

	runGit(t, root, "checkout", "main")
	return root
}

func writeTournamentSource(t *testing.T, root string, value int) {
	t.Helper()

	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/tournamentfixture\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "value.go"), "package tournamentfixture\n\nfunc Value() int { return "+strconv.Itoa(value)+" }\n")
	writeFile(t, filepath.Join(root, "value_test.go"), `package tournamentfixture

import "testing"

func TestValue(t *testing.T) {
	if Value() != 1 {
		t.Fatalf("Value() = %d, want 1", Value())
	}
}
`)
}

func writeTournamentBundle(t *testing.T, root string, runID string, decision string) {
	t.Helper()

	writeFile(t, filepath.Join(root, ".tailchase", "runs", runID, "failure-bundle.yml"), `version: 1
run:
  source: github_actions
  repository: owner/repo
  run_id: "`+runID+`"
goal:
  goal: Fix CI
budget:
  raw_evidence_bytes: 1000
  included_excerpt_bytes: 200
  repeated_blocks_collapsed: 2
  estimated_prompt_bytes: 500
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    confidence: high
safety_findings:
  - rule: repeated_root_failure
    decision: `+decision+`
    message: repeated root failure
artifacts:
  - name: github_actions_log
    path: .tailchase/runs/`+runID+`/evidence/github-actions.log
`)
}

func commitAll(t *testing.T, root string, message string) {
	t.Helper()

	runGit(t, root, "add", ".")
	runGit(t, root, "-c", "user.name=Tailchase Tests", "-c", "user.email=tailchase@example.test", "commit", "-m", message)
}

func runGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()

	output := runGit(t, root, args...)
	return strings.TrimSpace(output)
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s error = %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

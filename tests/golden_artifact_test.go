package tests

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/project"
)

func TestGoldenArtifactsCoverRepairContextSurface(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	run := prepareGoldenRun(t, root, "12345", false)

	assertArtifactContains(t, run.ArtifactPath(project.NormalizedEvidenceName),
		"source: github_actions",
		"provider_kind: ci",
		"undefined: Handler",
	)
	assertArtifactContains(t, run.ArtifactPath(project.FailureBundleName),
		"root_error_candidates:",
		"budget:",
		"undefined: Handler",
		"safety_findings:",
	)
	assertArtifactContains(t, run.ArtifactPath(project.RepairPromptName),
		"# Repair Prompt",
		"## Context Budget",
		"## Likely Root Cause Candidates",
		"Stop if the fix requires weakening tests",
	)
	assertArtifactContains(t, run.ArtifactPath(project.ReportName),
		"# Tailchase Run Report",
		"## Evidence Reduction",
		"## Safety",
	)
	assertArtifactContains(t, run.ArtifactPath(filepath.Join(project.ExportsDirName, "codex-prompt.md")),
		"# Codex Repair Context",
		"## Source Artifacts",
		"## Repair Prompt",
	)
	assertArtifactContains(t, run.ArtifactPath(filepath.Join(project.ExportsDirName, "claude-code-prompt.md")),
		"# Claude Code Repair Context",
		"Preserve the listed goal",
	)
	assertArtifactContains(t, run.ArtifactPath(filepath.Join(project.ExportsDirName, "copilot-instructions.md")),
		"# GitHub Copilot Repair Context",
		"focused repair brief",
	)

	assertArtifactSnapshots(t, run, map[string]string{
		project.NormalizedEvidenceName:                                   "08d212f7b3bb06766a0b23e28c31a97390780315bcf8ff283c22d131ca71ec3a",
		project.FailureBundleName:                                        "d4c225b4e78e4a304f3cd64f4cb43ea47f6b66874755f144121747a9cd41a486",
		project.RepairPromptName:                                         "e0b1ba0335f594a28b96102289d51aed434fd6c54a7d1b82aff108dc62155e5a",
		project.ReportName:                                               "a8038ddbb7b60be04f64ab3f228a3c29af4461e5c5901cdea86b258a2feaf230",
		filepath.Join(project.ExportsDirName, "codex-prompt.md"):         "0534855aa4d7625bfa0083d36841424acacbc0176e8b8bc508f55cc54e625524",
		filepath.Join(project.ExportsDirName, "claude-code-prompt.md"):   "975be1b157658413886edc917c1ea3e8d7dd1ed4035bc2cb65d30e3efde624da",
		filepath.Join(project.ExportsDirName, "copilot-instructions.md"): "a8da58cb39610d8d6abb38621d9cddfc0ca010740e778758e4d7a29eaa443de8",
	})
}

func TestNoNetworkCoreFlow(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	run := prepareGoldenRun(t, root, "67890", true)

	stdout, _, err := runTailchase(t, "comment", "--run", run.ID, "--pr", "7", "--dry-run")
	if err != nil {
		t.Fatalf("tailchase comment --dry-run error = %v", err)
	}
	if !strings.Contains(stdout, "Tailchase Repair Context") {
		t.Fatalf("dry-run comment = %q, want Tailchase repair context", stdout)
	}

	stdout, _, err = runTailchase(t, "mcp", "--run", run.ID, "--list-resources")
	if err != nil {
		t.Fatalf("tailchase mcp --list-resources error = %v", err)
	}
	if !strings.Contains(stdout, "repair-prompt") || !strings.Contains(stdout, "failure-bundle") {
		t.Fatalf("mcp resources = %q, want repair prompt and failure bundle resources", stdout)
	}

	stdout, _, err = runTailchase(t, "adapters", "--target", "codex")
	if err != nil {
		t.Fatalf("tailchase adapters error = %v", err)
	}
	if !strings.Contains(stdout, "codex") {
		t.Fatalf("adapters output = %q, want codex capability", stdout)
	}

	writeFile(t, filepath.Join(root, "commands.log"), "$ go test ./...\n$ go test ./...\n$ go test ./...\ninternal/app/app.go:42: undefined: Handler\n")
	if _, _, err := runTailchase(t, "guard", "--run", run.ID, "--command-log", "commands.log"); err != nil {
		t.Fatalf("tailchase guard error = %v", err)
	}
	if _, _, err := runTailchase(t, "steer", "--run", run.ID, "--target", "copilot", "--checkpoint", "stop_event", "--message", "Stop and ask for help."); err != nil {
		t.Fatalf("tailchase steer error = %v", err)
	}
	if _, _, err := runTailchase(t, "cost", "report", "--run", run.ID); err != nil {
		t.Fatalf("tailchase cost report error = %v", err)
	}
	assertArtifactContains(t, run.ArtifactPath(project.ReportName), "## Steering")
}

func prepareGoldenRun(t *testing.T, root string, runID string, delta bool) project.Run {
	t.Helper()

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeConfig(t, root, "file")
	writeGoal(t, root)
	run, err := project.NewStore(root).EnsureRun(runID)
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFile(t, run.EvidencePath(project.GitHubActionsLogName), `# Tailchase GitHub Actions evidence
repository: owner/repo
run_id: `+runID+`
collected_at: 2026-06-26T00:00:00Z
failed_jobs_only: true

--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="https://github.com/owner/repo/actions/runs/`+runID+`/job/11" ---
internal/app/app.go:42:10: undefined: Handler
.github/workflows/ci.yml:2: unexpected workflow change
--- FAIL: TestHandler
panic: missing required environment variable API_TOKEN
--- tailchase-end-job id=11 ---
`)

	args := []string{"prepare", "--run", runID}
	if delta {
		args = append(args, "--delta")
	}
	args = append(args,
		"--export", "codex",
		"--export", "claude-code",
		"--export", "copilot",
	)
	if _, _, err := runTailchase(t, args...); err != nil {
		t.Fatalf("tailchase prepare error = %v", err)
	}
	return run
}

func assertArtifactContains(t *testing.T, path string, needles ...string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	content := string(data)
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing %q\ncontent:\n%s", path, needle, content)
		}
	}
}

func assertArtifactSnapshots(t *testing.T, run project.Run, expected map[string]string) {
	t.Helper()

	for name, want := range expected {
		path := run.ArtifactPath(name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		got := snapshotHash(normalizeSnapshotContent(string(data)))
		if got != want {
			t.Fatalf("%s snapshot hash = %s, want %s\ncontent:\n%s", name, got, want, string(data))
		}
	}
}

func normalizeSnapshotContent(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "generated_at: ") {
			lines[i] = "generated_at: <timestamp>"
		}
	}
	return strings.Join(lines, "\n")
}

func snapshotHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

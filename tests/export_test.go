package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
)

func TestExportCommandWritesTargetFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeExportFailureBundle(t, run)
	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\nFix undefined Handler.\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}

	tests := []struct {
		target string
		file   string
		title  string
	}{
		{target: "codex", file: "codex-prompt.md", title: "Codex Repair Context"},
		{target: "claude-code", file: "claude-code-prompt.md", title: "Claude Code Repair Context"},
		{target: "copilot", file: "copilot-instructions.md", title: "GitHub Copilot Repair Context"},
	}
	for _, tt := range tests {
		stdout, _, err := runTailchase(t, "export", "--run", "12345", "--target", tt.target)
		if err != nil {
			t.Fatalf("tailchase export --target %s error = %v", tt.target, err)
		}
		wantPath := filepath.ToSlash(filepath.Join(".tailchase", "runs", "12345", project.ExportsDirName, tt.file))
		if !strings.Contains(stdout, wantPath) {
			t.Fatalf("stdout = %q, want %s", stdout, wantPath)
		}
		data, err := os.ReadFile(run.ArtifactPath(filepath.Join(project.ExportsDirName, tt.file)))
		if err != nil {
			t.Fatalf("ReadFile(export %s) error = %v", tt.target, err)
		}
		content := string(data)
		for _, want := range []string{tt.title, "# Repair Prompt", "Fix undefined Handler.", project.FailureBundleName, "github-actions.log", "test_weakening"} {
			if !strings.Contains(content, want) {
				t.Fatalf("%s export missing %q:\n%s", tt.target, want, content)
			}
		}
		if _, err := os.Stat(filepath.Join(root, tt.file)); !os.IsNotExist(err) {
			t.Fatalf("export wrote unexpected root file %s", tt.file)
		}
	}

	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	for _, name := range []string{"codex_export", "claude_code_export", "copilot_export"} {
		if !hasArtifact(meta.Artifacts, name) {
			t.Fatalf("metadata missing export artifact %q: %#v", name, meta.Artifacts)
		}
	}
}

func TestExportCommandRejectsUnsupportedTarget(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeExportFailureBundle(t, run)
	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}

	_, _, err = runTailchase(t, "export", "--run", "12345", "--target", "unknown")
	if err == nil {
		t.Fatal("tailchase export error = nil, want unsupported target error")
	}
	if !strings.Contains(err.Error(), "supported targets: claude-code, codex, copilot") {
		t.Fatalf("error = %v, want supported targets", err)
	}
}

func writeExportFailureBundle(t *testing.T, run project.Run) {
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
    raw_excerpt_path: .tailchase/runs/12345/evidence/github-actions.log
artifacts:
  - name: failure_bundle
    path: .tailchase/runs/12345/failure-bundle.yml
`)
}

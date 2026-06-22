package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestInitCommandCreatesProjectFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	stdout, _, err := runTailchase(t, "init")
	if err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}

	for _, path := range []string{project.ConfigPath(root), project.GoalPath(root)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s was not created: %v", path, err)
		}
	}
	if !strings.Contains(stdout, ".tailchase/config.yml") {
		t.Fatalf("output did not mention config file: %s", stdout)
	}
	for _, path := range []string{project.ConfigPath(root), project.GoalPath(root)} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if !strings.Contains(string(data), "version: 1") {
			t.Fatalf("%s missing schema version:\n%s", path, string(data))
		}
	}
}

func TestInitCommandDoesNotOverwriteExistingFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	writeFile(t, project.ConfigPath(root), "collectors: []\n")
	if _, _, err := runTailchase(t, "init"); err == nil {
		t.Fatal("tailchase init error = nil, want overwrite error")
	}
}

func TestBundleAndPromptCommands(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeGoal(t, root)

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFile(t, run.EvidencePath(project.GitHubActionsLogName), `# Tailchase GitHub Actions evidence
repository: owner/repo
run_id: 12345
--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="" ---
internal/app/app.go:42:10: undefined: Handler
--- tailchase-end-job id=11 ---
`)

	stdout, _, err := runTailchase(t, "bundle", "--run", "12345")
	if err != nil {
		t.Fatalf("tailchase bundle error = %v", err)
	}
	if !strings.Contains(stdout, project.FailureBundleName) {
		t.Fatalf("bundle output = %q, want failure bundle path", stdout)
	}

	stdout, _, err = runTailchase(t, "prompt", "--run", "12345")
	if err != nil {
		t.Fatalf("tailchase prompt error = %v", err)
	}
	if !strings.Contains(stdout, "undefined: Handler") {
		t.Fatalf("prompt output missing evidence:\n%s", stdout)
	}
	if _, err := os.Stat(run.ArtifactPath(project.RepairPromptName)); err != nil {
		t.Fatalf("repair prompt was not written: %v", err)
	}
	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	for _, name := range []string{project.ArtifactNormalizedEvidence, project.ArtifactFailureBundle, project.ArtifactRepairPrompt} {
		if !hasArtifact(meta.Artifacts, name) {
			t.Fatalf("metadata missing artifact %q: %#v", name, meta.Artifacts)
		}
	}
	if !hasArtifact(meta.Artifacts, project.ArtifactAttemptHistory) {
		t.Fatalf("metadata missing artifact %q: %#v", project.ArtifactAttemptHistory, meta.Artifacts)
	}
	history, err := run.ReadAttemptHistory()
	if err != nil {
		t.Fatalf("ReadAttemptHistory() error = %v", err)
	}
	if len(history.Attempts) != 1 {
		t.Fatalf("attempts = %d, want 1", len(history.Attempts))
	}
	if history.Attempts[0].Outcome != project.AttemptOutcomeUnknown {
		t.Fatalf("attempt outcome = %q, want unknown", history.Attempts[0].Outcome)
	}
	if len(history.Attempts[0].RootErrorCandidates) != 1 || history.Attempts[0].RootErrorCandidates[0] != "undefined: Handler" {
		t.Fatalf("attempt root candidates = %#v", history.Attempts[0].RootErrorCandidates)
	}
}

func TestCollectLocalCommandPreservesRawOutput(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeGoal(t, root)
	logPath := filepath.Join(root, "shell-output.log")
	logContent := "running custom check\nError: local command failed\nexit status 1\n"
	writeFile(t, logPath, logContent)

	stdout, _, err := runTailchase(t, "collect-local", "--run", "12345", "--kind", "shell", "--file", logPath)
	if err != nil {
		t.Fatalf("tailchase collect-local error = %v", err)
	}
	if !strings.Contains(stdout, project.ShellCommandLogName) {
		t.Fatalf("collect-local output = %q, want shell evidence path", stdout)
	}

	run, err := project.NewStore(root).OpenRun("12345")
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	data, err := os.ReadFile(run.EvidencePath(project.ShellCommandLogName))
	if err != nil {
		t.Fatalf("ReadFile(shell log) error = %v", err)
	}
	if string(data) != logContent {
		t.Fatalf("shell log = %q, want preserved raw output", string(data))
	}
	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if !hasArtifact(meta.Artifacts, project.ArtifactShellCommandLog) {
		t.Fatalf("metadata missing shell artifact: %#v", meta.Artifacts)
	}

	if _, _, err := runTailchase(t, "bundle", "--run", "12345"); err != nil {
		t.Fatalf("tailchase bundle local evidence error = %v", err)
	}
	normalized, err := bundlepkg.ReadNormalizedEvidence(run)
	if err != nil {
		t.Fatalf("ReadNormalizedEvidence() error = %v", err)
	}
	if normalized.Run.Source != "local_shell" {
		t.Fatalf("normalized source = %q, want local_shell", normalized.Run.Source)
	}
	if !hasSubstring(signalMessages(normalized.Signals), "exit status 1") {
		t.Fatalf("normalized signals = %#v, want shell failure", normalized.Signals)
	}
}

func TestCollectReportsWarnsOnMissingGlob(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}

	_, stderr, err := runTailchase(t, "collect-reports", "--run", "12345", "--glob", "missing/*.xml")
	if err != nil {
		t.Fatalf("tailchase collect-reports error = %v", err)
	}
	if !strings.Contains(stderr, "matched no files") {
		t.Fatalf("stderr = %q, want missing glob warning", stderr)
	}
}

func TestPromptCommandHonorsFileTarget(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeGoal(t, root)
	writeConfig(t, root, "file")

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFailureBundle(t, run)

	stdout, _, err := runTailchase(t, "prompt", "--run", "12345")
	if err != nil {
		t.Fatalf("tailchase prompt error = %v", err)
	}
	if strings.Contains(stdout, "# Repair Prompt") {
		t.Fatalf("file target printed full prompt:\n%s", stdout)
	}
	if !strings.Contains(stdout, project.RepairPromptName) {
		t.Fatalf("file target did not print prompt path:\n%s", stdout)
	}
}

func TestPromptDeltaCommandUsesAttemptHistory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeGoal(t, root)
	writeConfig(t, root, "file")

	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFailureBundle(t, run)
	if _, err := run.AppendAttempt(project.Attempt{
		RootErrorCandidates: []string{"undefined: Handler"},
		Outcome:             "failed",
	}); err != nil {
		t.Fatalf("AppendAttempt() error = %v", err)
	}

	stdout, _, err := runTailchase(t, "prompt", "--run", "12345", "--delta")
	if err != nil {
		t.Fatalf("tailchase prompt --delta error = %v", err)
	}
	if !strings.Contains(stdout, project.RepairPromptName) {
		t.Fatalf("delta file target did not print prompt path:\n%s", stdout)
	}
	data, err := os.ReadFile(run.ArtifactPath(project.RepairPromptName))
	if err != nil {
		t.Fatalf("ReadFile(repair prompt) error = %v", err)
	}
	content := string(data)
	for _, want := range []string{"# Delta Repair Prompt", "Prior attempts recorded: 1", "Same root error seen before: yes", "Evidence excerpt omitted"} {
		if !strings.Contains(content, want) {
			t.Fatalf("delta prompt missing %q:\n%s", want, content)
		}
	}

	history, err := run.ReadAttemptHistory()
	if err != nil {
		t.Fatalf("ReadAttemptHistory() error = %v", err)
	}
	if len(history.Attempts) != 2 {
		t.Fatalf("attempts = %d, want prior plus delta attempt", len(history.Attempts))
	}
}

func hasArtifact(artifacts []project.RunArtifact, name string) bool {
	for _, artifact := range artifacts {
		if artifact.Name == name {
			return true
		}
	}
	return false
}

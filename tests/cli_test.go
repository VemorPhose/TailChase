package tests

import (
	"os"
	"strings"
	"testing"

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

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func TestInitCommandCreatesProjectFiles(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer

	if err := runInit(commandWithOutput(&out), root); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	for _, path := range []string{project.ConfigPath(root), project.GoalPath(root)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s was not created: %v", path, err)
		}
	}
	if !strings.Contains(out.String(), ".tailchase/config.yml") {
		t.Fatalf("output did not mention config file: %s", out.String())
	}
}

func TestInitCommandDoesNotOverwriteExistingFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, project.DirName), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(project.ConfigPath(root), []byte("collectors: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runInit(commandWithOutput(&bytes.Buffer{}), root)
	if err == nil {
		t.Fatal("runInit() error = nil, want overwrite error")
	}
}

func commandWithOutput(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	return cmd
}

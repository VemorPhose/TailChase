package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, root, ConfigFileName, []byte("github:\n  repo: owner/repo\n"))

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.GitHub.Repo != "owner/repo" {
		t.Fatalf("repo = %q, want owner/repo", cfg.GitHub.Repo)
	}
	if cfg.MaxLogLinesPerJob != 1200 {
		t.Fatalf("max log lines = %d, want default", cfg.MaxLogLinesPerJob)
	}
	if !cfg.FailedJobsOnly {
		t.Fatalf("failed_jobs_only default was not applied")
	}
}

func TestConfigValidateRejectsUnknownCollector(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Collectors = []string{"buildkite"}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported collector error")
	}
}

func TestGoalValidateRequiresGoal(t *testing.T) {
	if err := (Goal{}).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing goal error")
	}
}

func writeProjectFile(t *testing.T, root string, name string, data []byte) {
	t.Helper()
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

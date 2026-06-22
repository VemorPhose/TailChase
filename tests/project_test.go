package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/project"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	root := t.TempDir()
	writeFile(t, project.ConfigPath(root), "github:\n  repo: owner/repo\n")

	cfg, err := project.LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.GitHub.Repo != "owner/repo" {
		t.Fatalf("repo = %q, want owner/repo", cfg.GitHub.Repo)
	}
	if cfg.Version != project.SchemaVersion {
		t.Fatalf("version = %d, want %d", cfg.Version, project.SchemaVersion)
	}
	if cfg.MaxLogLinesPerJob != 1200 {
		t.Fatalf("max log lines = %d, want default", cfg.MaxLogLinesPerJob)
	}
	if !cfg.FailedJobsOnly {
		t.Fatal("failed_jobs_only default was not applied")
	}
}

func TestConfigValidateRejectsUnknownCollector(t *testing.T) {
	cfg := project.DefaultConfig()
	cfg.Collectors = []string{"buildkite"}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported collector error")
	}
}

func TestConfigValidateRejectsUnsupportedVersion(t *testing.T) {
	cfg := project.DefaultConfig()
	cfg.Version = 99

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported version error")
	}
}

func TestMarshalConfigIncludesVersion(t *testing.T) {
	data, err := project.MarshalConfig(project.DefaultConfig())
	if err != nil {
		t.Fatalf("MarshalConfig() error = %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("config YAML missing version:\n%s", string(data))
	}
}

func TestLoadGoalDefaultsMissingVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, project.GoalPath(root), "goal: Fix CI\n")

	goal, err := project.LoadGoal(root)
	if err != nil {
		t.Fatalf("LoadGoal() error = %v", err)
	}
	if goal.Version != project.SchemaVersion {
		t.Fatalf("version = %d, want %d", goal.Version, project.SchemaVersion)
	}
}

func TestGoalValidateRejectsUnsupportedVersion(t *testing.T) {
	goal := project.DefaultGoal()
	goal.Version = 99

	if err := goal.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported version error")
	}
}

func TestGoalValidateRequiresGoal(t *testing.T) {
	if err := (project.Goal{}).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing goal error")
	}
}

func TestMarshalGoalIncludesVersion(t *testing.T) {
	data, err := project.MarshalGoal(project.DefaultGoal())
	if err != nil {
		t.Fatalf("MarshalGoal() error = %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("goal YAML missing version:\n%s", string(data))
	}
}

func TestEnsureRunCreatesExpectedLayout(t *testing.T) {
	root := t.TempDir()
	store := project.NewStore(root)

	run, err := store.EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	wantEvidence := filepath.Join(root, project.DirName, project.RunsDirName, "12345", project.EvidenceDirName)
	if run.EvidenceDir() != wantEvidence {
		t.Fatalf("EvidenceDir() = %q, want %q", run.EvidenceDir(), wantEvidence)
	}
	if _, err := os.Stat(wantEvidence); err != nil {
		t.Fatalf("evidence dir was not created: %v", err)
	}
}

func TestOpenRunRequiresExistingRun(t *testing.T) {
	_, err := project.NewStore(t.TempDir()).OpenRun("12345")
	if err == nil {
		t.Fatal("OpenRun() error = nil, want missing run error")
	}
}

func TestValidateRunID(t *testing.T) {
	tests := map[string]bool{
		"12345": true,
		"":      false,
		"abc":   false,
		"../1":  false,
	}

	for runID, wantOK := range tests {
		err := project.ValidateRunID(runID)
		if (err == nil) != wantOK {
			t.Fatalf("ValidateRunID(%q) error = %v, want ok = %v", runID, err, wantOK)
		}
	}
}

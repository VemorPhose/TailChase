package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if cfg.Safety.Mode != "manual" || len(cfg.Safety.StopOn) == 0 {
		t.Fatalf("safety defaults = %#v, want manual mode with stop rules", cfg.Safety)
	}
	if cfg.Prompt.Mode != "heuristic" {
		t.Fatalf("prompt mode = %q, want heuristic", cfg.Prompt.Mode)
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

func TestConfigValidateRejectsBadSafetyMode(t *testing.T) {
	cfg := project.DefaultConfig()
	cfg.Safety.Mode = "automatic"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid safety mode error")
	}
}

func TestConfigValidateModelModeRequiresSettings(t *testing.T) {
	cfg := project.DefaultConfig()
	cfg.Prompt.Mode = "model"
	cfg.Model = project.ModelConfig{}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing model settings error")
	}

	cfg.Model = project.ModelConfig{
		Provider:  "openai_compatible",
		BaseURL:   "https://api.example.com/v1",
		Model:     "example-model",
		APIKeyEnv: "EXAMPLE_API_KEY",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() with model settings error = %v", err)
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
	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if meta.Version != project.SchemaVersion {
		t.Fatalf("metadata version = %d, want %d", meta.Version, project.SchemaVersion)
	}
	if meta.ID != "12345" {
		t.Fatalf("metadata ID = %q, want 12345", meta.ID)
	}
	if meta.CreatedAt.IsZero() {
		t.Fatal("metadata CreatedAt was not set")
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

func TestRunArtifactIndex(t *testing.T) {
	run := mustRun(t)

	createdAt := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	if err := run.RecordArtifact("first", "test", run.ArtifactPath("first.txt"), createdAt); err != nil {
		t.Fatalf("RecordArtifact() error = %v", err)
	}
	if err := run.RecordArtifact("first", "test", run.ArtifactPath("second.txt"), createdAt.Add(time.Minute)); err != nil {
		t.Fatalf("RecordArtifact() replace error = %v", err)
	}

	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if len(meta.Artifacts) != 1 {
		t.Fatalf("artifacts = %d, want replacement to keep 1", len(meta.Artifacts))
	}
	got := meta.Artifacts[0]
	if got.Name != "first" || got.Type != "test" {
		t.Fatalf("artifact = %#v, want first/test", got)
	}
	if !strings.HasSuffix(got.Path, ".tailchase/runs/12345/second.txt") {
		t.Fatalf("artifact path = %q, want relative second.txt path", got.Path)
	}
	if !got.CreatedAt.Equal(createdAt.Add(time.Minute)) {
		t.Fatalf("artifact CreatedAt = %s, want replacement timestamp", got.CreatedAt)
	}
}

func TestRunArtifactFileHelpers(t *testing.T) {
	run := mustRun(t)

	if err := run.WriteArtifactFile("example.txt", "example", "test", []byte("hello")); err != nil {
		t.Fatalf("WriteArtifactFile() error = %v", err)
	}
	data, err := run.ReadArtifactFile("example.txt")
	if err != nil {
		t.Fatalf("ReadArtifactFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("artifact file = %q, want hello", string(data))
	}

	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if len(meta.Artifacts) != 1 || meta.Artifacts[0].Name != "example" {
		t.Fatalf("metadata artifacts = %#v, want example artifact", meta.Artifacts)
	}

	if _, err := run.ReadArtifactFile("missing.txt"); err == nil {
		t.Fatal("ReadArtifactFile() error = nil, want missing artifact error")
	}
}

func TestRunAbsolutePath(t *testing.T) {
	root := t.TempDir()
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	relative := ".tailchase/runs/12345/evidence/github-actions.log"
	if got := run.AbsolutePath(relative); got != run.EvidencePath(project.GitHubActionsLogName) {
		t.Fatalf("AbsolutePath(%q) = %q, want %q", relative, got, run.EvidencePath(project.GitHubActionsLogName))
	}
	if got := run.AbsolutePath(run.Dir()); got != run.Dir() {
		t.Fatalf("AbsolutePath(abs) = %q, want %q", got, run.Dir())
	}
}

func TestAttemptHistoryAppendReadOrderAndDefaults(t *testing.T) {
	root := t.TempDir()
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	first, err := run.AppendAttempt(project.Attempt{
		RootErrorCandidates: []string{"undefined: Handler"},
		Outcome:             "failed",
	})
	if err != nil {
		t.Fatalf("AppendAttempt(first) error = %v", err)
	}
	second, err := run.AppendAttempt(project.Attempt{
		RootErrorCandidates: []string{"missing API_TOKEN"},
		Outcome:             "passed",
	})
	if err != nil {
		t.Fatalf("AppendAttempt(second) error = %v", err)
	}

	if first.Number != 1 || second.Number != 2 {
		t.Fatalf("attempt numbers = %d, %d, want 1, 2", first.Number, second.Number)
	}
	if first.RunID != "12345" {
		t.Fatalf("run ID = %q, want 12345", first.RunID)
	}
	if !strings.HasSuffix(first.BundlePath, ".tailchase/runs/12345/failure-bundle.yml") {
		t.Fatalf("bundle path = %q, want failure bundle path", first.BundlePath)
	}
	if !strings.HasSuffix(first.PromptPath, ".tailchase/runs/12345/repair-prompt.md") {
		t.Fatalf("prompt path = %q, want repair prompt path", first.PromptPath)
	}
	if first.CreatedAt.IsZero() {
		t.Fatal("CreatedAt was not set")
	}

	reopened, err := project.NewStore(root).OpenRun("12345")
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	history, err := reopened.ReadAttemptHistory()
	if err != nil {
		t.Fatalf("ReadAttemptHistory() error = %v", err)
	}
	if len(history.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(history.Attempts))
	}
	if history.Attempts[0].RootErrorCandidates[0] != "undefined: Handler" || history.Attempts[1].Outcome != "passed" {
		t.Fatalf("history order/content = %#v", history.Attempts)
	}

	meta, err := reopened.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if !hasArtifact(meta.Artifacts, project.ArtifactAttemptHistory) {
		t.Fatalf("metadata missing attempt history artifact: %#v", meta.Artifacts)
	}
}

func TestAttemptHistoryDefaultsMissingVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.AttemptHistoryPath(), `attempts:
  - number: 1
    run_id: "12345"
    bundle_path: .tailchase/runs/12345/failure-bundle.yml
    prompt_path: .tailchase/runs/12345/repair-prompt.md
    outcome: failed
    created_at: 2026-06-22T10:00:00Z
`)

	history, err := run.ReadAttemptHistory()
	if err != nil {
		t.Fatalf("ReadAttemptHistory() error = %v", err)
	}
	if history.Version != project.SchemaVersion {
		t.Fatalf("version = %d, want %d", history.Version, project.SchemaVersion)
	}
}

func TestAttemptHistoryRejectsUnsupportedVersion(t *testing.T) {
	run := mustRun(t)
	writeFile(t, run.AttemptHistoryPath(), "version: 99\n")

	if _, err := run.ReadAttemptHistory(); err == nil {
		t.Fatal("ReadAttemptHistory() error = nil, want unsupported version error")
	}
}

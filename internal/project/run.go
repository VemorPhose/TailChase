package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	RunsDirName            = "runs"
	EvidenceDirName        = "evidence"
	TestReportsDirName     = "test-reports"
	ComposeLogsDirName     = "compose"
	RunMetadataName        = "run.yml"
	AttemptHistoryName     = "attempt-history.yml"
	GitHubActionsLogName   = "github-actions.log"
	GoTestLogName          = "go-test.log"
	ShellCommandLogName    = "shell-command.log"
	NormalizedEvidenceName = "normalized-evidence.yml"
	FailureBundleName      = "failure-bundle.yml"
	RepairPromptName       = "repair-prompt.md"
	ReportName             = "report.md"

	ArtifactGitHubActionsLog   = "github_actions_log"
	ArtifactGoTestLog          = "go_test_log"
	ArtifactShellCommandLog    = "shell_command_log"
	ArtifactTestReport         = "test_report"
	ArtifactDockerComposeLog   = "docker_compose_log"
	ArtifactNormalizedEvidence = "normalized_evidence"
	ArtifactFailureBundle      = "failure_bundle"
	ArtifactRepairPrompt       = "repair_prompt"
	ArtifactAttemptHistory     = "attempt_history"
)

type Store struct {
	Root string
}

type Run struct {
	ID    string
	root  string
	store Store
}

type RunMetadata struct {
	Version   int           `yaml:"version"`
	ID        string        `yaml:"id"`
	CreatedAt time.Time     `yaml:"created_at,omitempty"`
	Artifacts []RunArtifact `yaml:"artifacts,omitempty"`
}

type RunArtifact struct {
	Name      string    `yaml:"name"`
	Type      string    `yaml:"type"`
	Path      string    `yaml:"path"`
	CreatedAt time.Time `yaml:"created_at"`
}

func NewStore(root string) Store {
	if root == "" {
		root = "."
	}
	return Store{Root: root}
}

func (s Store) ProjectDir() string {
	return filepath.Join(s.Root, DirName)
}

func (s Store) RunsDir() string {
	return filepath.Join(s.ProjectDir(), RunsDirName)
}

func (s Store) EnsureProjectDir() error {
	return os.MkdirAll(s.ProjectDir(), 0o755)
}

func (s Store) EnsureRun(runID string) (Run, error) {
	if err := ValidateRunID(runID); err != nil {
		return Run{}, err
	}
	run := s.Run(runID)
	if err := os.MkdirAll(run.EvidenceDir(), 0o755); err != nil {
		return Run{}, err
	}
	if err := run.ensureMetadata(); err != nil {
		return Run{}, err
	}
	return run, nil
}

func (s Store) OpenRun(runID string) (Run, error) {
	if err := ValidateRunID(runID); err != nil {
		return Run{}, err
	}
	run := s.Run(runID)
	if _, err := os.Stat(run.Dir()); err != nil {
		if os.IsNotExist(err) {
			return Run{}, fmt.Errorf("run %s does not exist; run tailchase collect --run %s first", runID, runID)
		}
		return Run{}, err
	}
	return run, nil
}

func (s Store) Run(runID string) Run {
	return Run{
		ID:    runID,
		root:  filepath.Join(s.RunsDir(), runID),
		store: s,
	}
}

func (r Run) Dir() string {
	return r.root
}

func (r Run) EvidenceDir() string {
	return filepath.Join(r.root, EvidenceDirName)
}

func (r Run) MetadataPath() string {
	return filepath.Join(r.root, RunMetadataName)
}

func (r Run) EvidencePath(name string) string {
	return filepath.Join(r.EvidenceDir(), name)
}

func (r Run) ArtifactPath(name string) string {
	return filepath.Join(r.root, name)
}

func (r Run) RelativePath(path string) string {
	rel, err := filepath.Rel(r.store.Root, path)
	if err != nil {
		return path
	}
	return rel
}

func (r Run) AbsolutePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(r.store.Root, filepath.FromSlash(path))
}

func (r Run) WriteArtifactFile(fileName string, artifactName string, artifactType string, data []byte) error {
	path := r.ArtifactPath(fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	return r.RecordArtifact(artifactName, artifactType, path, time.Now().UTC())
}

func (r Run) ReadArtifactFile(fileName string) ([]byte, error) {
	path := r.ArtifactPath(fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s is missing for run %s", r.RelativePath(path), r.ID)
		}
		return nil, err
	}
	return data, nil
}

func (r Run) RecordArtifact(name string, artifactType string, path string, createdAt time.Time) error {
	name = strings.TrimSpace(name)
	artifactType = strings.TrimSpace(artifactType)
	if name == "" {
		return errors.New("artifact name is required")
	}
	if artifactType == "" {
		return errors.New("artifact type is required")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("artifact path is required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	meta, err := r.ReadMetadata()
	if err != nil {
		return err
	}
	meta.Version = SchemaVersion
	if meta.ID == "" {
		meta.ID = r.ID
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = createdAt.UTC()
	}

	record := RunArtifact{
		Name:      name,
		Type:      artifactType,
		Path:      r.relativeArtifactPath(path),
		CreatedAt: createdAt.UTC(),
	}
	replaced := false
	for i, artifact := range meta.Artifacts {
		if artifact.Name == name {
			meta.Artifacts[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		meta.Artifacts = append(meta.Artifacts, record)
	}
	return r.WriteMetadata(meta)
}

func (r Run) ReadMetadata() (RunMetadata, error) {
	data, err := os.ReadFile(r.MetadataPath())
	if err != nil {
		if os.IsNotExist(err) {
			return RunMetadata{Version: SchemaVersion, ID: r.ID}, nil
		}
		return RunMetadata{}, err
	}
	var meta RunMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return RunMetadata{}, fmt.Errorf("parse run metadata: %w", err)
	}
	if meta.Version == 0 {
		meta.Version = SchemaVersion
	}
	if meta.Version != SchemaVersion {
		return RunMetadata{}, fmt.Errorf("unsupported run metadata version %d", meta.Version)
	}
	if meta.ID == "" {
		meta.ID = r.ID
	}
	return meta, nil
}

func (r Run) WriteMetadata(meta RunMetadata) error {
	if meta.Version == 0 {
		meta.Version = SchemaVersion
	}
	if meta.Version != SchemaVersion {
		return fmt.Errorf("unsupported run metadata version %d", meta.Version)
	}
	if meta.ID == "" {
		meta.ID = r.ID
	}
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(r.MetadataPath(), data, 0o644)
}

func (r Run) ensureMetadata() error {
	if _, err := os.Stat(r.MetadataPath()); err == nil {
		_, err := r.ReadMetadata()
		return err
	} else if !os.IsNotExist(err) {
		return err
	}
	return r.WriteMetadata(RunMetadata{
		Version:   SchemaVersion,
		ID:        r.ID,
		CreatedAt: time.Now().UTC(),
	})
}

func (r Run) relativeArtifactPath(path string) string {
	if filepath.IsAbs(path) {
		return r.RelativePath(path)
	}
	return filepath.Clean(path)
}

func ValidateRunID(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return errors.New("run ID is required")
	}
	if _, err := strconv.ParseInt(runID, 10, 64); err != nil {
		return fmt.Errorf("run ID %q must be a GitHub Actions numeric run ID", runID)
	}
	if strings.ContainsAny(runID, `/\`) || filepath.Base(runID) != runID {
		return fmt.Errorf("run ID %q must not contain path separators", runID)
	}
	return nil
}

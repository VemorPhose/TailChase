package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	RunsDirName            = "runs"
	EvidenceDirName        = "evidence"
	GitHubActionsLogName   = "github-actions.log"
	NormalizedEvidenceName = "normalized-evidence.yml"
	FailureBundleName      = "failure-bundle.yml"
	RepairPromptName       = "repair-prompt.md"
	ReportName             = "report.md"
)

type Store struct {
	Root string
}

type Run struct {
	ID    string
	root  string
	store Store
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

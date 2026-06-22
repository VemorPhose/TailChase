package project

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const AttemptOutcomeUnknown = "unknown"

type AttemptHistory struct {
	Version  int       `yaml:"version"`
	Attempts []Attempt `yaml:"attempts,omitempty"`
}

type Attempt struct {
	Number              int       `yaml:"number"`
	RunID               string    `yaml:"run_id"`
	BundlePath          string    `yaml:"bundle_path"`
	PromptPath          string    `yaml:"prompt_path"`
	RootErrorCandidates []string  `yaml:"root_error_candidates,omitempty"`
	Outcome             string    `yaml:"outcome"`
	CreatedAt           time.Time `yaml:"created_at"`
}

func (r Run) AttemptHistoryPath() string {
	return filepath.Join(r.root, AttemptHistoryName)
}

func (r Run) ReadAttemptHistory() (AttemptHistory, error) {
	data, err := os.ReadFile(r.AttemptHistoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return AttemptHistory{Version: SchemaVersion}, nil
		}
		return AttemptHistory{}, err
	}
	var history AttemptHistory
	if err := yaml.Unmarshal(data, &history); err != nil {
		return AttemptHistory{}, fmt.Errorf("parse attempt history: %w", err)
	}
	if history.Version == 0 {
		history.Version = SchemaVersion
	}
	if history.Version != SchemaVersion {
		return AttemptHistory{}, fmt.Errorf("unsupported attempt history version %d", history.Version)
	}
	return history, nil
}

func (r Run) WriteAttemptHistory(history AttemptHistory) error {
	if history.Version == 0 {
		history.Version = SchemaVersion
	}
	if history.Version != SchemaVersion {
		return fmt.Errorf("unsupported attempt history version %d", history.Version)
	}
	data, err := yaml.Marshal(history)
	if err != nil {
		return err
	}
	return os.WriteFile(r.AttemptHistoryPath(), data, 0o644)
}

func (r Run) AppendAttempt(attempt Attempt) (Attempt, error) {
	history, err := r.ReadAttemptHistory()
	if err != nil {
		return Attempt{}, err
	}
	if attempt.Number == 0 {
		attempt.Number = len(history.Attempts) + 1
	}
	if attempt.RunID == "" {
		attempt.RunID = r.ID
	}
	if attempt.BundlePath == "" {
		attempt.BundlePath = r.RelativePath(r.ArtifactPath(FailureBundleName))
	}
	if attempt.PromptPath == "" {
		attempt.PromptPath = r.RelativePath(r.ArtifactPath(RepairPromptName))
	}
	if attempt.Outcome == "" {
		attempt.Outcome = AttemptOutcomeUnknown
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	history.Attempts = append(history.Attempts, attempt)
	if err := r.WriteAttemptHistory(history); err != nil {
		return Attempt{}, err
	}
	if err := r.RecordArtifact(ArtifactAttemptHistory, "attempt_history", r.AttemptHistoryPath(), attempt.CreatedAt); err != nil {
		return Attempt{}, err
	}
	return attempt, nil
}

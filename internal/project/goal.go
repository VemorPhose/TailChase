package project

import (
	"errors"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const GoalFileName = "goal.yml"

type Goal struct {
	Version         int      `yaml:"version"`
	Goal            string   `yaml:"goal"`
	NonGoals        []string `yaml:"non_goals"`
	MustPreserve    []string `yaml:"must_preserve"`
	DoneConditions  []string `yaml:"done_conditions"`
	SuspiciousPaths []string `yaml:"suspicious_paths,omitempty"`
}

func DefaultGoal() Goal {
	return Goal{
		Version: SchemaVersion,
		Goal:    "TODO: describe the original task goal.",
		NonGoals: []string{
			"Do not broaden the change beyond the task.",
			"Do not weaken or delete tests to make CI pass.",
		},
		MustPreserve: []string{
			"Existing public behavior unless the task explicitly changes it.",
		},
		DoneConditions: []string{
			"Relevant tests pass locally.",
			"GitHub Actions passes for the branch.",
		},
	}
}

func GoalPath(root string) string {
	return filepath.Join(root, DirName, GoalFileName)
}

func LoadGoal(root string) (Goal, error) {
	goal := Goal{Version: SchemaVersion}
	if err := loadYAML(GoalPath(root), &goal); err != nil {
		return Goal{}, err
	}
	if err := goal.Validate(); err != nil {
		return Goal{}, fmt.Errorf("invalid goal: %w", err)
	}
	return goal, nil
}

func (g Goal) Validate() error {
	if g.Version == 0 {
		g.Version = SchemaVersion
	}
	if g.Version != SchemaVersion {
		return fmt.Errorf("unsupported goal version %d", g.Version)
	}
	if g.Goal == "" {
		return errors.New("goal must not be empty")
	}
	return nil
}

func MarshalGoal(goal Goal) ([]byte, error) {
	if goal.Version == 0 {
		goal.Version = SchemaVersion
	}
	if err := goal.Validate(); err != nil {
		return nil, err
	}
	return yaml.Marshal(goal)
}

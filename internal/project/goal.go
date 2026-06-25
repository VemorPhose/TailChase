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
	ExpectedPaths   []string `yaml:"expected_paths,omitempty"`
	SuspiciousPaths []string `yaml:"suspicious_paths,omitempty"`
	StopRules       []string `yaml:"stop_rules,omitempty"`
}

func DefaultGoal() Goal {
	return Goal{
		Version: SchemaVersion,
		Goal:    "Keep the requested change small, focused, and aligned with the project intent.",
		NonGoals: []string{
			"Do not broaden the work beyond the user-requested change.",
			"Do not weaken, skip, or delete tests to make checks pass.",
			"Do not hide failures or ignore new errors.",
		},
		MustPreserve: []string{
			"Existing public behavior unless the task explicitly changes it.",
			"User data, secrets, credentials, and local environment assumptions.",
			"Clear errors and useful logs for future debugging.",
		},
		DoneConditions: []string{
			"Relevant local tests or checks pass.",
			"Remote CI passes for the branch when CI is available.",
			"Failure bundle, repair prompt, and report artifacts are up to date after a CI failure.",
		},
		ExpectedPaths: []string{
			".",
		},
		SuspiciousPaths: []string{
			".github/workflows",
			"go.mod",
			"go.sum",
			"package.json",
			"package-lock.json",
			"pnpm-lock.yaml",
			"yarn.lock",
		},
		StopRules: []string{
			"Stop before weakening or deleting tests.",
			"Stop before changing dependency or CI configuration unless that is the task.",
			"Stop before making broad rewrites unrelated to the observed failure.",
			"Stop if the same root failure repeats after a repair attempt.",
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

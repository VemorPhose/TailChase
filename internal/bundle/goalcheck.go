package bundle

import (
	"fmt"
	"strings"

	"github.com/VemorPhose/TailChase/internal/project"
)

type GoalCheckInput struct {
	Signals     []Signal
	EditedPaths []string
}

func GoalContractWarnings(goal project.Goal, input GoalCheckInput) []string {
	var warnings []string
	add := func(warning string) {
		if warning == "" {
			return
		}
		for _, existing := range warnings {
			if existing == warning {
				return
			}
		}
		warnings = append(warnings, warning)
	}

	if vagueGoal(goal.Goal) {
		add("goal.yml goal is missing or vague; repair prompts may be less anchored")
	}
	if len(goal.NonGoals) == 0 {
		add("goal.yml has no non_goals")
	}
	if len(goal.MustPreserve) == 0 {
		add("goal.yml has no must_preserve")
	}
	if len(goal.DoneConditions) == 0 {
		add("goal.yml has no done_conditions")
	}
	if len(cleanGoalPaths(goal.ExpectedPaths)) == 0 {
		add("goal.yml has no expected_paths; drift checks may be broad")
	}
	if len(goal.StopRules) == 0 {
		add("goal.yml has no stop_rules")
	}

	expectedPaths := cleanGoalPaths(goal.ExpectedPaths)
	suspiciousPaths := cleanGoalPaths(goal.SuspiciousPaths)
	for _, signal := range input.Signals {
		file := cleanGoalPath(signal.File)
		if file == "" {
			continue
		}
		if matchAnyPath(file, suspiciousPaths) {
			add(fmt.Sprintf("failure signal touches suspicious path %q", file))
		}
		if len(expectedPaths) > 0 && !matchAnyPath(file, expectedPaths) {
			add(fmt.Sprintf("failure signal %q is outside expected_paths", file))
		}
	}
	for _, path := range input.EditedPaths {
		path = cleanGoalPath(path)
		if path == "" {
			continue
		}
		if matchAnyPath(path, suspiciousPaths) {
			add(fmt.Sprintf("edit touches suspicious path %q", path))
		}
		if len(expectedPaths) > 0 && !matchAnyPath(path, expectedPaths) {
			add(fmt.Sprintf("edit path %q is outside expected_paths", path))
		}
	}
	return warnings
}

func vagueGoal(goal string) bool {
	goal = strings.TrimSpace(goal)
	lower := strings.ToLower(goal)
	return goal == "" || strings.Contains(lower, "todo") || len(strings.Fields(goal)) < 3
}

func cleanGoalPaths(paths []string) []string {
	cleaned := make([]string, 0, len(paths))
	for _, path := range paths {
		path = cleanGoalPath(path)
		if path == "" || strings.Contains(strings.ToLower(path), "todo") {
			continue
		}
		cleaned = append(cleaned, path)
	}
	return cleaned
}

func cleanGoalPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	return path
}

func matchAnyPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if path == pattern || strings.HasPrefix(path, pattern+"/") {
			return true
		}
	}
	return false
}

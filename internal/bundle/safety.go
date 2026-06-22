package bundle

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/VemorPhose/TailChase/internal/project"
)

const (
	SafetyDecisionWarn = "warn"
	SafetyDecisionStop = "stop"

	SafetyRuleRepeatedRootFailure = "repeated_root_failure"
	SafetyRuleGoalDrift           = "goal_drift"
	SafetyRuleTestWeakening       = "test_weakening"
	SafetyRuleDependencyChange    = "dependency_change"
	SafetyRuleSuspiciousPathEdit  = "suspicious_path_edit"
)

type SafetyEngine struct {
	Config project.SafetyConfig
}

type SafetyInput struct {
	Bundle       FailureBundle
	Goal         project.Goal
	Signals      []Signal
	EditedPaths  []string
	ChangedFiles []ChangedFile
}

type ChangedFile struct {
	Path string
	Diff string
}

func (e SafetyEngine) Evaluate(input SafetyInput) []SafetyFinding {
	var findings []SafetyFinding
	add := func(rule string, message string, path string) {
		findings = append(findings, SafetyFinding{
			Rule:     rule,
			Decision: e.decision(rule),
			Message:  message,
			Path:     cleanGoalPath(path),
		})
	}

	if input.Bundle.AttemptContext.SameRootErrorSeenBefore {
		add(SafetyRuleRepeatedRootFailure, "same root error was seen in prior attempts", "")
	}

	signals := input.Signals
	if signals == nil {
		signals = append(append([]Signal{}, input.Bundle.RootErrorCandidates...), input.Bundle.DownstreamSymptoms...)
	}
	for _, warning := range GoalContractWarnings(input.Goal, GoalCheckInput{Signals: signals, EditedPaths: input.EditedPaths}) {
		if strings.Contains(warning, "goal is missing or vague") {
			add(SafetyRuleGoalDrift, warning, "")
		}
		if strings.Contains(warning, "outside expected_paths") {
			add(SafetyRuleGoalDrift, warning, warningPath(warning))
		}
		if strings.Contains(warning, "edit touches suspicious path") {
			add(SafetyRuleSuspiciousPathEdit, warning, warningPath(warning))
		}
	}

	seenDependency := map[string]bool{}
	for _, path := range input.EditedPaths {
		path = cleanGoalPath(path)
		if isDependencyPath(path) && !seenDependency[path] {
			seenDependency[path] = true
			add(SafetyRuleDependencyChange, fmt.Sprintf("dependency file changed: %s", path), path)
		}
	}

	seenWeakening := map[string]bool{}
	for _, file := range input.ChangedFiles {
		path := cleanGoalPath(file.Path)
		if looksLikeTestWeakening(path, file.Diff) && !seenWeakening[path] {
			seenWeakening[path] = true
			add(SafetyRuleTestWeakening, fmt.Sprintf("possible test weakening in %s", path), path)
		}
	}
	return findings
}

func (e SafetyEngine) decision(rule string) string {
	stopOn := e.Config.StopOn
	if slices.Contains(stopOn, rule) {
		return SafetyDecisionStop
	}
	return SafetyDecisionWarn
}

func warningPath(warning string) string {
	start := strings.IndexByte(warning, '"')
	if start < 0 {
		return ""
	}
	rest := warning[start+1:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func isDependencyPath(path string) bool {
	switch filepath.Base(path) {
	case "go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "requirements.txt", "pyproject.toml", "Cargo.toml", "Cargo.lock":
		return true
	default:
		return false
	}
}

func looksLikeTestWeakening(path string, diff string) bool {
	lowerPath := strings.ToLower(path)
	if !strings.Contains(lowerPath, "test") && !strings.HasSuffix(lowerPath, "_test.go") && !strings.HasSuffix(lowerPath, ".spec.ts") && !strings.HasSuffix(lowerPath, ".test.ts") {
		return false
	}
	lowerDiff := strings.ToLower(diff)
	return strings.Contains(lowerDiff, "+\tt.skip") ||
		strings.Contains(lowerDiff, "+ t.skip") ||
		strings.Contains(lowerDiff, ".skip(") ||
		strings.Contains(lowerDiff, "pytest.mark.skip") ||
		strings.Contains(lowerDiff, "skip(")
}

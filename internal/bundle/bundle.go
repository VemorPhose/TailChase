package bundle

import (
	"fmt"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

type Compiler struct {
	Now func() time.Time
}

func (c Compiler) Compile(run project.Run, goal project.Goal, normalized NormalizedEvidence) (FailureBundle, error) {
	if c.Now == nil {
		c.Now = time.Now
	}
	if err := goal.Validate(); err != nil {
		return FailureBundle{}, err
	}

	rootCandidates, symptoms := classifySignals(normalized.Signals)
	warnings := append([]string{}, normalized.Warnings...)
	warnings = append(warnings, goalWarnings(goal, normalized.Signals)...)
	if len(rootCandidates) == 0 {
		warnings = append(warnings, "no root-error candidates were identified")
	}
	history, err := run.ReadAttemptHistory()
	if err != nil {
		return FailureBundle{}, err
	}
	context := attemptContext(rootCandidates, history)
	if warning := repeatedRootWarning(context); warning != "" {
		warnings = append(warnings, warning)
	}

	runMeta := normalized.Run
	if runMeta.Source == "" {
		runMeta.Source = "github_actions"
	}
	if runMeta.RunID == "" {
		runMeta.RunID = run.ID
	}

	return FailureBundle{
		Version:             SchemaVersion,
		GeneratedAt:         c.Now().UTC(),
		Run:                 runMeta,
		Goal:                goalContract(goal),
		Sources:             normalized.Sources,
		AttemptContext:      context,
		RootErrorCandidates: rootCandidates,
		DownstreamSymptoms:  symptoms,
		Artifacts: []Artifact{
			{Name: project.ArtifactGitHubActionsLog, Path: run.RelativePath(run.EvidencePath(project.GitHubActionsLogName))},
			{Name: project.ArtifactNormalizedEvidence, Path: run.RelativePath(run.ArtifactPath(project.NormalizedEvidenceName))},
			{Name: project.ArtifactFailureBundle, Path: run.RelativePath(run.ArtifactPath(project.FailureBundleName))},
		},
		Warnings: warnings,
	}, nil
}

func WriteFailureBundle(run project.Run, bundle FailureBundle) error {
	if bundle.Version == 0 {
		bundle.Version = SchemaVersion
	}
	data, err := yaml.Marshal(bundle)
	if err != nil {
		return err
	}
	return run.WriteArtifactFile(project.FailureBundleName, project.ArtifactFailureBundle, "failure_bundle", data)
}

func ReadFailureBundle(run project.Run) (FailureBundle, error) {
	data, err := run.ReadArtifactFile(project.FailureBundleName)
	if err != nil {
		return FailureBundle{}, fmt.Errorf("read failure bundle: %w", err)
	}
	var bundle FailureBundle
	if err := yaml.Unmarshal(data, &bundle); err != nil {
		return FailureBundle{}, fmt.Errorf("parse failure bundle: %w", err)
	}
	if bundle.Version == 0 {
		bundle.Version = SchemaVersion
	}
	if bundle.Version != SchemaVersion {
		return FailureBundle{}, fmt.Errorf("unsupported failure bundle version %d", bundle.Version)
	}
	return bundle, nil
}

func goalContract(goal project.Goal) GoalContract {
	return GoalContract{
		Goal:            goal.Goal,
		NonGoals:        goal.NonGoals,
		MustPreserve:    goal.MustPreserve,
		DoneConditions:  goal.DoneConditions,
		SuspiciousPaths: goal.SuspiciousPaths,
	}
}

func classifySignals(signals []Signal) ([]Signal, []Signal) {
	var roots []Signal
	var symptoms []Signal
	for _, signal := range signals {
		if len(roots) < 5 && (signal.Confidence == "high" || len(roots) == 0) {
			roots = append(roots, signal)
			continue
		}
		if len(symptoms) < 10 {
			symptoms = append(symptoms, signal)
		}
	}
	return roots, symptoms
}

func goalWarnings(goal project.Goal, signals []Signal) []string {
	var warnings []string
	if strings.Contains(strings.ToLower(goal.Goal), "todo") {
		warnings = append(warnings, "goal.yml still contains a TODO goal; repair prompts may be less anchored")
	}
	if len(goal.NonGoals) == 0 {
		warnings = append(warnings, "goal.yml has no non_goals")
	}
	if len(goal.DoneConditions) == 0 {
		warnings = append(warnings, "goal.yml has no done_conditions")
	}

	for _, suspicious := range goal.SuspiciousPaths {
		suspicious = strings.Trim(suspicious, "/")
		if suspicious == "" {
			continue
		}
		for _, signal := range signals {
			file := strings.Trim(signal.File, "/")
			if file == suspicious || strings.HasPrefix(file, suspicious+"/") {
				warnings = append(warnings, fmt.Sprintf("failure signal points at suspicious path %q", suspicious))
				return warnings
			}
		}
	}
	return warnings
}

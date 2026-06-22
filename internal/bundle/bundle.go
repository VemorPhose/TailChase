package bundle

import (
	"fmt"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

type Compiler struct {
	Now    func() time.Time
	Safety project.SafetyConfig
}

func (c Compiler) Compile(run project.Run, goal project.Goal, normalized NormalizedEvidence) (FailureBundle, error) {
	if c.Now == nil {
		c.Now = time.Now
	}
	if err := goal.Validate(); err != nil {
		return FailureBundle{}, err
	}

	rootCandidates, symptoms := classifySignals(normalized.Signals)
	rootCandidates, collapsedRoots := compactSignalExcerpts(rootCandidates)
	symptoms, collapsedSymptoms := compactSignalExcerpts(symptoms)
	warnings := append([]string{}, normalized.Warnings...)
	warnings = append(warnings, GoalContractWarnings(goal, GoalCheckInput{Signals: normalized.Signals})...)
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

	artifacts := rawEvidenceArtifacts(run, normalized.Sources)
	artifacts = append(artifacts,
		Artifact{Name: project.ArtifactNormalizedEvidence, Path: run.RelativePath(run.ArtifactPath(project.NormalizedEvidenceName))},
		Artifact{Name: project.ArtifactFailureBundle, Path: run.RelativePath(run.ArtifactPath(project.FailureBundleName))},
	)
	budget := BudgetMetadata{
		RawEvidenceBytes:        rawEvidenceBytes(run, normalized.Sources),
		IncludedExcerptBytes:    includedExcerptBytes(rootCandidates, symptoms),
		RepeatedBlocksCollapsed: collapsedRoots + collapsedSymptoms,
	}
	failureBundle := FailureBundle{
		Version:             SchemaVersion,
		GeneratedAt:         c.Now().UTC(),
		Run:                 runMeta,
		Goal:                goalContract(goal),
		Sources:             normalized.Sources,
		AttemptContext:      context,
		Budget:              budget,
		RootErrorCandidates: rootCandidates,
		DownstreamSymptoms:  symptoms,
		Artifacts:           artifacts,
		Warnings:            warnings,
	}
	failureBundle.Budget.EstimatedPromptBytes = estimatePromptBytes(failureBundle)
	failureBundle.SafetyFindings = (SafetyEngine{Config: c.Safety}).Evaluate(SafetyInput{
		Bundle:  failureBundle,
		Goal:    goal,
		Signals: normalized.Signals,
	})
	return failureBundle, nil
}

func rawEvidenceArtifacts(run project.Run, sources []EvidenceSource) []Artifact {
	seen := map[string]bool{}
	var artifacts []Artifact
	add := func(name string, path string) {
		if name == "" || path == "" || seen[name] {
			return
		}
		seen[name] = true
		artifacts = append(artifacts, Artifact{Name: name, Path: path})
	}
	for _, source := range sources {
		switch source.Source {
		case "github_actions":
			add(project.ArtifactGitHubActionsLog, source.Path)
		case "gitlab_ci":
			add(project.ArtifactGitLabCILog, source.Path)
		}
	}
	if len(artifacts) == 0 {
		artifacts = append(artifacts, Artifact{Name: project.ArtifactGitHubActionsLog, Path: run.RelativePath(run.EvidencePath(project.GitHubActionsLogName))})
	}
	return artifacts
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
		ExpectedPaths:   goal.ExpectedPaths,
		SuspiciousPaths: goal.SuspiciousPaths,
		StopRules:       goal.StopRules,
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

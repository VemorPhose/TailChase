package report

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/loop"
	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

type Metrics struct {
	RawEvidenceBytes            int64
	IncludedExcerptBytes        int64
	RepeatedBlocksCollapsed     int
	EstimatedPromptBytes        int64
	RepeatedContextAvoidedBytes int64
	RootErrorCandidates         int
	DownstreamSymptoms          int
	SafetyFindings              int
	StopFindings                int
	Attempts                    int
	LastAttemptOutcome          string
	SteeringEvents              int
	RunLoopDecisions            int
}

type Summary struct {
	RunID          string
	Repository     string
	Source         string
	Goal           string
	Metrics        Metrics
	Attempts       []project.Attempt
	SafetyFindings []bundle.SafetyFinding
	Warnings       []string
}

type runLoopDecisionLog struct {
	Version   int             `yaml:"version"`
	Stopped   bool            `yaml:"stopped"`
	Reason    string          `yaml:"reason"`
	Decisions []loop.Decision `yaml:"decisions,omitempty"`
}

func Build(run project.Run) (Summary, error) {
	failureBundle, err := bundle.ReadFailureBundle(run)
	if err != nil {
		return Summary{}, err
	}
	history, err := run.ReadAttemptHistory()
	if err != nil {
		return Summary{}, err
	}
	eventLog, err := guard.ReadEventLog(run)
	if err != nil {
		return Summary{}, err
	}
	decisionLog, err := readRunLoopDecisionLog(run)
	if err != nil {
		return Summary{}, err
	}

	metrics := Metrics{
		RawEvidenceBytes:        failureBundle.Budget.RawEvidenceBytes,
		IncludedExcerptBytes:    failureBundle.Budget.IncludedExcerptBytes,
		RepeatedBlocksCollapsed: failureBundle.Budget.RepeatedBlocksCollapsed,
		EstimatedPromptBytes:    failureBundle.Budget.EstimatedPromptBytes,
		RootErrorCandidates:     len(failureBundle.RootErrorCandidates),
		DownstreamSymptoms:      len(failureBundle.DownstreamSymptoms),
		SafetyFindings:          len(failureBundle.SafetyFindings),
		Attempts:                len(history.Attempts),
		SteeringEvents:          len(eventLog.Events),
		RunLoopDecisions:        len(decisionLog.Decisions),
	}
	metrics.RepeatedContextAvoidedBytes = maxInt64(0, metrics.RawEvidenceBytes-metrics.IncludedExcerptBytes)
	for _, finding := range failureBundle.SafetyFindings {
		if finding.Decision == bundle.SafetyDecisionStop {
			metrics.StopFindings++
		}
	}
	if len(history.Attempts) > 0 {
		metrics.LastAttemptOutcome = history.Attempts[len(history.Attempts)-1].Outcome
	}

	return Summary{
		RunID:          firstNonEmpty(failureBundle.Run.RunID, run.ID),
		Repository:     failureBundle.Run.Repository,
		Source:         failureBundle.Run.Source,
		Goal:           failureBundle.Goal.Goal,
		Metrics:        metrics,
		Attempts:       append([]project.Attempt(nil), history.Attempts...),
		SafetyFindings: append([]bundle.SafetyFinding(nil), failureBundle.SafetyFindings...),
		Warnings:       append([]string(nil), failureBundle.Warnings...),
	}, nil
}

func Write(run project.Run) (Summary, error) {
	summary, err := Build(run)
	if err != nil {
		return Summary{}, err
	}
	if err := run.WriteArtifactFile(project.ReportName, project.ArtifactReport, "report", []byte(RenderMarkdown(summary))); err != nil {
		return Summary{}, err
	}
	return summary, nil
}

func RenderMarkdown(summary Summary) string {
	var out bytes.Buffer
	fmt.Fprintln(&out, "# Tailchase Run Report")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "- Run: `%s`\n", summary.RunID)
	writeOptionalLine(&out, "Repository", summary.Repository)
	writeOptionalLine(&out, "Source", summary.Source)
	writeOptionalLine(&out, "Goal", summary.Goal)
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## Evidence Reduction")
	fmt.Fprintf(&out, "- Raw evidence bytes: %d\n", summary.Metrics.RawEvidenceBytes)
	fmt.Fprintf(&out, "- Included excerpt bytes: %d\n", summary.Metrics.IncludedExcerptBytes)
	fmt.Fprintf(&out, "- Repeated context avoided bytes: %d\n", summary.Metrics.RepeatedContextAvoidedBytes)
	fmt.Fprintf(&out, "- Repeated blocks collapsed: %d\n", summary.Metrics.RepeatedBlocksCollapsed)
	fmt.Fprintf(&out, "- Estimated prompt bytes: %d\n", summary.Metrics.EstimatedPromptBytes)
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## Failure Signals")
	fmt.Fprintf(&out, "- Root error candidates: %d\n", summary.Metrics.RootErrorCandidates)
	fmt.Fprintf(&out, "- Downstream symptoms: %d\n", summary.Metrics.DownstreamSymptoms)
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## Safety")
	fmt.Fprintf(&out, "- Safety findings: %d\n", summary.Metrics.SafetyFindings)
	fmt.Fprintf(&out, "- Stop findings: %d\n", summary.Metrics.StopFindings)
	for _, finding := range summary.SafetyFindings {
		fmt.Fprintf(&out, "- `%s` %s: %s", finding.Rule, finding.Decision, finding.Message)
		if finding.Path != "" {
			fmt.Fprintf(&out, " (`%s`)", finding.Path)
		}
		fmt.Fprintln(&out)
	}
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## Attempts")
	fmt.Fprintf(&out, "- Attempts recorded: %d\n", summary.Metrics.Attempts)
	if summary.Metrics.LastAttemptOutcome != "" {
		fmt.Fprintf(&out, "- Last outcome: %s\n", summary.Metrics.LastAttemptOutcome)
	}
	for _, attempt := range summary.Attempts {
		fmt.Fprintf(&out, "- Attempt %d: %s", attempt.Number, attempt.Outcome)
		if len(attempt.RootErrorCandidates) > 0 {
			fmt.Fprintf(&out, " `%s`", attempt.RootErrorCandidates[0])
		}
		fmt.Fprintln(&out)
	}
	fmt.Fprintln(&out)

	fmt.Fprintln(&out, "## Steering")
	fmt.Fprintf(&out, "- Steering events: %d\n", summary.Metrics.SteeringEvents)
	fmt.Fprintf(&out, "- Run-loop decisions: %d\n", summary.Metrics.RunLoopDecisions)

	if len(summary.Warnings) > 0 {
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "## Warnings")
		for _, warning := range summary.Warnings {
			fmt.Fprintf(&out, "- %s\n", warning)
		}
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func readRunLoopDecisionLog(run project.Run) (runLoopDecisionLog, error) {
	data, err := run.ReadArtifactFile(project.RunLoopDecisionsName)
	if err != nil {
		if strings.Contains(err.Error(), "is missing") {
			return runLoopDecisionLog{Version: 1}, nil
		}
		return runLoopDecisionLog{}, err
	}
	var log runLoopDecisionLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return runLoopDecisionLog{}, fmt.Errorf("parse run-loop decisions: %w", err)
	}
	if log.Version == 0 {
		log.Version = 1
	}
	if log.Version != 1 {
		return runLoopDecisionLog{}, fmt.Errorf("unsupported run-loop decisions version %d", log.Version)
	}
	return log, nil
}

func writeOptionalLine(out *bytes.Buffer, label string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(out, "- %s: %s\n", label, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

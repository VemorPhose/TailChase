package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
	reportpkg "github.com/VemorPhose/TailChase/internal/report"
)

func TestReportBuildsMetricsFromStoredArtifacts(t *testing.T) {
	_, run := writeReportFixture(t)

	summary, err := reportpkg.Write(run)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if summary.Metrics.RawEvidenceBytes != 1000 || summary.Metrics.IncludedExcerptBytes != 250 || summary.Metrics.RepeatedContextAvoidedBytes != 750 {
		t.Fatalf("metrics = %#v, want raw/included/avoided 1000/250/750", summary.Metrics)
	}
	if summary.Metrics.StopFindings != 1 || summary.Metrics.Attempts != 1 || summary.Metrics.SteeringEvents != 1 || summary.Metrics.RunLoopDecisions != 2 {
		t.Fatalf("metrics = %#v, want stop/attempt/event/decision counts", summary.Metrics)
	}

	data, err := run.ReadArtifactFile(project.ReportName)
	if err != nil {
		t.Fatalf("ReadArtifactFile(report) error = %v", err)
	}
	report := string(data)
	for _, want := range []string{"# Tailchase Run Report", "Repeated context avoided bytes: 750", "Stop findings: 1", "Attempt 1: failed"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}

	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if !hasArtifact(meta.Artifacts, project.ArtifactReport) {
		t.Fatalf("metadata artifacts = %#v, want report artifact", meta.Artifacts)
	}
}

func TestCostReportCommandWritesReport(t *testing.T) {
	root, run := writeReportFixture(t)
	t.Chdir(root)

	stdout, _, err := runTailchase(t, "cost", "report", "--run", run.ID)
	if err != nil {
		t.Fatalf("tailchase cost report error = %v", err)
	}
	if !strings.Contains(stdout, "Repeated context avoided bytes: 750") {
		t.Fatalf("stdout = %q, want repeated context metric", stdout)
	}
	data, err := run.ReadArtifactFile(project.ReportName)
	if err != nil {
		t.Fatalf("ReadArtifactFile(report) error = %v", err)
	}
	if !strings.Contains(string(data), "Evidence Reduction") {
		t.Fatalf("report missing Evidence Reduction:\n%s", string(data))
	}
}

func writeReportFixture(t *testing.T) (string, project.Run) {
	t.Helper()

	root := t.TempDir()
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFile(t, run.ArtifactPath(project.FailureBundleName), `version: 1
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
goal:
  goal: Fix CI
budget:
  raw_evidence_bytes: 1000
  included_excerpt_bytes: 250
  repeated_blocks_collapsed: 3
  estimated_prompt_bytes: 500
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    confidence: high
downstream_symptoms:
  - type: generic_failure
    source: github_actions
    message: exit status 1
    confidence: medium
safety_findings:
  - rule: repeated_root_failure
    decision: stop
    message: same root error seen before
warnings:
  - same root error seen before
artifacts: []
`)
	if _, err := run.AppendAttempt(project.Attempt{
		RootErrorCandidates: []string{"undefined: Handler"},
		Outcome:             "failed",
		CreatedAt:           time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("AppendAttempt() error = %v", err)
	}
	if _, err := guard.AppendEvent(run, guard.Event{
		CreatedAt: time.Date(2026, 6, 22, 10, 5, 0, 0, time.UTC),
		Type:      "guard_check",
		Message:   "guard produced 1 finding(s)",
		Findings:  []guard.Finding{{Rule: "known_failure_repeated", Decision: "warn", Message: "known failure repeated"}},
	}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}
	writeFile(t, run.ArtifactPath(project.RunLoopDecisionsName), `version: 1
stopped: true
reason: repeated failure
decisions:
  - attempt: 1
    prompt: .tailchase/runs/12345/repair-prompt.md
    bundle: .tailchase/runs/12345/failure-bundle.yml
    exit_code: 1
    decision: continue
    reason: collect new evidence and generate delta context
    created_at: "2026-06-22T10:10:00Z"
  - attempt: 2
    prompt: .tailchase/runs/12345/repair-prompt.md
    bundle: .tailchase/runs/12345/failure-bundle.yml
    exit_code: 1
    decision: stop
    reason: repeated failure
    created_at: "2026-06-22T10:12:00Z"
`)
	return root, run
}

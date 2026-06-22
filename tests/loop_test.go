package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/loop"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestAssistedLoopStopsOnRepeatedFailure(t *testing.T) {
	agentCalls := 0
	collectorCalls := 0
	result, err := loop.Run(context.Background(), loop.Options{
		MaxAttempts: 3,
		Agent: loopAgentFunc(func(ctx context.Context, attempt int, prompt string) (loop.AttemptResult, error) {
			agentCalls++
			return loop.AttemptResult{Output: "same failure\n", ExitCode: 1}, nil
		}),
		Collector: loopCollectorFunc(func(ctx context.Context, attempt int) (loop.EvidenceResult, error) {
			collectorCalls++
			return loop.EvidenceResult{PromptPath: "repair-prompt.md", BundlePath: "failure-bundle.yml", Failure: "same failure"}, nil
		}),
		Now: func() time.Time { return time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Reason != "repeated failure" || len(result.Decisions) != 2 || agentCalls != 2 || collectorCalls != 2 {
		t.Fatalf("result = %#v agent calls = %d collector calls = %d", result, agentCalls, collectorCalls)
	}
}

func TestAssistedLoopEnforcesMaxAttempts(t *testing.T) {
	result, err := loop.Run(context.Background(), loop.Options{
		MaxAttempts: 2,
		Agent: loopAgentFunc(func(ctx context.Context, attempt int, prompt string) (loop.AttemptResult, error) {
			return loop.AttemptResult{Output: "failure " + prompt, ExitCode: 1}, nil
		}),
		Collector: loopCollectorFunc(func(ctx context.Context, attempt int) (loop.EvidenceResult, error) {
			return loop.EvidenceResult{PromptPath: "prompt-" + strconv.Itoa(attempt) + ".md", BundlePath: "bundle.yml", Failure: "failure " + strconv.Itoa(attempt)}, nil
		}),
	})
	if err != nil {
		t.Fatalf("Run(max attempts) error = %v", err)
	}
	if result.Reason != "max attempts reached" || len(result.Decisions) != 2 {
		t.Fatalf("result = %#v, want max attempts", result)
	}
}

func TestRunLoopCommandRecordsEvent(t *testing.T) {
	root, run := writeMCPFixture(t)
	t.Chdir(root)

	stdout, _, err := runTailchase(t, "run-loop", "--run", run.ID, "--agent", "codex", "--agent-command", "false", "--max-attempts", "1")
	if err != nil {
		t.Fatalf("tailchase run-loop error = %v", err)
	}
	if !strings.Contains(stdout, "max attempts reached") {
		t.Fatalf("stdout = %q, want max attempts", stdout)
	}
	decisionData, err := run.ReadArtifactFile(project.RunLoopDecisionsName)
	if err != nil {
		t.Fatalf("ReadArtifactFile(run loop decisions) error = %v", err)
	}
	if !strings.Contains(string(decisionData), "max attempts reached") {
		t.Fatalf("run loop decisions missing stop reason:\n%s", string(decisionData))
	}
	data, err := run.ReadArtifactFile(project.SteeringEventsName)
	if err != nil {
		t.Fatalf("ReadArtifactFile(steering events) error = %v", err)
	}
	if !strings.Contains(string(data), "assisted_repair_loop") {
		t.Fatalf("steering events missing loop:\n%s", string(data))
	}
}

type loopAgentFunc func(context.Context, int, string) (loop.AttemptResult, error)

func (f loopAgentFunc) RunAttempt(ctx context.Context, attempt int, prompt string) (loop.AttemptResult, error) {
	return f(ctx, attempt, prompt)
}

type loopCollectorFunc func(context.Context, int) (loop.EvidenceResult, error)

func (f loopCollectorFunc) Collect(ctx context.Context, attempt int) (loop.EvidenceResult, error) {
	return f(ctx, attempt)
}

package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/VemorPhose/TailChase/internal/wrapper"
)

func TestManagedWrapperStopsOnRepeatedFailure(t *testing.T) {
	calls := 0
	result, err := wrapper.Run(context.Background(), wrapper.Options{
		MaxAttempts: 3,
		Runner: wrapper.RunnerFunc(func(ctx context.Context, attempt int) (wrapper.CommandResult, error) {
			calls++
			return wrapper.CommandResult{Command: "fake-agent", Output: "same failure\n", ExitCode: 1}, nil
		}),
		Now: func() time.Time { return time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Stopped || result.Reason != "repeated failure" || len(result.Attempts) != 2 || calls != 2 {
		t.Fatalf("result = %#v calls = %d, want repeated failure stop", result, calls)
	}
}

func TestManagedWrapperStopsOnSuccess(t *testing.T) {
	result, err := wrapper.Run(context.Background(), wrapper.Options{
		MaxAttempts: 3,
		Runner: wrapper.RunnerFunc(func(ctx context.Context, attempt int) (wrapper.CommandResult, error) {
			return wrapper.CommandResult{Command: "fake-agent", Output: "ok\n", ExitCode: 0}, nil
		}),
	})
	if err != nil {
		t.Fatalf("Run(success) error = %v", err)
	}
	if result.Reason != "agent command succeeded" || len(result.Attempts) != 1 {
		t.Fatalf("result = %#v, want success stop", result)
	}
}

func TestGuardAgentModeRequiresCommand(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeGoal(t, root)
	if _, err := project.NewStore(root).EnsureRun("12345"); err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	_, _, err := runTailchase(t, "guard", "--run", "12345", "--agent", "codex")
	if err == nil || !strings.Contains(err.Error(), "--agent-command is required") {
		t.Fatalf("error = %v, want missing agent command", err)
	}
}

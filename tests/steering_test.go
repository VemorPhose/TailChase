package tests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/adapter"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/VemorPhose/TailChase/internal/steering"
)

func TestCheckpointSteeringDeliversThroughCapableAdapter(t *testing.T) {
	run := mustRun(t)
	fake := &fakeSteeringAdapter{}

	delivery, err := steering.Deliver(context.Background(), steering.Options{
		Run: run,
		AdapterInfo: adapter.Adapter{
			Target:       "fake",
			Capabilities: []adapter.CapabilityLevel{adapter.CapabilityCheckpoint},
			Fallback:     adapter.CapabilityArtifact,
		},
		Adapter: fake,
		Message: steering.Message{
			Checkpoint: steering.CheckpointCommandCompletion,
			Reason:     "tests completed",
			Body:       "Re-run the failing package.",
		},
		Now: func() time.Time { return time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	if !delivery.Sent || fake.delivered.Body != "Re-run the failing package." {
		t.Fatalf("delivery = %#v fake = %#v, want delivered message", delivery, fake.delivered)
	}
	log, err := guard.ReadEventLog(run)
	if err != nil {
		t.Fatalf("ReadEventLog() error = %v", err)
	}
	if len(log.Events) != 1 || log.Events[0].Type != "checkpoint_steering" {
		t.Fatalf("events = %#v, want checkpoint steering event", log.Events)
	}
}

func TestCheckpointSteeringFallsBackForUnsupportedAdapter(t *testing.T) {
	run := mustRun(t)
	copilot, err := adapter.Discover("copilot", nil)
	if err != nil {
		t.Fatalf("Discover(copilot) error = %v", err)
	}
	fake := &fakeSteeringAdapter{}

	delivery, err := steering.Deliver(context.Background(), steering.Options{
		Run:         run,
		AdapterInfo: copilot,
		Adapter:     fake,
		Message: steering.Message{
			Checkpoint: steering.CheckpointFileWrite,
			Reason:     "file changed",
			Body:       "Inspect the generated fallback.",
		},
		Now: func() time.Time { return time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Deliver(fallback) error = %v", err)
	}
	if delivery.Sent || fake.called || delivery.Path == "" {
		t.Fatalf("delivery = %#v fake called = %t, want artifact fallback only", delivery, fake.called)
	}
	data, err := os.ReadFile(run.AbsolutePath(delivery.Path))
	if err != nil {
		t.Fatalf("ReadFile(fallback) error = %v", err)
	}
	if !strings.Contains(string(data), "Inspect the generated fallback") {
		t.Fatalf("fallback content = %s", string(data))
	}
}

func TestSteerCommandWritesFallback(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeConfig(t, root, "file")
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	stdout, _, err := runTailchase(t, "steer", "--run", "12345", "--target", "copilot", "--checkpoint", "stop_event", "--message", "Stop and ask for help.")
	if err != nil {
		t.Fatalf("tailchase steer error = %v", err)
	}
	if !strings.Contains(stdout, project.SteeringDirName) {
		t.Fatalf("stdout = %q, want steering fallback path", stdout)
	}
	if _, err := os.Stat(run.ArtifactPath(project.SteeringEventsName)); err != nil {
		t.Fatalf("steering events missing: %v", err)
	}
}

type fakeSteeringAdapter struct {
	called    bool
	delivered steering.Message
}

func (f *fakeSteeringAdapter) Deliver(ctx context.Context, message steering.Message) error {
	f.called = true
	f.delivered = message
	return nil
}

package steering

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/adapter"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
)

type Checkpoint string

const (
	CheckpointCommandCompletion Checkpoint = "command_completion"
	CheckpointFileWrite         Checkpoint = "file_write"
	CheckpointPermissionPrompt  Checkpoint = "permission_prompt"
	CheckpointStopEvent         Checkpoint = "stop_event"
)

type Message struct {
	Checkpoint Checkpoint `yaml:"checkpoint"`
	Reason     string     `yaml:"reason"`
	Body       string     `yaml:"body"`
}

type Delivery struct {
	Target string `yaml:"target"`
	Mode   string `yaml:"mode"`
	Path   string `yaml:"path,omitempty"`
	Sent   bool   `yaml:"sent"`
	Reason string `yaml:"reason"`
}

type Adapter interface {
	Deliver(ctx context.Context, message Message) error
}

type Options struct {
	Run         project.Run
	AdapterInfo adapter.Adapter
	Adapter     Adapter
	Message     Message
	Now         func() time.Time
}

func Deliver(ctx context.Context, opts Options) (Delivery, error) {
	if strings.TrimSpace(opts.Message.Body) == "" {
		return Delivery{}, fmt.Errorf("steering message body is required")
	}
	if opts.Message.Checkpoint == "" {
		return Delivery{}, fmt.Errorf("checkpoint is required")
	}
	if !ValidCheckpoint(opts.Message.Checkpoint) {
		return Delivery{}, fmt.Errorf("unsupported checkpoint %q", opts.Message.Checkpoint)
	}
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}

	if err := adapter.RequireCapability(opts.AdapterInfo, adapter.CapabilityCheckpoint); err == nil && opts.Adapter != nil {
		if err := opts.Adapter.Deliver(ctx, opts.Message); err != nil {
			return Delivery{}, err
		}
		delivery := Delivery{Target: opts.AdapterInfo.Target, Mode: "checkpoint", Sent: true, Reason: opts.Message.Reason}
		return delivery, recordDelivery(opts.Run, now, opts.Message, delivery)
	}

	fileName := filepath.Join(project.SteeringDirName, fmt.Sprintf("%d-%s.md", now.Unix(), opts.Message.Checkpoint))
	content := renderFallback(opts.AdapterInfo, opts.Message)
	if err := opts.Run.WriteArtifactFile(fileName, project.ArtifactSteeringMessage, "steering_message", []byte(content)); err != nil {
		return Delivery{}, err
	}
	delivery := Delivery{
		Target: opts.AdapterInfo.Target,
		Mode:   "artifact",
		Path:   opts.Run.RelativePath(opts.Run.ArtifactPath(fileName)),
		Sent:   false,
		Reason: "adapter does not support checkpoint delivery; wrote artifact fallback",
	}
	return delivery, recordDelivery(opts.Run, now, opts.Message, delivery)
}

func ValidCheckpoint(checkpoint Checkpoint) bool {
	switch checkpoint {
	case CheckpointCommandCompletion, CheckpointFileWrite, CheckpointPermissionPrompt, CheckpointStopEvent:
		return true
	default:
		return false
	}
}

func recordDelivery(run project.Run, now time.Time, message Message, delivery Delivery) error {
	event := guard.Event{
		CreatedAt: now,
		Type:      "checkpoint_steering",
		Message:   fmt.Sprintf("%s steering for %s via %s", message.Checkpoint, delivery.Target, delivery.Mode),
		Commands:  []string{string(message.Checkpoint)},
		Findings: []guard.Finding{{
			Rule:     "checkpoint_steering",
			Decision: "warn",
			Message:  delivery.Reason,
			Path:     delivery.Path,
		}},
	}
	_, err := guard.AppendEvent(run, event)
	return err
}

func renderFallback(adapterInfo adapter.Adapter, message Message) string {
	return fmt.Sprintf(`# Tailchase Checkpoint Steering

Target: %s
Checkpoint: %s
Reason: %s

%s
`, adapterInfo.Target, message.Checkpoint, message.Reason, strings.TrimSpace(message.Body))
}

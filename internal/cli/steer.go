package cli

import (
	"fmt"
	"os"

	"github.com/VemorPhose/TailChase/internal/adapter"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/VemorPhose/TailChase/internal/steering"
	"github.com/spf13/cobra"
)

func newSteerCommand() *cobra.Command {
	var runID string
	var target string
	var checkpoint string
	var reason string
	var message string
	cmd := &cobra.Command{
		Use:   "steer",
		Short: "Record checkpoint steering or write a fallback prompt file",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runSteer(cmd, root, runID, target, checkpoint, reason, message)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to record steering against")
	cmd.Flags().StringVar(&target, "target", "generic", "Adapter target")
	cmd.Flags().StringVar(&checkpoint, "checkpoint", string(steering.CheckpointCommandCompletion), "Checkpoint type")
	cmd.Flags().StringVar(&reason, "reason", "manual checkpoint", "Reason for steering")
	cmd.Flags().StringVar(&message, "message", "", "Steering message body")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("message")
	return cmd
}

func runSteer(cmd *cobra.Command, root string, runID string, target string, checkpoint string, reason string, message string) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	adapterInfo, err := adapter.Discover(target, cfg.Adapters)
	if err != nil {
		return err
	}
	delivery, err := steering.Deliver(cmd.Context(), steering.Options{
		Run:         run,
		AdapterInfo: adapterInfo,
		Message: steering.Message{
			Checkpoint: steering.Checkpoint(checkpoint),
			Reason:     reason,
			Body:       message,
		},
	})
	if err != nil {
		return err
	}
	if delivery.Path != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Wrote steering fallback %s\n", delivery.Path)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Delivered checkpoint steering to %s\n", delivery.Target)
	return nil
}

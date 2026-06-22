package cli

import (
	"fmt"
	"os"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/exporter"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var runID string
	var target string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Write target-specific repair prompt files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runExport(cmd, root, runID, target)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	cmd.Flags().StringVar(&target, "target", "", "Export target: codex, claude-code, or copilot")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

func runExport(cmd *cobra.Command, root string, runID string, target string) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	failureBundle, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		return err
	}
	repairPrompt, err := run.ReadArtifactFile(project.RepairPromptName)
	if err != nil {
		return err
	}

	result, err := exporter.Write(run, target, failureBundle, string(repairPrompt))
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), result.Path)
	return nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPrepareCommand() *cobra.Command {
	var runID string
	var delta bool
	var exports []string
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Bundle evidence, write a repair prompt, and generate a report",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runPrepare(cmd, root, runID, delta, exports)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to prepare")
	cmd.Flags().BoolVar(&delta, "delta", false, "Render a delta repair prompt")
	cmd.Flags().StringSliceVar(&exports, "export", nil, "Optional export target: codex, claude-code, or copilot")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runPrepare(cmd *cobra.Command, root string, runID string, delta bool, exports []string) error {
	if err := runBundle(cmd, root, runID); err != nil {
		return err
	}
	if err := runPrompt(cmd, root, runID, delta); err != nil {
		return err
	}
	for _, target := range exports {
		if err := runExport(cmd, root, runID, target); err != nil {
			return err
		}
	}
	if err := runCostReport(cmd, root, runID); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Prepared run %s\n", runID)
	return nil
}

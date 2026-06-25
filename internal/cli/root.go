package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.28"

func Execute() error {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tailchase",
		Short: "Collect failure evidence and render repair prompts",
		Long:  "Tailchase is a local-first CLI for turning CI, local, runtime, and browser failure evidence into structured repair context.",
	}

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newCollectCommand())
	cmd.AddCommand(newCollectGitLabCommand())
	cmd.AddCommand(newCollectLocalCommand())
	cmd.AddCommand(newCollectReportsCommand())
	cmd.AddCommand(newCollectComposeCommand())
	cmd.AddCommand(newCollectPlaywrightCommand())
	cmd.AddCommand(newPrepareCommand())
	cmd.AddCommand(newCICommand())
	cmd.AddCommand(newBundleCommand())
	cmd.AddCommand(newPromptCommand())
	cmd.AddCommand(newExportCommand())
	cmd.AddCommand(newCommentCommand())
	cmd.AddCommand(newMCPCommand())
	cmd.AddCommand(newAdaptersCommand())
	cmd.AddCommand(newGuardCommand())
	cmd.AddCommand(newSteerCommand())
	cmd.AddCommand(newRunLoopCommand())
	cmd.AddCommand(newCostCommand())
	cmd.AddCommand(newTournamentCommand())
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the Tailchase version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	})

	return cmd
}

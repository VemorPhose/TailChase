package cli

import (
	"fmt"
	"os"

	"github.com/VemorPhose/TailChase/internal/tournament"
	"github.com/spf13/cobra"
)

func newTournamentCommand() *cobra.Command {
	var testCommand string
	cmd := &cobra.Command{
		Use:   "tournament <branch-a> <branch-b>",
		Short: "Compare two candidate repair branches",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runTournament(cmd, root, args[0], args[1], testCommand)
		},
	}
	cmd.Flags().StringVar(&testCommand, "test-command", "", "Command to run in each temporary candidate worktree")
	return cmd
}

func runTournament(cmd *cobra.Command, root string, branchA string, branchB string, testCommand string) error {
	result, err := tournament.WriteReport(cmd.Context(), tournament.Options{
		Root:        root,
		BranchA:     branchA,
		BranchB:     branchB,
		TestCommand: testCommand,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Winner: %s\n", result.Winner)
	fmt.Fprintf(cmd.OutOrStdout(), "Rationale: %s\n", result.Rationale)
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", result.ReportPath)
	return nil
}

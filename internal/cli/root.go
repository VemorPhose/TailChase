package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.5"

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
		Short: "Collect failed CI evidence and render repair prompts",
		Long:  "Tailchase is a local-first CLI for turning failed GitHub Actions evidence into a structured repair prompt.",
	}

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newCollectCommand())
	cmd.AddCommand(newBundleCommand())
	cmd.AddCommand(newPromptCommand())
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the Tailchase version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	})

	return cmd
}

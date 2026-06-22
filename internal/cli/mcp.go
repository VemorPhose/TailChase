package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/VemorPhose/TailChase/internal/mcpserver"
	"github.com/spf13/cobra"
)

func newMCPCommand() *cobra.Command {
	var runID string
	var listResources bool
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start a local MCP stdio server for Tailchase artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runMCP(cmd, root, runID, listResources)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to expose; defaults to latest run with a failure bundle")
	cmd.Flags().BoolVar(&listResources, "list-resources", false, "Print resource metadata and exit")
	return cmd
}

func runMCP(cmd *cobra.Command, root string, runID string, listResources bool) error {
	snapshot, err := mcpserver.BuildSnapshot(root, runID)
	if err != nil {
		return err
	}
	if listResources {
		data, err := json.MarshalIndent(snapshot.ResourceList(), "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	return mcpserver.Serve(cmd.Context(), snapshot, os.Stdin, cmd.OutOrStdout())
}

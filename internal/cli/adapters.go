package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/VemorPhose/TailChase/internal/adapter"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newAdaptersCommand() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "adapters",
		Short: "List agent adapter capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runAdapters(cmd, root, target)
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "Adapter target to inspect")
	return cmd
}

func runAdapters(cmd *cobra.Command, root string, target string) error {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	if strings.TrimSpace(target) != "" {
		adapterInfo, err := adapter.Discover(target, cfg.Adapters)
		if err != nil {
			return err
		}
		writeAdapter(cmd, adapterInfo)
		return nil
	}
	for _, adapterInfo := range adapter.List() {
		discovered, err := adapter.Discover(adapterInfo.Target, cfg.Adapters)
		if err != nil {
			return err
		}
		writeAdapter(cmd, discovered)
	}
	return nil
}

func writeAdapter(cmd *cobra.Command, adapterInfo adapter.Adapter) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", adapterInfo.DisplayName, adapterInfo.Target)
	fmt.Fprintf(cmd.OutOrStdout(), "  capabilities: %s\n", strings.Join(adapter.CapabilityNames(adapterInfo.Capabilities), ", "))
	fmt.Fprintf(cmd.OutOrStdout(), "  fallback: %s\n", adapterInfo.Fallback)
}

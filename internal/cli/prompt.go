package cli

import (
	"fmt"
	"os"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
	"github.com/spf13/cobra"
)

func newPromptCommand() *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Render repair-prompt.md from failure-bundle.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runPrompt(cmd, root, runID)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runPrompt(cmd *cobra.Command, root string, runID string) error {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	failureBundle, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		return err
	}

	result, err := (promptpkg.Generator{}).Generate(failureBundle, promptpkg.Options{SizeLimit: cfg.PromptSizeLimit})
	if err != nil {
		return err
	}
	if err := promptpkg.WriteRepairPrompt(run, result); err != nil {
		return err
	}

	promptPath := run.RelativePath(run.ArtifactPath(project.RepairPromptName))
	switch cfg.PromptTarget {
	case "stdout":
		fmt.Fprint(cmd.OutOrStdout(), result.Content)
		fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s\n", promptPath)
	case "file":
		fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", promptPath)
	default:
		return fmt.Errorf("unsupported prompt target %q", cfg.PromptTarget)
	}
	return nil
}

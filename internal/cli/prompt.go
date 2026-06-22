package cli

import (
	"fmt"
	"os"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	modelpkg "github.com/VemorPhose/TailChase/internal/model"
	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
	"github.com/spf13/cobra"
)

func newPromptCommand() *cobra.Command {
	var runID string
	var delta bool
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Render repair-prompt.md from failure-bundle.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runPrompt(cmd, root, runID, delta)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	cmd.Flags().BoolVar(&delta, "delta", false, "Render a compact prompt focused on changes since prior attempts")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runPrompt(cmd *cobra.Command, root string, runID string, delta bool) error {
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

	opts := promptpkg.Options{SizeLimit: cfg.PromptSizeLimit, Delta: delta}
	if delta {
		history, err := run.ReadAttemptHistory()
		if err != nil {
			return err
		}
		opts.AttemptHistory = history
	}
	result, err := generatePrompt(cmd, cfg, failureBundle, opts)
	if err != nil {
		return err
	}
	if err := promptpkg.WriteRepairPrompt(run, result); err != nil {
		return err
	}
	if _, err := run.AppendAttempt(project.Attempt{
		BundlePath:          run.RelativePath(run.ArtifactPath(project.FailureBundleName)),
		PromptPath:          run.RelativePath(run.ArtifactPath(project.RepairPromptName)),
		RootErrorCandidates: rootCandidateMessages(failureBundle),
	}); err != nil {
		return err
	}

	promptPath := run.RelativePath(run.ArtifactPath(project.RepairPromptName))
	switch cfg.PromptTarget {
	case "stdout":
		fmt.Fprint(cmd.OutOrStdout(), result.Content)
		fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s\n", promptPath)
	case "file":
		fmt.Fprintln(cmd.OutOrStdout(), promptPath)
	default:
		return fmt.Errorf("unsupported prompt target %q", cfg.PromptTarget)
	}
	return nil
}

func generatePrompt(cmd *cobra.Command, cfg project.Config, failureBundle bundlepkg.FailureBundle, opts promptpkg.Options) (promptpkg.Result, error) {
	if cfg.Prompt.Mode != "model" {
		return (promptpkg.Generator{}).Generate(failureBundle, opts)
	}
	provider, err := modelpkg.NewOpenAICompatibleProvider(cfg.Model)
	if err != nil {
		return promptpkg.Result{}, err
	}
	return (promptpkg.ModelGenerator{Provider: provider}).Generate(cmd.Context(), failureBundle, cfg.Model, opts)
}

func rootCandidateMessages(failureBundle bundlepkg.FailureBundle) []string {
	messages := make([]string, 0, len(failureBundle.RootErrorCandidates))
	for _, signal := range failureBundle.RootErrorCandidates {
		if signal.Message != "" {
			messages = append(messages, signal.Message)
		}
	}
	return messages
}

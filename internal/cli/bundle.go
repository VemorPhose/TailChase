package cli

import (
	"fmt"
	"os"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newBundleCommand() *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Normalize evidence and write failure-bundle.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runBundle(cmd, root, runID)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runBundle(cmd *cobra.Command, root string, runID string) error {
	store := project.NewStore(root)
	run, err := store.OpenRun(runID)
	if err != nil {
		return err
	}

	goal, err := project.LoadGoal(root)
	if err != nil {
		return err
	}

	normalized, err := (bundlepkg.Normalizer{}).NormalizeRun(run)
	if err != nil {
		return err
	}
	if err := bundlepkg.WriteNormalizedEvidence(run, normalized); err != nil {
		return err
	}

	failureBundle, err := (bundlepkg.Compiler{}).Compile(run, goal, normalized)
	if err != nil {
		return err
	}
	if err := bundlepkg.WriteFailureBundle(run, failureBundle); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(run.ArtifactPath(project.NormalizedEvidenceName)))
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(run.ArtifactPath(project.FailureBundleName)))
	return nil
}

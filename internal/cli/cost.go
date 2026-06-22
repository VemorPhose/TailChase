package cli

import (
	"fmt"
	"os"

	"github.com/VemorPhose/TailChase/internal/project"
	reportpkg "github.com/VemorPhose/TailChase/internal/report"
	"github.com/spf13/cobra"
)

func newCostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Report context cost and evidence reduction metrics",
	}
	cmd.AddCommand(newCostReportCommand())
	return cmd
}

func newCostReportCommand() *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Write report.md for a run",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCostReport(cmd, root, runID)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to report on")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCostReport(cmd *cobra.Command, root string, runID string) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	summary, err := reportpkg.Write(run)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(run.ArtifactPath(project.ReportName)))
	fmt.Fprintf(cmd.OutOrStdout(), "Raw evidence bytes: %d\n", summary.Metrics.RawEvidenceBytes)
	fmt.Fprintf(cmd.OutOrStdout(), "Included excerpt bytes: %d\n", summary.Metrics.IncludedExcerptBytes)
	fmt.Fprintf(cmd.OutOrStdout(), "Repeated context avoided bytes: %d\n", summary.Metrics.RepeatedContextAvoidedBytes)
	fmt.Fprintf(cmd.OutOrStdout(), "Safety findings: %d\n", summary.Metrics.SafetyFindings)
	return nil
}

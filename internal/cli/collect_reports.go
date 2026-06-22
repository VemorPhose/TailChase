package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectReportsCommand() *cobra.Command {
	var runID string
	var globs []string
	cmd := &cobra.Command{
		Use:   "collect-reports",
		Short: "Import JUnit-style test report XML files into the run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollectReports(cmd, root, runID, globs)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "local run ID")
	cmd.Flags().StringArrayVar(&globs, "glob", nil, "report path glob; may be repeated")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCollectReports(cmd *cobra.Command, root string, runID string, globs []string) error {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	if len(globs) == 0 {
		globs = cfg.ReportGlobs
	}
	run, err := project.NewStore(root).EnsureRun(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	if len(globs) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: no report globs configured")
		return nil
	}

	reportDir := run.EvidencePath(project.TestReportsDirName)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	written := 0
	for _, pattern := range globs {
		matches, err := filepath.Glob(resolveGlob(root, pattern))
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: report glob matched no files: %s\n", pattern)
			continue
		}
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return err
			}
			written++
			dest := filepath.Join(reportDir, fmt.Sprintf("%02d-%s", written, filepath.Base(match)))
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return err
			}
			if err := run.RecordArtifact(fmt.Sprintf("%s_%d", project.ArtifactTestReport, written), "junit_report", dest, time.Now().UTC()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(dest))
		}
	}
	return nil
}

func resolveGlob(root string, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if filepath.IsAbs(pattern) {
		return pattern
	}
	return filepath.Join(root, pattern)
}

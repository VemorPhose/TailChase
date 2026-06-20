package cli

import (
	"fmt"
	"os"
	"strings"

	collectpkg "github.com/VemorPhose/TailChase/internal/collect"
	githubpkg "github.com/VemorPhose/TailChase/internal/github"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectCommand() *cobra.Command {
	var runID string
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Fetch failed GitHub Actions logs into the local run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollect(cmd, root, runID, repoFlag)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "GitHub repository owner/name")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCollect(cmd *cobra.Command, root string, runID string, repoFlag string) error {
	runID = strings.TrimSpace(runID)
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	numericRunID, err := collectpkg.ParseRunID(runID)
	if err != nil {
		return err
	}
	if err := project.ValidateRunID(runID); err != nil {
		return err
	}

	repo, repoSource, err := githubpkg.ResolveRepository(root, repoFlag, cfg.GitHub.Repo)
	if err != nil {
		return err
	}

	run, err := project.NewStore(root).EnsureRun(runID)
	if err != nil {
		return err
	}

	token := githubpkg.TokenFromEnv()
	if token == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: GITHUB_TOKEN/GH_TOKEN is not set; private repositories and some logs may be inaccessible.")
	}

	client := githubpkg.NewClient(token)
	result, err := collectpkg.NewGitHubActionsCollector(client).Collect(cmd.Context(), run, collectpkg.GitHubActionsOptions{
		Owner:             repo.Owner,
		Repo:              repo.Name,
		RunID:             numericRunID,
		FailedJobsOnly:    cfg.FailedJobsOnly,
		MaxLogLinesPerJob: cfg.MaxLogLinesPerJob,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Collected %d GitHub Actions job(s) from %s run %d (%s)\n", len(result.Jobs), repo.String(), numericRunID, repoSource)
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", result.EvidencePath)
	for _, warning := range result.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning)
	}
	return nil
}

package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	collectpkg "github.com/VemorPhose/TailChase/internal/collect"
	gitlabpkg "github.com/VemorPhose/TailChase/internal/gitlab"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectGitLabCommand() *cobra.Command {
	var runID string
	var projectFlag string
	var baseURLFlag string
	cmd := &cobra.Command{
		Use:   "collect-gitlab",
		Short: "Fetch failed GitLab CI logs into the local run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollectGitLab(cmd, root, runID, projectFlag, baseURLFlag)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitLab pipeline ID")
	cmd.Flags().StringVar(&projectFlag, "project", "", "GitLab project path, such as group/project")
	cmd.Flags().StringVar(&baseURLFlag, "base-url", "", "GitLab base URL")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCollectGitLab(cmd *cobra.Command, root string, runID string, projectFlag string, baseURLFlag string) error {
	runID = strings.TrimSpace(runID)
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	pipelineID, err := strconv.ParseInt(runID, 10, 64)
	if err != nil || pipelineID <= 0 {
		return fmt.Errorf("run ID %q must be a positive GitLab pipeline ID", runID)
	}
	if err := project.ValidateRunID(runID); err != nil {
		return err
	}

	projectPath := strings.TrimSpace(projectFlag)
	if projectPath == "" {
		projectPath = strings.TrimSpace(cfg.GitLab.Project)
	}
	if projectPath == "" {
		return fmt.Errorf("gitlab project is required; pass --project group/name or set gitlab.project")
	}
	baseURL := strings.TrimSpace(baseURLFlag)
	if baseURL == "" {
		baseURL = cfg.GitLab.BaseURL
	}
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	token := gitlabpkg.TokenFromEnv()
	if token == "" {
		return fmt.Errorf("GITLAB_TOKEN is required to collect GitLab CI logs")
	}

	run, err := project.NewStore(root).EnsureRun(runID)
	if err != nil {
		return err
	}
	client := gitlabpkg.NewClient(baseURL, token)
	result, err := collectpkg.NewGitLabCICollector(client).Collect(cmd.Context(), run, collectpkg.GitLabCIOptions{
		Project:           projectPath,
		PipelineID:        pipelineID,
		FailedJobsOnly:    cfg.FailedJobsOnly,
		MaxLogLinesPerJob: cfg.MaxLogLinesPerJob,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Collected %d GitLab CI job(s) from %s pipeline %d\n", len(result.Jobs), projectPath, pipelineID)
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", result.EvidencePath)
	for _, warning := range result.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning)
	}
	return nil
}

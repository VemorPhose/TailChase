package cli

import (
	"fmt"
	"os"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/commenter"
	githubpkg "github.com/VemorPhose/TailChase/internal/github"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCommentCommand() *cobra.Command {
	var runID string
	var repoFlag string
	var prNumber int
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Post or preview Tailchase repair context as a GitHub PR comment",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runComment(cmd, root, runID, repoFlag, prNumber, dryRun)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "GitHub Actions run ID")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "GitHub repository owner/name")
	cmd.Flags().IntVar(&prNumber, "pr", 0, "GitHub pull request number")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the comment body without posting")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("pr")
	return cmd
}

func runComment(cmd *cobra.Command, root string, runID string, repoFlag string, prNumber int, dryRun bool) error {
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be greater than zero")
	}
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	failureBundle, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		return err
	}
	repairPrompt, err := run.ReadArtifactFile(project.RepairPromptName)
	if err != nil {
		return err
	}
	body, err := commenter.BuildBody(commenter.BodyOptions{
		Run:          run,
		Bundle:       failureBundle,
		RepairPrompt: string(repairPrompt),
	})
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Fprint(cmd.OutOrStdout(), body)
		return nil
	}

	token := githubpkg.TokenFromEnv()
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN or GH_TOKEN is required to post PR comments")
	}
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	repo, repoSource, err := githubpkg.ResolveRepository(root, repoFlag, cfg.GitHub.Repo)
	if err != nil {
		return err
	}
	client := githubpkg.NewClient(token)
	if err := githubpkg.NewPullRequestCommenter(client).Post(cmd.Context(), repo, prNumber, body); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Posted Tailchase comment to %s#%d (%s)\n", repo.String(), prNumber, repoSource)
	return nil
}

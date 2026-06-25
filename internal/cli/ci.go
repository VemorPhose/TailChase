package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/ciwatch"
	githubpkg "github.com/VemorPhose/TailChase/internal/github"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

type ciOptions struct {
	repo       string
	branch     string
	sha        string
	timeout    string
	interval   string
	delta      bool
	exports    []string
	pushArgs   []string
	runGitPush bool
}

func newCICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Watch GitHub Actions and prepare Tailchase artifacts",
	}
	cmd.AddCommand(newCIWatchCommand(false))
	cmd.AddCommand(newCIWatchCommand(true))
	return cmd
}

func newCIWatchCommand(push bool) *cobra.Command {
	var opts ciOptions
	use := "watch"
	short := "Wait for GitHub Actions to finish, then collect failed logs"
	if push {
		use = "push [git push args...]"
		short = "Run git push, wait for GitHub Actions, then collect failed logs"
		opts.runGitPush = true
	}
	argsRule := cobra.NoArgs
	if push {
		argsRule = cobra.ArbitraryArgs
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  argsRule,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			opts.pushArgs = args
			return runCIWatch(cmd, root, opts)
		},
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository owner/name; defaults to config or git remote origin")
	cmd.Flags().StringVar(&opts.branch, "branch", "", "Branch to watch; defaults to current branch")
	cmd.Flags().StringVar(&opts.sha, "sha", "", "Commit SHA to watch; defaults to current HEAD")
	cmd.Flags().StringVar(&opts.timeout, "timeout", "30m", "Maximum time to wait for CI")
	cmd.Flags().StringVar(&opts.interval, "interval", "15s", "Polling interval")
	cmd.Flags().BoolVar(&opts.delta, "delta", true, "Render a delta repair prompt when failure evidence is collected")
	cmd.Flags().StringSliceVar(&opts.exports, "export", nil, "Optional export target after CI failure: codex, claude-code, or copilot")
	return cmd
}

func runCIWatch(cmd *cobra.Command, root string, opts ciOptions) error {
	if opts.runGitPush {
		if err := runGitPush(cmd, root, opts.pushArgs); err != nil {
			return err
		}
	}
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	repo, source, err := githubpkg.ResolveRepository(root, opts.repo, cfg.GitHub.Repo)
	if err != nil {
		return err
	}
	token := githubpkg.TokenFromEnv()
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN or GH_TOKEN is required to watch GitHub Actions and download logs")
	}
	currentBranch, branchErr := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
	branch, err := firstNonEmptyOr(opts.branch, currentBranch, branchErr)
	if err != nil {
		return err
	}
	currentSHA, shaErr := gitOutput(root, "rev-parse", "HEAD")
	sha, err := firstNonEmptyOr(opts.sha, currentSHA, shaErr)
	if err != nil {
		return err
	}
	timeout, err := time.ParseDuration(opts.timeout)
	if err != nil {
		return fmt.Errorf("parse --timeout: %w", err)
	}
	interval, err := time.ParseDuration(opts.interval)
	if err != nil {
		return fmt.Errorf("parse --interval: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Watching %s (%s) branch %s at %s\n", repo.String(), source, branch, shortSHA(sha))
	run, err := (ciwatch.Watcher{Runs: githubpkg.NewClient(token).Actions}).Wait(cmd.Context(), ciwatch.Options{
		Owner:        repo.Owner,
		Repo:         repo.Name,
		Branch:       branch,
		HeadSHA:      sha,
		Timeout:      timeout,
		PollInterval: interval,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "GitHub Actions run %d completed with conclusion %q\n", run.ID, run.Conclusion)
	if run.HTMLURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", run.HTMLURL)
	}
	if strings.EqualFold(run.Conclusion, "success") || strings.EqualFold(run.Conclusion, "skipped") {
		fmt.Fprintln(cmd.OutOrStdout(), "CI passed; no repair bundle was needed.")
		return nil
	}

	runID := strconv.FormatInt(run.ID, 10)
	if err := runCollect(cmd, root, runID, repo.String()); err != nil {
		return err
	}
	return runPrepare(cmd, root, runID, opts.delta, opts.exports)
}

func runGitPush(cmd *cobra.Command, root string, args []string) error {
	gitArgs := append([]string{"push"}, args...)
	gitCmd := exec.CommandContext(cmd.Context(), "git", gitArgs...)
	gitCmd.Dir = root
	gitCmd.Stdout = cmd.OutOrStdout()
	gitCmd.Stderr = cmd.ErrOrStderr()
	return gitCmd.Run()
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func firstNonEmptyOr(value string, fallback string, fallbackErr error) (string, error) {
	value = strings.TrimSpace(value)
	if value != "" {
		return value, nil
	}
	if fallbackErr != nil {
		return "", fallbackErr
	}
	if strings.TrimSpace(fallback) == "" {
		return "", fmt.Errorf("git value is empty")
	}
	return strings.TrimSpace(fallback), nil
}

func shortSHA(sha string) string {
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}

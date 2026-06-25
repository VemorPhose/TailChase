package ciwatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v72/github"
)

type RunsClient interface {
	ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *gh.ListWorkflowRunsOptions) (*gh.WorkflowRuns, *gh.Response, error)
}

type Watcher struct {
	Runs  RunsClient
	Sleep func(context.Context, time.Duration) error
	Now   func() time.Time
}

type Options struct {
	Owner        string
	Repo         string
	Branch       string
	HeadSHA      string
	Timeout      time.Duration
	PollInterval time.Duration
}

type Run struct {
	ID         int64
	Status     string
	Conclusion string
	HTMLURL    string
	HeadSHA    string
	Branch     string
}

func (w Watcher) Wait(ctx context.Context, opts Options) (Run, error) {
	if w.Runs == nil {
		return Run{}, fmt.Errorf("github actions runs client is required")
	}
	opts.Owner = strings.TrimSpace(opts.Owner)
	opts.Repo = strings.TrimSpace(opts.Repo)
	opts.Branch = strings.TrimSpace(opts.Branch)
	opts.HeadSHA = strings.TrimSpace(opts.HeadSHA)
	if opts.Owner == "" || opts.Repo == "" {
		return Run{}, fmt.Errorf("github repository is required")
	}
	if opts.Branch == "" && opts.HeadSHA == "" {
		return Run{}, fmt.Errorf("branch or head SHA is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Minute
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 15 * time.Second
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	sleep := sleepContext
	if w.Sleep != nil {
		sleep = w.Sleep
	}

	deadline := now().Add(opts.Timeout)
	for {
		run, found, err := w.latestRun(ctx, opts)
		if err != nil {
			return Run{}, err
		}
		if found && strings.EqualFold(run.Status, "completed") {
			return run, nil
		}
		if !now().Before(deadline) {
			if found {
				return Run{}, fmt.Errorf("timed out waiting for GitHub Actions run %d to complete", run.ID)
			}
			return Run{}, fmt.Errorf("timed out waiting for a GitHub Actions run on %s", firstNonEmpty(opts.HeadSHA, opts.Branch))
		}
		if err := sleep(ctx, opts.PollInterval); err != nil {
			return Run{}, err
		}
	}
}

func (w Watcher) latestRun(ctx context.Context, opts Options) (Run, bool, error) {
	listOpts := &gh.ListWorkflowRunsOptions{
		Branch:      opts.Branch,
		HeadSHA:     opts.HeadSHA,
		ListOptions: gh.ListOptions{PerPage: 20},
	}
	runs, _, err := w.Runs.ListRepositoryWorkflowRuns(ctx, opts.Owner, opts.Repo, listOpts)
	if err != nil {
		return Run{}, false, fmt.Errorf("list GitHub Actions runs: %w", err)
	}
	if runs == nil {
		return Run{}, false, nil
	}
	for _, workflowRun := range runs.WorkflowRuns {
		if workflowRun == nil {
			continue
		}
		if opts.HeadSHA != "" && !strings.EqualFold(workflowRun.GetHeadSHA(), opts.HeadSHA) {
			continue
		}
		return Run{
			ID:         workflowRun.GetID(),
			Status:     workflowRun.GetStatus(),
			Conclusion: workflowRun.GetConclusion(),
			HTMLURL:    workflowRun.GetHTMLURL(),
			HeadSHA:    workflowRun.GetHeadSHA(),
			Branch:     workflowRun.GetHeadBranch(),
		}, true, nil
	}
	return Run{}, false, nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

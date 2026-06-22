package collect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	gh "github.com/google/go-github/v72/github"
)

type actionsClient interface {
	ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64, opts *gh.ListWorkflowJobsOptions) (*gh.Jobs, *gh.Response, error)
	GetWorkflowJobLogs(ctx context.Context, owner, repo string, jobID int64, maxRedirects int) (*url.URL, *gh.Response, error)
}

type GitHubActionsCollector struct {
	Actions    actionsClient
	HTTPClient *http.Client
	Now        func() time.Time
}

func NewGitHubActionsCollector(client *gh.Client) GitHubActionsCollector {
	return GitHubActionsCollector{
		Actions:    client.Actions,
		HTTPClient: http.DefaultClient,
		Now:        time.Now,
	}
}

func (c GitHubActionsCollector) Collect(ctx context.Context, run project.Run, opts GitHubActionsOptions) (Result, error) {
	if c.Actions == nil {
		return Result{}, fmt.Errorf("github actions client is required")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if opts.MaxLogLinesPerJob <= 0 {
		return Result{}, fmt.Errorf("max log lines per job must be greater than zero")
	}

	jobs, err := c.listJobs(ctx, opts)
	if err != nil {
		return Result{}, err
	}

	evidencePath := run.EvidencePath(project.GitHubActionsLogName)
	if err := os.MkdirAll(run.EvidenceDir(), 0o755); err != nil {
		return Result{}, err
	}
	file, err := os.Create(evidencePath)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	collectedAt := c.Now().UTC()
	result := Result{
		Repository:   opts.Owner + "/" + opts.Repo,
		RunID:        opts.RunID,
		EvidencePath: run.RelativePath(evidencePath),
		CollectedAt:  collectedAt,
	}

	fmt.Fprintf(file, "# Tailchase GitHub Actions evidence\n")
	fmt.Fprintf(file, "repository: %s\n", result.Repository)
	fmt.Fprintf(file, "run_id: %d\n", opts.RunID)
	fmt.Fprintf(file, "collected_at: %s\n", collectedAt.Format(time.RFC3339))
	fmt.Fprintf(file, "failed_jobs_only: %t\n\n", opts.FailedJobsOnly)

	for _, job := range jobs {
		if opts.FailedJobsOnly && !isEvidenceJob(job) {
			continue
		}
		jobResult, err := c.writeJobLog(ctx, file, opts, job)
		if err != nil {
			return Result{}, err
		}
		result.Jobs = append(result.Jobs, jobResult)
	}

	if len(result.Jobs) == 0 {
		result.Warnings = append(result.Warnings, "no failed GitHub Actions jobs were found for this run")
		fmt.Fprintln(file, "warning: no failed GitHub Actions jobs were found for this run")
	}
	if err := run.RecordArtifact(project.ArtifactGitHubActionsLog, "github_actions", evidencePath, collectedAt); err != nil {
		return Result{}, err
	}

	return result, nil
}

func (c GitHubActionsCollector) listJobs(ctx context.Context, opts GitHubActionsOptions) ([]*gh.WorkflowJob, error) {
	listOpts := &gh.ListWorkflowJobsOptions{
		Filter:      "latest",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	var out []*gh.WorkflowJob
	for {
		jobs, resp, err := c.Actions.ListWorkflowJobs(ctx, opts.Owner, opts.Repo, opts.RunID, listOpts)
		if err != nil {
			return nil, fmt.Errorf("list workflow jobs for run %d: %w", opts.RunID, err)
		}
		if jobs != nil {
			out = append(out, jobs.Jobs...)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	return out, nil
}

func (c GitHubActionsCollector) writeJobLog(ctx context.Context, out io.Writer, opts GitHubActionsOptions, job *gh.WorkflowJob) (JobResult, error) {
	id := job.GetID()
	if id == 0 {
		return JobResult{}, fmt.Errorf("workflow job is missing an ID")
	}

	logURL, _, err := c.Actions.GetWorkflowJobLogs(ctx, opts.Owner, opts.Repo, id, 1)
	if err != nil {
		return JobResult{}, fmt.Errorf("get logs for job %d (%s): %w", id, job.GetName(), err)
	}
	if logURL == nil {
		return JobResult{}, fmt.Errorf("get logs for job %d (%s): GitHub returned no log URL", id, job.GetName())
	}

	logText, lines, truncated, err := c.downloadCappedLog(ctx, logURL, opts.MaxLogLinesPerJob)
	if err != nil {
		return JobResult{}, fmt.Errorf("download logs for job %d (%s): %w", id, job.GetName(), err)
	}

	fmt.Fprintf(out, "--- tailchase-job id=%d name=%q status=%q conclusion=%q html_url=%q ---\n", id, job.GetName(), job.GetStatus(), job.GetConclusion(), job.GetHTMLURL())
	fmt.Fprint(out, logText)
	if !strings.HasSuffix(logText, "\n") {
		fmt.Fprintln(out)
	}
	if truncated {
		fmt.Fprintf(out, "[tailchase] log truncated after %d lines\n", opts.MaxLogLinesPerJob)
	}
	fmt.Fprintf(out, "--- tailchase-end-job id=%d ---\n\n", id)

	return JobResult{
		ID:           id,
		Name:         job.GetName(),
		Status:       job.GetStatus(),
		Conclusion:   job.GetConclusion(),
		HTMLURL:      job.GetHTMLURL(),
		LinesWritten: lines,
		Truncated:    truncated,
	}, nil
}

func (c GitHubActionsCollector) downloadCappedLog(ctx context.Context, logURL *url.URL, maxLines int) (string, int, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logURL.String(), nil)
	if err != nil {
		return "", 0, false, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", 0, false, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := 0
	truncated := false
	for scanner.Scan() {
		if lines >= maxLines {
			truncated = true
			break
		}
		builder.WriteString(scanner.Text())
		builder.WriteByte('\n')
		lines++
	}
	if err := scanner.Err(); err != nil {
		return "", lines, truncated, err
	}
	return builder.String(), lines, truncated, nil
}

func isEvidenceJob(job *gh.WorkflowJob) bool {
	switch strings.ToLower(job.GetConclusion()) {
	case "failure", "timed_out", "cancelled", "action_required", "startup_failure":
		return true
	}
	status := strings.ToLower(job.GetStatus())
	conclusion := strings.ToLower(job.GetConclusion())
	return status == "completed" && conclusion != "" && conclusion != "success" && conclusion != "skipped"
}

func ParseRunID(runID string) (int64, error) {
	trimmed := strings.TrimSpace(runID)
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("run ID %q must be a positive GitHub Actions run ID", runID)
	}
	return id, nil
}

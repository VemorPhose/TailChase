package collect

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/gitlab"
	"github.com/VemorPhose/TailChase/internal/project"
)

type gitLabCIClient interface {
	ListPipelineJobs(ctx context.Context, project string, pipelineID int64, failedOnly bool) ([]gitlab.Job, error)
	GetJobTrace(ctx context.Context, project string, jobID int64) (string, error)
}

type GitLabCICollector struct {
	Client gitLabCIClient
	Now    func() time.Time
}

func NewGitLabCICollector(client gitLabCIClient) GitLabCICollector {
	return GitLabCICollector{
		Client: client,
		Now:    time.Now,
	}
}

func (c GitLabCICollector) ProviderMetadata() ProviderMetadata {
	return ProviderMetadata{Name: "gitlab_ci", Kind: "ci"}
}

func (c GitLabCICollector) Collect(ctx context.Context, run project.Run, opts GitLabCIOptions) (Result, error) {
	if c.Client == nil {
		return Result{}, fmt.Errorf("gitlab client is required")
	}
	if strings.TrimSpace(opts.Project) == "" {
		return Result{}, fmt.Errorf("gitlab project is required")
	}
	if opts.PipelineID <= 0 {
		return Result{}, fmt.Errorf("gitlab pipeline ID must be greater than zero")
	}
	if opts.MaxLogLinesPerJob <= 0 {
		return Result{}, fmt.Errorf("max log lines per job must be greater than zero")
	}
	if c.Now == nil {
		c.Now = time.Now
	}

	jobs, err := c.Client.ListPipelineJobs(ctx, opts.Project, opts.PipelineID, opts.FailedJobsOnly)
	if err != nil {
		return Result{}, fmt.Errorf("list GitLab pipeline jobs for %d: %w", opts.PipelineID, err)
	}

	evidencePath := run.EvidencePath(project.GitLabCILogName)
	if err := os.MkdirAll(run.EvidenceDir(), 0o755); err != nil {
		return Result{}, err
	}
	file, err := os.Create(evidencePath)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	collectedAt := c.Now().UTC()
	provider := c.ProviderMetadata()
	evidenceRelativePath := run.RelativePath(evidencePath)
	result := Result{
		Repository:   opts.Project,
		RunID:        opts.PipelineID,
		Provider:     provider,
		EvidencePath: evidenceRelativePath,
		Sources:      []bundle.EvidenceSource{EvidenceSource("gitlab_ci", provider, evidenceRelativePath)},
		CollectedAt:  collectedAt,
	}

	fmt.Fprintln(file, "# Tailchase GitLab CI evidence")
	fmt.Fprintf(file, "repository: %s\n", opts.Project)
	fmt.Fprintf(file, "run_id: %d\n", opts.PipelineID)
	fmt.Fprintf(file, "collected_at: %s\n", collectedAt.Format(time.RFC3339))
	fmt.Fprintf(file, "failed_jobs_only: %t\n\n", opts.FailedJobsOnly)

	for _, job := range jobs {
		if opts.FailedJobsOnly && !isGitLabEvidenceJob(job) {
			continue
		}
		jobResult, err := c.writeJobTrace(ctx, file, opts, job)
		if err != nil {
			return Result{}, err
		}
		result.Jobs = append(result.Jobs, jobResult)
	}
	if len(result.Jobs) == 0 {
		result.Warnings = append(result.Warnings, "no failed GitLab CI jobs were found for this pipeline")
		fmt.Fprintln(file, "warning: no failed GitLab CI jobs were found for this pipeline")
	}
	if err := run.RecordArtifact(project.ArtifactGitLabCILog, "gitlab_ci", evidencePath, collectedAt); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (c GitLabCICollector) writeJobTrace(ctx context.Context, out *os.File, opts GitLabCIOptions, job gitlab.Job) (JobResult, error) {
	if job.ID <= 0 {
		return JobResult{}, fmt.Errorf("gitlab job is missing an ID")
	}
	trace, err := c.Client.GetJobTrace(ctx, opts.Project, job.ID)
	if err != nil {
		return JobResult{}, fmt.Errorf("get GitLab trace for job %d (%s): %w", job.ID, job.Name, err)
	}
	logText, lines, truncated, err := capLogLines(trace, opts.MaxLogLinesPerJob)
	if err != nil {
		return JobResult{}, err
	}

	fmt.Fprintf(out, "--- tailchase-job id=%d name=%q status=%q conclusion=%q html_url=%q ---\n", job.ID, job.Name, job.Status, job.Status, job.WebURL)
	fmt.Fprint(out, logText)
	if !strings.HasSuffix(logText, "\n") {
		fmt.Fprintln(out)
	}
	if truncated {
		fmt.Fprintf(out, "[tailchase] log truncated after %d lines\n", opts.MaxLogLinesPerJob)
	}
	fmt.Fprintf(out, "--- tailchase-end-job id=%d ---\n\n", job.ID)

	return JobResult{
		ID:           job.ID,
		Name:         job.Name,
		Status:       job.Status,
		Conclusion:   job.Status,
		HTMLURL:      job.WebURL,
		LinesWritten: lines,
		Truncated:    truncated,
	}, nil
}

func isGitLabEvidenceJob(job gitlab.Job) bool {
	switch strings.ToLower(job.Status) {
	case "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func capLogLines(text string, maxLines int) (string, int, bool, error) {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(text))
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

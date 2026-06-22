package collect

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/gitlab"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestGitLabCICollectorWritesFailedJobTraces(t *testing.T) {
	client := fakeGitLabClient{
		jobs: []gitlab.Job{
			{ID: 21, Name: "unit tests", Status: "failed", WebURL: "https://gitlab.example/job/21"},
			{ID: 22, Name: "lint", Status: "success", WebURL: "https://gitlab.example/job/22"},
		},
		traces: map[int64]string{
			21: "line 1\nline 2\nline 3\n",
		},
	}
	run := mustCollectRun(t)
	collector := GitLabCICollector{
		Client: client,
		Now:    func() time.Time { return time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC) },
	}

	result, err := collector.Collect(context.Background(), run, GitLabCIOptions{
		Project:           "group/project",
		PipelineID:        12345,
		FailedJobsOnly:    true,
		MaxLogLinesPerJob: 2,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(result.Jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(result.Jobs))
	}
	if !result.Jobs[0].Truncated {
		t.Fatal("job should be marked truncated")
	}
	if result.Provider.Name != "gitlab_ci" || result.Provider.Kind != "ci" {
		t.Fatalf("provider metadata = %#v, want GitLab CI", result.Provider)
	}
	if len(result.Sources) != 1 || result.Sources[0].Provider != "gitlab_ci" || result.Sources[0].ProviderKind != "ci" {
		t.Fatalf("sources = %#v, want GitLab source metadata", result.Sources)
	}

	data, err := os.ReadFile(run.EvidencePath(project.GitLabCILogName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `name="unit tests"`) || !strings.Contains(text, "repository: group/project") {
		t.Fatalf("evidence missing GitLab job metadata:\n%s", text)
	}
	if strings.Contains(text, `name="lint"`) {
		t.Fatalf("evidence included successful job:\n%s", text)
	}
	if strings.Contains(text, "line 3") {
		t.Fatalf("evidence was not capped:\n%s", text)
	}
	meta, err := run.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() error = %v", err)
	}
	if len(meta.Artifacts) != 1 || meta.Artifacts[0].Name != project.ArtifactGitLabCILog {
		t.Fatalf("metadata artifacts = %#v, want gitlab ci log", meta.Artifacts)
	}
}

func TestGitLabCICollectorValidatesConfig(t *testing.T) {
	run := mustCollectRun(t)
	_, err := (GitLabCICollector{}).Collect(context.Background(), run, GitLabCIOptions{
		Project:           "group/project",
		PipelineID:        12345,
		MaxLogLinesPerJob: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "gitlab client is required") {
		t.Fatalf("error = %v, want missing client", err)
	}

	_, err = (GitLabCICollector{Client: fakeGitLabClient{}}).Collect(context.Background(), run, GitLabCIOptions{
		PipelineID:        12345,
		MaxLogLinesPerJob: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "gitlab project is required") {
		t.Fatalf("error = %v, want missing project", err)
	}
}

func TestGitLabCICollectorImplementsProviderInterface(t *testing.T) {
	var collector ProviderCollector[GitLabCIOptions] = GitLabCICollector{}

	metadata := collector.ProviderMetadata()
	if metadata.Name != "gitlab_ci" || metadata.Kind != "ci" {
		t.Fatalf("metadata = %#v, want GitLab CI", metadata)
	}
}

type fakeGitLabClient struct {
	jobs   []gitlab.Job
	traces map[int64]string
}

func (f fakeGitLabClient) ListPipelineJobs(context.Context, string, int64, bool) ([]gitlab.Job, error) {
	return f.jobs, nil
}

func (f fakeGitLabClient) GetJobTrace(_ context.Context, _ string, jobID int64) (string, error) {
	return f.traces[jobID], nil
}

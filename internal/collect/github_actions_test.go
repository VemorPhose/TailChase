package collect

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	gh "github.com/google/go-github/v72/github"
)

func TestGitHubActionsCollectorWritesFailedJobLogs(t *testing.T) {
	logURL, err := url.Parse("https://logs.example/job/11")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	actions := fakeActionsClient{
		jobs: []*gh.WorkflowJob{
			{
				ID:         gh.Ptr(int64(11)),
				Name:       gh.Ptr("unit tests"),
				Status:     gh.Ptr("completed"),
				Conclusion: gh.Ptr("failure"),
			},
			{
				ID:         gh.Ptr(int64(12)),
				Name:       gh.Ptr("lint"),
				Status:     gh.Ptr("completed"),
				Conclusion: gh.Ptr("success"),
			},
		},
		logURL: logURL,
	}
	run := mustRun(t)
	collector := GitHubActionsCollector{
		Actions:    actions,
		HTTPClient: fakeHTTPClient("line 1\nline 2\nline 3\n"),
		Now:        func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) },
	}

	result, err := collector.Collect(context.Background(), run, GitHubActionsOptions{
		Owner:             "owner",
		Repo:              "repo",
		RunID:             12345,
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
		t.Fatalf("job should be marked truncated")
	}

	data, err := os.ReadFile(run.EvidencePath(project.GitHubActionsLogName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `name="unit tests"`) {
		t.Fatalf("evidence did not include failed job header:\n%s", text)
	}
	if strings.Contains(text, `name="lint"`) {
		t.Fatalf("evidence included successful job:\n%s", text)
	}
	if strings.Contains(text, "line 3") {
		t.Fatalf("evidence was not capped:\n%s", text)
	}
	if !strings.Contains(text, "log truncated after 2 lines") {
		t.Fatalf("evidence did not mention truncation:\n%s", text)
	}
}

func TestParseRunID(t *testing.T) {
	if _, err := ParseRunID("12345"); err != nil {
		t.Fatalf("ParseRunID() error = %v", err)
	}
	for _, input := range []string{"", "abc", "-1"} {
		if _, err := ParseRunID(input); err == nil {
			t.Fatalf("ParseRunID(%q) error = nil, want error", input)
		}
	}
}

func mustRun(t *testing.T) project.Run {
	t.Helper()
	run, err := project.NewStore(t.TempDir()).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	return run
}

type fakeActionsClient struct {
	jobs   []*gh.WorkflowJob
	logURL *url.URL
}

func (f fakeActionsClient) ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64, opts *gh.ListWorkflowJobsOptions) (*gh.Jobs, *gh.Response, error) {
	return &gh.Jobs{Jobs: f.jobs}, &gh.Response{}, nil
}

func (f fakeActionsClient) GetWorkflowJobLogs(ctx context.Context, owner, repo string, jobID int64, maxRedirects int) (*url.URL, *gh.Response, error) {
	return f.logURL, &gh.Response{}, nil
}

func fakeHTTPClient(body string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

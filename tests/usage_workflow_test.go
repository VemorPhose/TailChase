package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/ciwatch"
	"github.com/VemorPhose/TailChase/internal/project"
	gh "github.com/google/go-github/v72/github"
)

func TestInitDefaultGoalIsProjectAgnosticAndActionable(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	goal, err := project.LoadGoal(root)
	if err != nil {
		t.Fatalf("LoadGoal() error = %v", err)
	}
	if strings.Contains(strings.ToLower(goal.Goal), "todo") || len(goal.DoneConditions) < 3 {
		t.Fatalf("default goal = %#v, want actionable project-agnostic defaults", goal)
	}
	if len(goal.ExpectedPaths) != 1 || goal.ExpectedPaths[0] != "." {
		t.Fatalf("expected paths = %#v, want whole-repo default", goal.ExpectedPaths)
	}
	if !hasSubstring(goal.StopRules, "weakening") || !hasSubstring(goal.SuspiciousPaths, ".github/workflows") {
		t.Fatalf("goal = %#v, want safety stop rules and suspicious paths", goal)
	}
}

func TestPrepareCommandBuildsStandardArtifacts(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	writeConfig(t, root, "file")
	writeGoal(t, root)
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFile(t, run.EvidencePath(project.GitHubActionsLogName), `# Tailchase GitHub Actions evidence
repository: owner/repo
run_id: 12345
--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="" ---
internal/app/app.go:42:10: undefined: Handler
--- tailchase-end-job id=11 ---
`)

	stdout, _, err := runTailchase(t, "prepare", "--run", "12345", "--export", "codex")
	if err != nil {
		t.Fatalf("tailchase prepare error = %v", err)
	}
	if !strings.Contains(stdout, "Prepared run 12345") {
		t.Fatalf("stdout = %q, want prepared message", stdout)
	}
	for _, name := range []string{
		project.NormalizedEvidenceName,
		project.FailureBundleName,
		project.RepairPromptName,
		project.ReportName,
		filepath.Join(project.ExportsDirName, "codex-prompt.md"),
	} {
		if _, err := os.Stat(run.ArtifactPath(name)); err != nil {
			t.Fatalf("%s missing after prepare: %v", name, err)
		}
	}
}

func TestPrepareCommandReportsMissingRun(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}
	_, _, err := runTailchase(t, "prepare", "--run", "99999")
	if err == nil || !strings.Contains(err.Error(), "run 99999 does not exist") {
		t.Fatalf("error = %v, want missing run message", err)
	}
}

func TestCIWatchRequiresGitHubTokenBeforeNetworkUse(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	runGit(t, root, "init")
	runGit(t, root, "remote", "add", "origin", "https://github.com/owner/repo.git")
	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}

	_, _, err := runTailchase(t, "ci", "watch", "--branch", "main", "--sha", "abc123")
	if err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN or GH_TOKEN is required") {
		t.Fatalf("error = %v, want missing token error", err)
	}
}

func TestCIPushValidatesTokenBeforeGitPush(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	if _, _, err := runTailchase(t, "init"); err != nil {
		t.Fatalf("tailchase init error = %v", err)
	}

	_, _, err := runTailchase(t, "ci", "push", "--repo", "owner/repo", "--branch", "main", "--sha", "abc123", "origin", "main")
	if err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN or GH_TOKEN is required") {
		t.Fatalf("error = %v, want missing token error before git push", err)
	}
}

func TestCIWatcherWaitsForMatchingCompletedRun(t *testing.T) {
	client := &fakeRunsClient{
		pages: [][]*gh.WorkflowRun{
			{workflowRun(101, "in_progress", "", "abc123", "main")},
			{workflowRun(101, "completed", "failure", "abc123", "main")},
		},
	}
	sleeps := 0
	run, err := (ciwatch.Watcher{
		Runs: client,
		Sleep: func(ctx context.Context, d time.Duration) error {
			sleeps++
			return nil
		},
	}).Wait(context.Background(), ciwatch.Options{
		Owner:        "owner",
		Repo:         "repo",
		Branch:       "main",
		HeadSHA:      "abc123",
		Timeout:      time.Minute,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if run.ID != 101 || run.Conclusion != "failure" || sleeps != 1 {
		t.Fatalf("run = %#v sleeps = %d, want completed matching run after one sleep", run, sleeps)
	}
}

func TestCIWatcherTimesOutWhenNoRunAppears(t *testing.T) {
	current := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	_, err := (ciwatch.Watcher{
		Runs: &fakeRunsClient{},
		Now:  func() time.Time { return current },
		Sleep: func(ctx context.Context, d time.Duration) error {
			current = current.Add(d)
			return nil
		},
	}).Wait(context.Background(), ciwatch.Options{
		Owner:        "owner",
		Repo:         "repo",
		Branch:       "main",
		Timeout:      time.Second,
		PollInterval: time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

type fakeRunsClient struct {
	calls int
	pages [][]*gh.WorkflowRun
}

func (f *fakeRunsClient) ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *gh.ListWorkflowRunsOptions) (*gh.WorkflowRuns, *gh.Response, error) {
	if owner != "owner" || repo != "repo" {
		return nil, nil, fmt.Errorf("repo = %s/%s", owner, repo)
	}
	var runs []*gh.WorkflowRun
	if f.calls < len(f.pages) {
		runs = f.pages[f.calls]
	}
	f.calls++
	return &gh.WorkflowRuns{WorkflowRuns: runs}, nil, nil
}

func workflowRun(id int64, status string, conclusion string, sha string, branch string) *gh.WorkflowRun {
	return &gh.WorkflowRun{
		ID:         gh.Ptr(id),
		Status:     gh.Ptr(status),
		Conclusion: gh.Ptr(conclusion),
		HeadSHA:    gh.Ptr(sha),
		HeadBranch: gh.Ptr(branch),
		HTMLURL:    gh.Ptr("https://github.com/owner/repo/actions/runs/101"),
	}
}

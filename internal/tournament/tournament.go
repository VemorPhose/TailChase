package tournament

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

const defaultTestCommand = "go test ./..."

type Options struct {
	Root        string
	BranchA     string
	BranchB     string
	TestCommand string
	Now         func() time.Time
}

type Result struct {
	GeneratedAt time.Time
	BranchA     Candidate
	BranchB     Candidate
	Winner      string
	Rationale   string
	ReportPath  string
}

type Candidate struct {
	Branch             string
	Commit             string
	Score              int
	TestOutcome        string
	TestCommand        string
	TestOutput         string
	ChangedPaths       []string
	DependencyChanges  []string
	SafetyFindings     int
	StopFindings       int
	BundleQualityScore int
	BundleNotes        []string
}

func Evaluate(ctx context.Context, opts Options) (Result, error) {
	opts.Root = firstNonEmpty(opts.Root, ".")
	opts.BranchA = strings.TrimSpace(opts.BranchA)
	opts.BranchB = strings.TrimSpace(opts.BranchB)
	if opts.BranchA == "" || opts.BranchB == "" {
		return Result{}, fmt.Errorf("two branch names are required")
	}
	if opts.BranchA == opts.BranchB {
		return Result{}, fmt.Errorf("branches must be different")
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	git := gitRunner{root: opts.Root}

	left, err := evaluateCandidate(ctx, git, opts.BranchA, opts.TestCommand)
	if err != nil {
		return Result{}, err
	}
	right, err := evaluateCandidate(ctx, git, opts.BranchB, opts.TestCommand)
	if err != nil {
		return Result{}, err
	}
	winner, rationale := chooseWinner(left, right)
	reportPath := ReportPath(opts.Root, opts.BranchA, opts.BranchB)

	return Result{
		GeneratedAt: now().UTC(),
		BranchA:     left,
		BranchB:     right,
		Winner:      winner,
		Rationale:   rationale,
		ReportPath:  reportPath,
	}, nil
}

func WriteReport(ctx context.Context, opts Options) (Result, error) {
	result, err := Evaluate(ctx, opts)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(result.ReportPath), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(result.ReportPath, []byte(RenderMarkdown(result)), 0o644); err != nil {
		return Result{}, err
	}
	return result, nil
}

func RenderMarkdown(result Result) string {
	var out bytes.Buffer
	fmt.Fprintln(&out, "# Tailchase Tournament Report")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "- Generated at: %s\n", result.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&out, "- Winner: %s\n", result.Winner)
	fmt.Fprintf(&out, "- Rationale: %s\n", result.Rationale)
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "## Evaluation Criteria")
	fmt.Fprintln(&out, "- Test outcome from a temporary detached worktree")
	fmt.Fprintln(&out, "- Changed path count and dependency file changes")
	fmt.Fprintln(&out, "- Safety findings and stop findings from Tailchase bundles")
	fmt.Fprintln(&out, "- Bundle quality based on root candidates, artifacts, and budget metadata")
	fmt.Fprintln(&out)
	writeCandidate(&out, result.BranchA)
	fmt.Fprintln(&out)
	writeCandidate(&out, result.BranchB)
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func ReportPath(root string, branchA string, branchB string) string {
	return filepath.Join(root, project.DirName, "tournaments", safeName(branchA)+"-vs-"+safeName(branchB)+".md")
}

func evaluateCandidate(ctx context.Context, git gitRunner, branch string, testCommand string) (Candidate, error) {
	commit, err := git.output(ctx, "rev-parse", "--verify", branch+"^{commit}")
	if err != nil {
		return Candidate{}, fmt.Errorf("resolve branch %q: %w", branch, err)
	}
	base, err := git.output(ctx, "merge-base", "HEAD", branch)
	if err != nil {
		return Candidate{}, fmt.Errorf("find merge base for %q: %w", branch, err)
	}
	changedPaths, err := git.lines(ctx, "diff", "--name-only", strings.TrimSpace(base), branch)
	if err != nil {
		return Candidate{}, fmt.Errorf("list changed paths for %q: %w", branch, err)
	}
	sort.Strings(changedPaths)

	candidate := Candidate{
		Branch:            branch,
		Commit:            strings.TrimSpace(commit),
		ChangedPaths:      changedPaths,
		DependencyChanges: dependencyChanges(changedPaths),
	}
	candidate.TestCommand = firstNonEmpty(strings.TrimSpace(testCommand), defaultTestCommand)
	if !branchHasPath(ctx, git, branch, "go.mod") && strings.TrimSpace(testCommand) == "" {
		candidate.TestOutcome = "not_run"
		candidate.TestOutput = "go.mod not present and no --test-command was provided"
	} else {
		candidate.TestOutcome, candidate.TestOutput = runCandidateTests(ctx, git, branch, candidate.TestCommand)
	}

	failureBundle, notes, err := latestFailureBundle(ctx, git, branch)
	if err != nil {
		return Candidate{}, err
	}
	candidate.BundleNotes = notes
	if failureBundle != nil {
		candidate.SafetyFindings = len(failureBundle.SafetyFindings)
		for _, finding := range failureBundle.SafetyFindings {
			if finding.Decision == bundle.SafetyDecisionStop {
				candidate.StopFindings++
			}
		}
		candidate.BundleQualityScore, candidate.BundleNotes = bundleQuality(*failureBundle, candidate.BundleNotes)
	}
	candidate.Score = scoreCandidate(candidate)
	return candidate, nil
}

func runCandidateTests(ctx context.Context, git gitRunner, branch string, testCommand string) (string, string) {
	worktree, err := os.MkdirTemp("", "tailchase-tournament-*")
	if err != nil {
		return "error", err.Error()
	}
	defer os.RemoveAll(worktree)
	if _, err := git.output(ctx, "worktree", "add", "--detach", worktree, branch); err != nil {
		return "error", err.Error()
	}
	defer git.output(context.Background(), "worktree", "remove", "--force", worktree)

	cmd := exec.CommandContext(ctx, "sh", "-c", testCommand)
	cmd.Dir = worktree
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "failed", strings.TrimSpace(string(output))
	}
	return "passed", strings.TrimSpace(string(output))
}

func latestFailureBundle(ctx context.Context, git gitRunner, branch string) (*bundle.FailureBundle, []string, error) {
	paths, err := git.lines(ctx, "ls-tree", "-r", "--name-only", branch, "--", project.DirName)
	if err != nil {
		return nil, []string{"no Tailchase artifacts found"}, nil
	}
	var bundles []string
	for _, path := range paths {
		if strings.HasSuffix(path, "/"+project.FailureBundleName) {
			bundles = append(bundles, path)
		}
	}
	if len(bundles) == 0 {
		return nil, []string{"no failure bundle found"}, nil
	}
	sort.Strings(bundles)
	path := bundles[len(bundles)-1]
	data, err := git.output(ctx, "show", branch+":"+path)
	if err != nil {
		return nil, nil, fmt.Errorf("read failure bundle %q from %q: %w", path, branch, err)
	}
	var failureBundle bundle.FailureBundle
	if err := yaml.Unmarshal([]byte(data), &failureBundle); err != nil {
		return nil, nil, fmt.Errorf("parse failure bundle %q from %q: %w", path, branch, err)
	}
	if failureBundle.Version == 0 {
		failureBundle.Version = bundle.SchemaVersion
	}
	if failureBundle.Version != bundle.SchemaVersion {
		return nil, nil, fmt.Errorf("unsupported failure bundle version %d in %q", failureBundle.Version, branch)
	}
	return &failureBundle, []string{"bundle: " + path}, nil
}

func bundleQuality(failureBundle bundle.FailureBundle, notes []string) (int, []string) {
	score := 0
	if len(failureBundle.RootErrorCandidates) > 0 {
		score += 10
		notes = append(notes, "root candidates present")
	}
	if len(failureBundle.Artifacts) > 0 {
		score += 5
		notes = append(notes, "artifact references present")
	}
	if failureBundle.Budget.RawEvidenceBytes > 0 || failureBundle.Budget.EstimatedPromptBytes > 0 {
		score += 5
		notes = append(notes, "budget metadata present")
	}
	return score, notes
}

func scoreCandidate(candidate Candidate) int {
	score := candidate.BundleQualityScore
	switch candidate.TestOutcome {
	case "passed":
		score += 50
	case "not_run":
		score += 10
	}
	if len(candidate.ChangedPaths) <= 10 {
		score += 10
	}
	score -= len(candidate.DependencyChanges) * 10
	score -= candidate.SafetyFindings * 5
	score -= candidate.StopFindings * 20
	return score
}

func chooseWinner(left Candidate, right Candidate) (string, string) {
	switch {
	case left.Score > right.Score:
		return left.Branch, fmt.Sprintf("%s scored %d vs %d with stronger test, safety, or bundle signals", left.Branch, left.Score, right.Score)
	case right.Score > left.Score:
		return right.Branch, fmt.Sprintf("%s scored %d vs %d with stronger test, safety, or bundle signals", right.Branch, right.Score, left.Score)
	default:
		return "tie", fmt.Sprintf("both branches scored %d", left.Score)
	}
}

func writeCandidate(out *bytes.Buffer, candidate Candidate) {
	fmt.Fprintf(out, "## Candidate: `%s`\n", candidate.Branch)
	fmt.Fprintf(out, "- Commit: `%s`\n", candidate.Commit)
	fmt.Fprintf(out, "- Score: %d\n", candidate.Score)
	fmt.Fprintf(out, "- Tests: %s\n", candidate.TestOutcome)
	fmt.Fprintf(out, "- Test command: `%s`\n", candidate.TestCommand)
	fmt.Fprintf(out, "- Changed paths: %d\n", len(candidate.ChangedPaths))
	for _, path := range candidate.ChangedPaths {
		fmt.Fprintf(out, "  - `%s`\n", path)
	}
	fmt.Fprintf(out, "- Dependency changes: %d\n", len(candidate.DependencyChanges))
	for _, path := range candidate.DependencyChanges {
		fmt.Fprintf(out, "  - `%s`\n", path)
	}
	fmt.Fprintf(out, "- Safety findings: %d\n", candidate.SafetyFindings)
	fmt.Fprintf(out, "- Stop findings: %d\n", candidate.StopFindings)
	fmt.Fprintf(out, "- Bundle quality score: %d\n", candidate.BundleQualityScore)
	for _, note := range candidate.BundleNotes {
		fmt.Fprintf(out, "  - %s\n", note)
	}
}

func dependencyChanges(paths []string) []string {
	dependencyNames := map[string]bool{
		"go.mod":            true,
		"go.sum":            true,
		"package.json":      true,
		"package-lock.json": true,
		"pnpm-lock.yaml":    true,
		"yarn.lock":         true,
		"bun.lockb":         true,
		"Cargo.toml":        true,
		"Cargo.lock":        true,
		"requirements.txt":  true,
		"pyproject.toml":    true,
		"poetry.lock":       true,
	}
	var changes []string
	for _, path := range paths {
		if dependencyNames[filepath.Base(path)] {
			changes = append(changes, path)
		}
	}
	return changes
}

func branchHasPath(ctx context.Context, git gitRunner, branch string, path string) bool {
	_, err := git.output(ctx, "cat-file", "-e", branch+":"+path)
	return err == nil
}

type gitRunner struct {
	root string
}

func (g gitRunner) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", g.root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "git " + strings.Join(args, " ") + " failed"
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

func (g gitRunner) lines(ctx context.Context, args ...string) ([]string, error) {
	output, err := g.output(ctx, args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func safeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	dash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			dash = false
			continue
		}
		if !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

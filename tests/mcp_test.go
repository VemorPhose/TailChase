package tests

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/mcpserver"
	"github.com/VemorPhose/TailChase/internal/project"
	promptpkg "github.com/VemorPhose/TailChase/internal/prompt"
)

func TestMCPSnapshotExposesTailchaseResources(t *testing.T) {
	root, run := writeMCPFixture(t)

	snapshot, err := mcpserver.BuildSnapshot(root, "12345")
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}
	if snapshot.RunID != "12345" {
		t.Fatalf("run ID = %q, want 12345", snapshot.RunID)
	}
	resources := snapshot.ResourceList()
	if len(resources) != 3 {
		t.Fatalf("resources = %#v, want goal, bundle, prompt", resources)
	}
	for _, resource := range resources {
		if resource.Text != "" {
			t.Fatalf("resource list leaked text: %#v", resource)
		}
	}

	prompt, err := snapshot.ReadResource("tailchase://runs/" + run.ID + "/repair-prompt")
	if err != nil {
		t.Fatalf("ReadResource(prompt) error = %v", err)
	}
	if !strings.Contains(prompt.Text, "Fix undefined Handler") {
		t.Fatalf("prompt resource = %q, want repair prompt", prompt.Text)
	}
	budget, err := snapshot.CallTool("tailchase.budget_summary")
	if err != nil {
		t.Fatalf("CallTool(budget) error = %v", err)
	}
	if !strings.Contains(budget, "raw_evidence_bytes: 9000") || !strings.Contains(budget, "estimated_prompt_bytes: 1200") {
		t.Fatalf("budget summary = %q", budget)
	}
}

func TestMCPServerReadsResource(t *testing.T) {
	root, run := writeMCPFixture(t)
	snapshot, err := mcpserver.BuildSnapshot(root, run.ID)
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"tailchase://runs/12345/failure-bundle"}}` + "\n")
	var output bytes.Buffer
	if err := mcpserver.Serve(context.Background(), snapshot, input, &output); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if !strings.Contains(output.String(), `"jsonrpc":"2.0"`) || !strings.Contains(output.String(), "undefined: Handler") {
		t.Fatalf("server output = %s", output.String())
	}
}

func TestMCPCommandListsResources(t *testing.T) {
	root, _ := writeMCPFixture(t)
	t.Chdir(root)

	stdout, _, err := runTailchase(t, "mcp", "--run", "12345", "--list-resources")
	if err != nil {
		t.Fatalf("tailchase mcp --list-resources error = %v", err)
	}
	for _, want := range []string{"Current goal", "Latest failure bundle", "Next repair instruction"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("mcp resources missing %q:\n%s", want, stdout)
		}
	}
}

func writeMCPFixture(t *testing.T) (string, project.Run) {
	t.Helper()

	root := t.TempDir()
	writeGoal(t, root)
	run, err := project.NewStore(root).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	writeFile(t, run.ArtifactPath(project.FailureBundleName), `version: 1
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
goal:
  goal: Fix CI
budget:
  raw_evidence_bytes: 9000
  included_excerpt_bytes: 500
  repeated_blocks_collapsed: 4
  estimated_prompt_bytes: 1200
safety_findings:
  - rule: test_weakening
    decision: stop
    message: test weakening detected
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    file: internal/app/app.go
    line: 42
    confidence: high
artifacts:
  - name: failure_bundle
    path: .tailchase/runs/12345/failure-bundle.yml
`)
	if err := promptpkg.WriteRepairPrompt(run, promptpkg.Result{Content: "# Repair Prompt\nFix undefined Handler.\n"}); err != nil {
		t.Fatalf("WriteRepairPrompt() error = %v", err)
	}
	return root, run
}

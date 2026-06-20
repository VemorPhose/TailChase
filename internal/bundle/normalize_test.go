package bundle

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
)

func TestNormalizerExtractsSignals(t *testing.T) {
	run := mustBundleRun(t)
	log := `# Tailchase GitHub Actions evidence
--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="" ---
2026-06-20T10:00:00Z ::error file=internal/app/app.go,line=42::undefined: Handler
internal/app/app.go:42:10: undefined: Handler
--- FAIL: TestHandler
panic: missing required environment variable API_TOKEN
--- tailchase-end-job id=11 ---
`
	if err := os.WriteFile(run.EvidencePath(project.GitHubActionsLogName), []byte(log), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	normalized, err := (Normalizer{
		Now: func() time.Time { return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC) },
	}).NormalizeRun(run)
	if err != nil {
		t.Fatalf("NormalizeRun() error = %v", err)
	}

	if len(normalized.Signals) != 4 {
		t.Fatalf("signals = %d, want 4: %#v", len(normalized.Signals), normalized.Signals)
	}
	if normalized.Signals[0].Type != "github_annotation" {
		t.Fatalf("first signal type = %q, want github_annotation", normalized.Signals[0].Type)
	}
	if normalized.Signals[0].File != "internal/app/app.go" || normalized.Signals[0].Line != 42 {
		t.Fatalf("annotation location = %s:%d, want internal/app/app.go:42", normalized.Signals[0].File, normalized.Signals[0].Line)
	}
	if normalized.Signals[0].Job != "unit tests" {
		t.Fatalf("job = %q, want unit tests", normalized.Signals[0].Job)
	}
}

func TestWriteAndReadNormalizedEvidence(t *testing.T) {
	run := mustBundleRun(t)
	normalized := NormalizedEvidence{
		Version:     schemaVersion,
		GeneratedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		Signals: []Signal{
			{Type: "generic_failure", Source: "github_actions", Message: "build failed", Confidence: "medium"},
		},
	}

	if err := WriteNormalizedEvidence(run, normalized); err != nil {
		t.Fatalf("WriteNormalizedEvidence() error = %v", err)
	}
	got, err := ReadNormalizedEvidence(run)
	if err != nil {
		t.Fatalf("ReadNormalizedEvidence() error = %v", err)
	}
	if got.Signals[0].Message != "build failed" {
		t.Fatalf("message = %q, want build failed", got.Signals[0].Message)
	}

	data, err := os.ReadFile(run.ArtifactPath(project.NormalizedEvidenceName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "generic_failure") {
		t.Fatalf("normalized YAML did not contain signal: %s", string(data))
	}
}

func mustBundleRun(t *testing.T) project.Run {
	t.Helper()
	run, err := project.NewStore(t.TempDir()).EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	return run
}

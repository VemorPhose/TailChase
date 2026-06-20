package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRunCreatesExpectedLayout(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	run, err := store.EnsureRun("12345")
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	wantEvidence := filepath.Join(root, DirName, RunsDirName, "12345", EvidenceDirName)
	if run.EvidenceDir() != wantEvidence {
		t.Fatalf("EvidenceDir() = %q, want %q", run.EvidenceDir(), wantEvidence)
	}
	if _, err := os.Stat(wantEvidence); err != nil {
		t.Fatalf("evidence dir was not created: %v", err)
	}
}

func TestOpenRunRequiresExistingRun(t *testing.T) {
	_, err := NewStore(t.TempDir()).OpenRun("12345")
	if err == nil {
		t.Fatal("OpenRun() error = nil, want missing run error")
	}
}

func TestValidateRunID(t *testing.T) {
	tests := map[string]bool{
		"12345": true,
		"":      false,
		"abc":   false,
		"../1":  false,
	}

	for runID, wantOK := range tests {
		err := ValidateRunID(runID)
		if (err == nil) != wantOK {
			t.Fatalf("ValidateRunID(%q) error = %v, want ok = %v", runID, err, wantOK)
		}
	}
}

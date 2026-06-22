package tests

import (
	"strings"
	"testing"

	"github.com/VemorPhose/TailChase/internal/adapter"
	"github.com/VemorPhose/TailChase/internal/project"
)

func TestAdapterCapabilityDiscovery(t *testing.T) {
	codex, err := adapter.Discover("codex", nil)
	if err != nil {
		t.Fatalf("Discover(codex) error = %v", err)
	}
	if err := adapter.RequireCapability(codex, adapter.CapabilityArtifact); err != nil {
		t.Fatalf("RequireCapability(artifact) error = %v", err)
	}

	copilot, err := adapter.Discover("copilot", nil)
	if err != nil {
		t.Fatalf("Discover(copilot) error = %v", err)
	}
	err = adapter.RequireCapability(copilot, adapter.CapabilityCheckpoint)
	if err == nil || !strings.Contains(err.Error(), "use \"artifact\" fallback") {
		t.Fatalf("error = %v, want safe unsupported capability", err)
	}
}

func TestAdapterConfigOverrideMustBeSupported(t *testing.T) {
	_, err := adapter.Discover("copilot", []project.AdapterConfig{{Target: "copilot", Capability: "checkpoint"}})
	if err == nil || !strings.Contains(err.Error(), "does not support capability") {
		t.Fatalf("error = %v, want unsupported override", err)
	}

	got, err := adapter.Discover("codex", []project.AdapterConfig{{Target: "codex", Capability: "hook_mcp"}})
	if err != nil {
		t.Fatalf("Discover(codex override) error = %v", err)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0] != adapter.CapabilityHookMCP || got.Fallback != adapter.CapabilityArtifact {
		t.Fatalf("adapter = %#v, want hook MCP with artifact fallback", got)
	}
}

func TestAdaptersCommandListsCapabilities(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeConfig(t, root, "file")

	stdout, _, err := runTailchase(t, "adapters", "--target", "codex")
	if err != nil {
		t.Fatalf("tailchase adapters error = %v", err)
	}
	for _, want := range []string{"Codex", "artifact", "hook_mcp", "fallback: artifact"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("adapters output missing %q:\n%s", want, stdout)
		}
	}
}

package adapter

import (
	"fmt"
	"sort"

	"github.com/VemorPhose/TailChase/internal/project"
)

type CapabilityLevel string

const (
	CapabilityArtifact   CapabilityLevel = "artifact"
	CapabilityQueued     CapabilityLevel = "queued"
	CapabilityCheckpoint CapabilityLevel = "checkpoint"
	CapabilityHookMCP    CapabilityLevel = "hook_mcp"
	CapabilityWrapper    CapabilityLevel = "wrapper"
)

type Adapter struct {
	Target       string            `json:"target"`
	DisplayName  string            `json:"display_name"`
	Capabilities []CapabilityLevel `json:"capabilities"`
	Fallback     CapabilityLevel   `json:"fallback"`
}

var catalog = map[string]Adapter{
	"codex": {
		Target:       "codex",
		DisplayName:  "Codex",
		Capabilities: []CapabilityLevel{CapabilityArtifact, CapabilityHookMCP},
		Fallback:     CapabilityArtifact,
	},
	"claude-code": {
		Target:       "claude-code",
		DisplayName:  "Claude Code",
		Capabilities: []CapabilityLevel{CapabilityArtifact, CapabilityHookMCP},
		Fallback:     CapabilityArtifact,
	},
	"copilot": {
		Target:       "copilot",
		DisplayName:  "GitHub Copilot",
		Capabilities: []CapabilityLevel{CapabilityArtifact},
		Fallback:     CapabilityArtifact,
	},
	"cursor-vscode": {
		Target:       "cursor-vscode",
		DisplayName:  "Cursor / VS Code",
		Capabilities: []CapabilityLevel{CapabilityArtifact},
		Fallback:     CapabilityArtifact,
	},
	"generic": {
		Target:       "generic",
		DisplayName:  "Generic File/Stdout",
		Capabilities: []CapabilityLevel{CapabilityArtifact},
		Fallback:     CapabilityArtifact,
	},
}

func List() []Adapter {
	targets := make([]string, 0, len(catalog))
	for target := range catalog {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	adapters := make([]Adapter, 0, len(targets))
	for _, target := range targets {
		adapters = append(adapters, catalog[target])
	}
	return adapters
}

func Discover(target string, overrides []project.AdapterConfig) (Adapter, error) {
	base, ok := catalog[target]
	if !ok {
		return Adapter{}, fmt.Errorf("unsupported adapter target %q", target)
	}
	for _, override := range overrides {
		if override.Target != target {
			continue
		}
		capability := CapabilityLevel(override.Capability)
		if !HasCapability(base, capability) {
			return Adapter{}, fmt.Errorf("adapter %q does not support capability %q", target, capability)
		}
		base.Capabilities = []CapabilityLevel{capability}
		base.Fallback = CapabilityArtifact
	}
	return base, nil
}

func RequireCapability(adapter Adapter, capability CapabilityLevel) error {
	if HasCapability(adapter, capability) {
		return nil
	}
	return fmt.Errorf("adapter %q does not support capability %q; use %q fallback", adapter.Target, capability, adapter.Fallback)
}

func HasCapability(adapter Adapter, capability CapabilityLevel) bool {
	for _, existing := range adapter.Capabilities {
		if existing == capability {
			return true
		}
	}
	return false
}

func CapabilityNames(capabilities []CapabilityLevel) []string {
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, string(capability))
	}
	return out
}

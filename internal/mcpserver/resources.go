package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	MimeType    string `json:"mimeType"`
	Description string `json:"description,omitempty"`
	Text        string `json:"text,omitempty"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Snapshot struct {
	RunID     string
	Bundle    bundle.FailureBundle
	Resources []Resource
	Tools     []Tool
}

func BuildSnapshot(root string, runID string) (Snapshot, error) {
	store := project.NewStore(root)
	if strings.TrimSpace(runID) == "" {
		latest, err := latestRunWithBundle(store)
		if err != nil {
			return Snapshot{}, err
		}
		runID = latest
	}
	run, err := store.OpenRun(runID)
	if err != nil {
		return Snapshot{}, err
	}

	goalData, err := os.ReadFile(project.GoalPath(root))
	if err != nil {
		return Snapshot{}, err
	}
	bundleData, err := run.ReadArtifactFile(project.FailureBundleName)
	if err != nil {
		return Snapshot{}, err
	}
	promptData, err := run.ReadArtifactFile(project.RepairPromptName)
	if err != nil {
		return Snapshot{}, err
	}
	failureBundle, err := bundle.ReadFailureBundle(run)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		RunID:  run.ID,
		Bundle: failureBundle,
		Resources: []Resource{
			{
				URI:         "tailchase://goal",
				Name:        "Current goal",
				MimeType:    "application/yaml",
				Description: "Current Tailchase goal contract.",
				Text:        string(goalData),
			},
			{
				URI:         "tailchase://runs/" + run.ID + "/failure-bundle",
				Name:        "Latest failure bundle",
				MimeType:    "application/yaml",
				Description: "Structured failure context for the selected run.",
				Text:        string(bundleData),
			},
			{
				URI:         "tailchase://runs/" + run.ID + "/repair-prompt",
				Name:        "Next repair instruction",
				MimeType:    "text/markdown",
				Description: "Latest generated repair prompt for the selected run.",
				Text:        string(promptData),
			},
		},
		Tools: []Tool{
			{Name: "tailchase.budget_summary", Description: "Return context budget metadata for the selected failure bundle."},
			{Name: "tailchase.safety_findings", Description: "Return deterministic safety findings for the selected failure bundle."},
		},
	}, nil
}

func (s Snapshot) ResourceList() []Resource {
	resources := make([]Resource, 0, len(s.Resources))
	for _, resource := range s.Resources {
		resource.Text = ""
		resources = append(resources, resource)
	}
	return resources
}

func (s Snapshot) ReadResource(uri string) (Resource, error) {
	for _, resource := range s.Resources {
		if resource.URI == uri {
			return resource, nil
		}
	}
	return Resource{}, fmt.Errorf("unknown resource URI %q", uri)
}

func (s Snapshot) CallTool(name string) (string, error) {
	switch name {
	case "tailchase.budget_summary":
		budget := s.Bundle.Budget
		return fmt.Sprintf("raw_evidence_bytes: %d\nincluded_excerpt_bytes: %d\nrepeated_blocks_collapsed: %d\nestimated_prompt_bytes: %d\n", budget.RawEvidenceBytes, budget.IncludedExcerptBytes, budget.RepeatedBlocksCollapsed, budget.EstimatedPromptBytes), nil
	case "tailchase.safety_findings":
		if len(s.Bundle.SafetyFindings) == 0 {
			return "safety_findings: []\n", nil
		}
		var builder strings.Builder
		builder.WriteString("safety_findings:\n")
		for _, finding := range s.Bundle.SafetyFindings {
			builder.WriteString(fmt.Sprintf("- decision: %s\n  rule: %s\n  message: %s\n", finding.Decision, finding.Rule, finding.Message))
			if finding.Path != "" {
				builder.WriteString(fmt.Sprintf("  path: %s\n", finding.Path))
			}
		}
		return builder.String(), nil
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

func latestRunWithBundle(store project.Store) (string, error) {
	entries, err := os.ReadDir(store.RunsDir())
	if err != nil {
		return "", fmt.Errorf("read runs directory: %w", err)
	}
	type candidate struct {
		runID     string
		createdAt time.Time
	}
	var candidates []candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		run := store.Run(entry.Name())
		if _, err := os.Stat(run.ArtifactPath(project.FailureBundleName)); err != nil {
			continue
		}
		meta, err := run.ReadMetadata()
		if err != nil {
			continue
		}
		createdAt := meta.CreatedAt
		if createdAt.IsZero() {
			if info, err := os.Stat(filepath.Join(store.RunsDir(), entry.Name())); err == nil {
				createdAt = info.ModTime()
			}
		}
		candidates = append(candidates, candidate{runID: entry.Name(), createdAt: createdAt})
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no runs with %s found; pass --run", project.FailureBundleName)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].createdAt.After(candidates[j].createdAt)
	})
	return candidates[0].runID, nil
}

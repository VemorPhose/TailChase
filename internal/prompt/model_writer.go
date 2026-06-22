package prompt

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/model"
	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

type ModelGenerator struct {
	Provider model.Provider
	Now      func() time.Time
}

type modelPromptInput struct {
	Version          int                    `yaml:"version"`
	PromptMode       string                 `yaml:"prompt_mode"`
	Delta            bool                   `yaml:"delta"`
	FailureBundle    bundle.FailureBundle   `yaml:"failure_bundle"`
	AttemptHistory   project.AttemptHistory `yaml:"attempt_history,omitempty"`
	RawEvidenceLinks []string               `yaml:"raw_evidence_links,omitempty"`
	SafetyFindings   []bundle.SafetyFinding `yaml:"safety_findings,omitempty"`
}

func (g ModelGenerator) Generate(ctx context.Context, failureBundle bundle.FailureBundle, cfg project.ModelConfig, opts Options) (Result, error) {
	input := modelPromptInput{
		Version:          project.SchemaVersion,
		PromptMode:       "model",
		Delta:            opts.Delta,
		FailureBundle:    failureBundle,
		AttemptHistory:   opts.AttemptHistory,
		RawEvidenceLinks: collectRawEvidenceLinks(failureBundle),
		SafetyFindings:   failureBundle.SafetyFindings,
	}
	inputData, err := yaml.Marshal(input)
	if err != nil {
		return Result{}, err
	}

	response, err := (model.Client{Provider: g.Provider}).Generate(ctx, model.Request{
		Model: cfg.Model,
		Messages: []model.Message{
			{
				Role:    "system",
				Content: modelSystemInstruction,
			},
			{
				Role:    "user",
				Content: "Write the next Tailchase repair prompt from this deterministic YAML context.\n\n```yaml\n" + string(inputData) + "```",
			},
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("model prompt generation failed: %w", err)
	}

	content := strings.TrimSpace(response.Content)
	if content == "" {
		return Result{}, fmt.Errorf("model prompt generation failed: provider returned an empty prompt")
	}
	content, truncated := applySizeLimit(content+"\n", opts.SizeLimit)

	generatedAt := time.Now().UTC()
	if g.Now != nil {
		generatedAt = g.Now().UTC()
	}
	metadata := &ModelMetadata{
		Version:          project.SchemaVersion,
		Provider:         cfg.Provider,
		Model:            cfg.Model,
		PromptMode:       "model",
		Delta:            opts.Delta,
		GeneratedAt:      generatedAt,
		PromptBytes:      len(content),
		Truncated:        truncated,
		ResponseMetadata: response.Metadata,
	}

	return Result{Content: content, Truncated: truncated, ModelMetadata: metadata}, nil
}

func collectRawEvidenceLinks(failureBundle bundle.FailureBundle) []string {
	seen := map[string]bool{}
	var links []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		links = append(links, path)
	}

	for _, source := range failureBundle.Sources {
		add(source.Path)
	}
	for _, artifact := range failureBundle.Artifacts {
		add(artifact.Path)
	}
	for _, signal := range append(failureBundle.RootErrorCandidates, failureBundle.DownstreamSymptoms...) {
		add(signal.RawExcerptPath)
	}

	sort.Strings(links)
	return links
}

const modelSystemInstruction = `You are Tailchase's model-backed prompt writer.

Write one concise repair prompt for a coding agent.
Preserve the goal, non-goals, stop rules, safety findings, context budget, and raw artifact paths.
Reference raw evidence paths instead of embedding large logs or binary artifacts.
When delta is true, emphasize what changed since prior attempts and avoid repeating unchanged context.
If safety findings include stop decisions, make the stop condition explicit.`

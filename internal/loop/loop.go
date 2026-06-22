package loop

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Agent interface {
	RunAttempt(ctx context.Context, attempt int, prompt string) (AttemptResult, error)
}

type Collector interface {
	Collect(ctx context.Context, attempt int) (EvidenceResult, error)
}

type AttemptResult struct {
	Output   string
	ExitCode int
}

type EvidenceResult struct {
	PromptPath string
	BundlePath string
	Failure    string
}

type Decision struct {
	Attempt   int       `yaml:"attempt"`
	Prompt    string    `yaml:"prompt,omitempty"`
	Bundle    string    `yaml:"bundle,omitempty"`
	ExitCode  int       `yaml:"exit_code"`
	Decision  string    `yaml:"decision"`
	Reason    string    `yaml:"reason"`
	CreatedAt time.Time `yaml:"created_at"`
}

type Result struct {
	Decisions []Decision `yaml:"decisions"`
	Stopped   bool       `yaml:"stopped"`
	Reason    string     `yaml:"reason"`
}

type Options struct {
	Agent         Agent
	Collector     Collector
	InitialPrompt string
	MaxAttempts   int
	Now           func() time.Time
}

func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Agent == nil {
		return Result{}, fmt.Errorf("loop agent is required")
	}
	if opts.Collector == nil {
		return Result{}, fmt.Errorf("loop collector is required")
	}
	if opts.MaxAttempts <= 0 {
		return Result{}, fmt.Errorf("max attempts must be greater than zero")
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	prompt := opts.InitialPrompt
	failures := map[string]int{}
	var result Result

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		agentResult, err := opts.Agent.RunAttempt(ctx, attempt, prompt)
		if err != nil {
			return Result{}, err
		}
		evidence, err := opts.Collector.Collect(ctx, attempt)
		if err != nil {
			return Result{}, err
		}
		decision := Decision{
			Attempt:   attempt,
			Prompt:    evidence.PromptPath,
			Bundle:    evidence.BundlePath,
			ExitCode:  agentResult.ExitCode,
			CreatedAt: now().UTC(),
		}
		switch {
		case agentResult.ExitCode == 0:
			decision.Decision = "stop"
			decision.Reason = "agent attempt succeeded"
			result.Decisions = append(result.Decisions, decision)
			result.Stopped = true
			result.Reason = decision.Reason
			return result, nil
		default:
			fingerprint := failureFingerprint(firstNonEmpty(evidence.Failure, agentResult.Output))
			if fingerprint != "" {
				failures[fingerprint]++
				if failures[fingerprint] >= 2 {
					decision.Decision = "stop"
					decision.Reason = "repeated failure"
					result.Decisions = append(result.Decisions, decision)
					result.Stopped = true
					result.Reason = decision.Reason
					return result, nil
				}
			}
			decision.Decision = "continue"
			decision.Reason = "collect new evidence and generate delta context"
			result.Decisions = append(result.Decisions, decision)
			prompt = evidence.PromptPath
		}
	}
	result.Stopped = true
	result.Reason = "max attempts reached"
	return result, nil
}

func failureFingerprint(value string) string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, strings.ToLower(line))
		if len(lines) == 3 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

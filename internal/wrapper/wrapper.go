package wrapper

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Runner interface {
	Run(ctx context.Context, attempt int) (CommandResult, error)
}

type RunnerFunc func(ctx context.Context, attempt int) (CommandResult, error)

func (f RunnerFunc) Run(ctx context.Context, attempt int) (CommandResult, error) {
	return f(ctx, attempt)
}

type CommandResult struct {
	Command  string
	Output   string
	ExitCode int
}

type Attempt struct {
	Number   int       `yaml:"number"`
	Command  string    `yaml:"command"`
	ExitCode int       `yaml:"exit_code"`
	Output   string    `yaml:"output,omitempty"`
	EndedAt  time.Time `yaml:"ended_at"`
}

type Result struct {
	Attempts []Attempt `yaml:"attempts"`
	Stopped  bool      `yaml:"stopped"`
	Reason   string    `yaml:"reason"`
}

type Options struct {
	Runner      Runner
	MaxAttempts int
	Now         func() time.Time
}

func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Runner == nil {
		return Result{}, fmt.Errorf("wrapper runner is required")
	}
	if opts.MaxAttempts <= 0 {
		return Result{}, fmt.Errorf("max attempts must be greater than zero")
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}

	var result Result
	seenFailures := map[string]int{}
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		commandResult, err := opts.Runner.Run(ctx, attempt)
		if err != nil {
			return Result{}, err
		}
		result.Attempts = append(result.Attempts, Attempt{
			Number:   attempt,
			Command:  commandResult.Command,
			ExitCode: commandResult.ExitCode,
			Output:   commandResult.Output,
			EndedAt:  now().UTC(),
		})
		if commandResult.ExitCode == 0 {
			result.Stopped = true
			result.Reason = "agent command succeeded"
			return result, nil
		}
		fingerprint := failureFingerprint(commandResult.Output)
		if fingerprint != "" {
			seenFailures[fingerprint]++
			if seenFailures[fingerprint] >= 2 {
				result.Stopped = true
				result.Reason = "repeated failure"
				return result, nil
			}
		}
	}
	result.Stopped = true
	result.Reason = "max attempts reached"
	return result, nil
}

func failureFingerprint(output string) string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
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

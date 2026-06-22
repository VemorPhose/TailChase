package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/loop"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const runLoopDecisionLogVersion = 1

func newRunLoopCommand() *cobra.Command {
	var runID string
	var agentTarget string
	var agentCommand string
	var maxAttempts int
	cmd := &cobra.Command{
		Use:   "run-loop",
		Short: "Run a conservative assisted repair loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runLoop(cmd, root, runID, agentTarget, agentCommand, maxAttempts)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to record loop decisions against")
	cmd.Flags().StringVar(&agentTarget, "agent", "", "Agent target name")
	cmd.Flags().StringVar(&agentCommand, "agent-command", "", "Command to run for each attempt")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", 2, "Maximum repair attempts")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("agent")
	_ = cmd.MarkFlagRequired("agent-command")
	return cmd
}

func runLoop(cmd *cobra.Command, root string, runID string, agentTarget string, agentCommand string, maxAttempts int) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	result, err := loop.Run(cmd.Context(), loop.Options{
		Agent:         loopShellAgent{Root: root, Command: agentCommand},
		Collector:     runArtifactCollector{Run: run},
		InitialPrompt: run.RelativePath(run.ArtifactPath(project.RepairPromptName)),
		MaxAttempts:   maxAttempts,
	})
	if err != nil {
		return err
	}
	if err := writeLoopDecisionLog(run, result); err != nil {
		return err
	}
	if err := recordLoopEvent(run, agentTarget, result); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Run loop for %s stopped after %d decision(s): %s\n", agentTarget, len(result.Decisions), result.Reason)
	return nil
}

type loopShellAgent struct {
	Root    string
	Command string
}

func (a loopShellAgent) RunAttempt(ctx context.Context, attempt int, prompt string) (loop.AttemptResult, error) {
	if strings.TrimSpace(a.Command) == "" {
		return loop.AttemptResult{}, fmt.Errorf("agent command is empty")
	}
	command := exec.CommandContext(ctx, "sh", "-c", a.Command)
	command.Dir = a.Root
	command.Env = append(os.Environ(), "TAILCHASE_PROMPT="+prompt)
	output, err := command.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return loop.AttemptResult{Output: string(output), ExitCode: exitCode}, nil
}

type runArtifactCollector struct {
	Run project.Run
}

func (c runArtifactCollector) Collect(ctx context.Context, attempt int) (loop.EvidenceResult, error) {
	failureBundle, err := bundlepkg.ReadFailureBundle(c.Run)
	if err != nil {
		return loop.EvidenceResult{}, err
	}
	failure := ""
	if len(failureBundle.RootErrorCandidates) > 0 {
		failure = failureBundle.RootErrorCandidates[0].Message
	}
	return loop.EvidenceResult{
		PromptPath: c.Run.RelativePath(c.Run.ArtifactPath(project.RepairPromptName)),
		BundlePath: c.Run.RelativePath(c.Run.ArtifactPath(project.FailureBundleName)),
		Failure:    failure,
	}, nil
}

type runLoopDecisionLog struct {
	Version   int             `yaml:"version"`
	Stopped   bool            `yaml:"stopped"`
	Reason    string          `yaml:"reason"`
	Decisions []loop.Decision `yaml:"decisions,omitempty"`
}

func writeLoopDecisionLog(run project.Run, result loop.Result) error {
	data, err := yaml.Marshal(runLoopDecisionLog{
		Version:   runLoopDecisionLogVersion,
		Stopped:   result.Stopped,
		Reason:    result.Reason,
		Decisions: result.Decisions,
	})
	if err != nil {
		return err
	}
	return run.WriteArtifactFile(project.RunLoopDecisionsName, project.ArtifactRunLoopDecisions, "run_loop_decisions", data)
}

func recordLoopEvent(run project.Run, agentTarget string, result loop.Result) error {
	event := guard.Event{
		CreatedAt: time.Now().UTC(),
		Type:      "assisted_repair_loop",
		Message:   fmt.Sprintf("run-loop for %s stopped: %s", agentTarget, result.Reason),
		Findings: []guard.Finding{{
			Rule:     "assisted_repair_loop",
			Decision: "warn",
			Message:  result.Reason,
		}},
	}
	for _, decision := range result.Decisions {
		event.Commands = append(event.Commands, fmt.Sprintf("attempt %d: %s", decision.Attempt, decision.Reason))
	}
	_, err := guard.AppendEvent(run, event)
	return err
}

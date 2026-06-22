package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/VemorPhose/TailChase/internal/wrapper"
	"github.com/spf13/cobra"
)

func newGuardCommand() *cobra.Command {
	var runID string
	var commandLogPath string
	var agentTarget string
	var agentCommand string
	var maxAttempts int
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Run advisory guard checks against local work",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runGuard(cmd, root, runID, commandLogPath, agentTarget, agentCommand, maxAttempts)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to compare against")
	cmd.Flags().StringVar(&commandLogPath, "command-log", "", "Optional command log with lines prefixed by $ or >")
	cmd.Flags().StringVar(&agentTarget, "agent", "", "Opt-in managed agent target")
	cmd.Flags().StringVar(&agentCommand, "agent-command", "", "Command to run in managed wrapper mode")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", 1, "Maximum managed wrapper attempts")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runGuard(cmd *cobra.Command, root string, runID string, commandLogPath string, agentTarget string, agentCommand string, maxAttempts int) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
	}
	if agentTarget != "" {
		return runGuardAgent(cmd, root, run, agentTarget, agentCommand, maxAttempts)
	}
	goal, err := project.LoadGoal(root)
	if err != nil {
		return err
	}
	failureBundle, err := bundlepkg.ReadFailureBundle(run)
	if err != nil {
		return err
	}

	commandLog, err := readOptionalFile(commandLogPath)
	if err != nil {
		return err
	}
	input := guard.Input{
		Goal:           goal,
		FailureBundle:  failureBundle,
		EditedPaths:    gitEditedPaths(cmd, root),
		CommandHistory: guard.ParseCommandLog(commandLog),
		CommandOutput:  commandLog,
	}
	findings := guard.Analyze(input)
	event := guard.BuildEvent(input, findings)
	if _, err := guard.AppendEvent(run, event); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Recorded %d guard finding(s) in %s\n", len(findings), run.RelativePath(run.ArtifactPath(project.SteeringEventsName)))
	for _, finding := range findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", finding.Rule, finding.Message)
	}
	return nil
}

func runGuardAgent(cmd *cobra.Command, root string, run project.Run, agentTarget string, agentCommand string, maxAttempts int) error {
	if strings.TrimSpace(agentCommand) == "" {
		return fmt.Errorf("--agent-command is required when --agent is set")
	}
	result, err := wrapper.Run(cmd.Context(), wrapper.Options{
		Runner:      shellCommandRunner{Root: root, Command: agentCommand},
		MaxAttempts: maxAttempts,
	})
	if err != nil {
		return err
	}
	event := guard.Event{
		CreatedAt: result.Attempts[len(result.Attempts)-1].EndedAt,
		Type:      "managed_agent_wrapper",
		Message:   fmt.Sprintf("agent %s stopped: %s", agentTarget, result.Reason),
		Commands:  wrapperCommands(result.Attempts),
		Findings: []guard.Finding{{
			Rule:     "managed_agent_wrapper",
			Decision: "warn",
			Message:  result.Reason,
		}},
	}
	if _, err := guard.AppendEvent(run, event); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Managed agent %s stopped after %d attempt(s): %s\n", agentTarget, len(result.Attempts), result.Reason)
	return nil
}

func readOptionalFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func gitEditedPaths(cmd *cobra.Command, root string) []string {
	gitCmd := exec.CommandContext(cmd.Context(), "git", "-C", root, "diff", "--name-only")
	out, err := gitCmd.Output()
	if err != nil {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}

type shellCommandRunner struct {
	Root    string
	Command string
}

func (r shellCommandRunner) Run(ctx context.Context, attempt int) (wrapper.CommandResult, error) {
	parts := strings.Fields(r.Command)
	if len(parts) == 0 {
		return wrapper.CommandResult{}, fmt.Errorf("agent command is empty")
	}
	command := exec.CommandContext(ctx, parts[0], parts[1:]...)
	command.Dir = r.Root
	output, err := command.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return wrapper.CommandResult{
		Command:  r.Command,
		Output:   string(output),
		ExitCode: exitCode,
	}, nil
}

func wrapperCommands(attempts []wrapper.Attempt) []string {
	commands := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		commands = append(commands, attempt.Command)
	}
	return commands
}

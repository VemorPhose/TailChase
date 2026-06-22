package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	bundlepkg "github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/guard"
	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newGuardCommand() *cobra.Command {
	var runID string
	var commandLogPath string
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Run advisory guard checks against local work",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runGuard(cmd, root, runID, commandLogPath)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "Run ID to compare against")
	cmd.Flags().StringVar(&commandLogPath, "command-log", "", "Optional command log with lines prefixed by $ or >")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runGuard(cmd *cobra.Command, root string, runID string, commandLogPath string) error {
	run, err := project.NewStore(root).OpenRun(runID)
	if err != nil {
		return err
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

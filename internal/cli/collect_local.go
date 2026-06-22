package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectLocalCommand() *cobra.Command {
	var runID string
	var kind string
	var filePath string
	cmd := &cobra.Command{
		Use:   "collect-local",
		Short: "Import local command or test output into the run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollectLocal(cmd, root, runID, kind, filePath)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "local run ID")
	cmd.Flags().StringVar(&kind, "kind", "", "local evidence kind: go_test or shell")
	cmd.Flags().StringVar(&filePath, "file", "", "path to captured output")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func runCollectLocal(cmd *cobra.Command, root string, runID string, kind string, filePath string) error {
	spec, err := localEvidenceSpec(kind)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	run, err := project.NewStore(root).EnsureRun(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	path := run.EvidencePath(spec.fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	if err := run.RecordArtifact(spec.artifactName, spec.artifactType, path, time.Now().UTC()); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(path))
	return nil
}

type localEvidence struct {
	fileName     string
	artifactName string
	artifactType string
}

func localEvidenceSpec(kind string) (localEvidence, error) {
	switch strings.TrimSpace(kind) {
	case "go_test":
		return localEvidence{fileName: project.GoTestLogName, artifactName: project.ArtifactGoTestLog, artifactType: "local_go_test"}, nil
	case "shell":
		return localEvidence{fileName: project.ShellCommandLogName, artifactName: project.ArtifactShellCommandLog, artifactType: "local_shell"}, nil
	default:
		return localEvidence{}, fmt.Errorf("unsupported local evidence kind %q", kind)
	}
}

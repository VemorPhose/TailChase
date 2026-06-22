package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectComposeCommand() *cobra.Command {
	var runID string
	var service string
	var filePath string
	cmd := &cobra.Command{
		Use:   "collect-compose",
		Short: "Collect Docker Compose service logs into the run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollectCompose(cmd.Context(), cmd, root, runID, service, filePath)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "local run ID")
	cmd.Flags().StringVar(&service, "service", "", "Compose service name")
	cmd.Flags().StringVar(&filePath, "file", "", "path to captured Compose log for the service")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCollectCompose(ctx context.Context, cmd *cobra.Command, root string, runID string, service string, filePath string) error {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	services := composeServices(service, cfg.Compose.Services)
	if len(services) == 0 {
		return fmt.Errorf("no Docker Compose service configured; pass --service or set compose.services")
	}
	run, err := project.NewStore(root).EnsureRun(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	for _, svc := range services {
		var data []byte
		if filePath != "" {
			if len(services) > 1 {
				return fmt.Errorf("--file can only be used with one Compose service")
			}
			data, err = os.ReadFile(filePath)
		} else {
			data, err = dockerComposeLogs(ctx, root, svc, cfg.Compose.TailLines)
		}
		if err != nil {
			return err
		}
		path := run.EvidencePath(filepath.Join(project.ComposeLogsDirName, safeServiceFileName(svc)+".log"))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		if err := run.RecordArtifact(project.ArtifactDockerComposeLog+"_"+safeServiceFileName(svc), "docker_compose", path, time.Now().UTC()); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(path))
	}
	return nil
}

func composeServices(flag string, configured []string) []string {
	if strings.TrimSpace(flag) != "" {
		return []string{strings.TrimSpace(flag)}
	}
	return configured
}

func dockerComposeLogs(ctx context.Context, root string, service string, tailLines int) ([]byte, error) {
	if tailLines == 0 {
		tailLines = 300
	}
	args := []string{"compose", "logs", "--no-color", "--tail", fmt.Sprintf("%d", tailLines), service}
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir = root
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("docker compose logs failed for service %q: %s", service, message)
	}
	return data, nil
}

func safeServiceFileName(service string) string {
	service = strings.TrimSpace(service)
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-")
	return replacer.Replace(service)
}

package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newCollectPlaywrightCommand() *cobra.Command {
	var runID string
	var artifactDir string
	cmd := &cobra.Command{
		Use:   "collect-playwright",
		Short: "Import Playwright traces, screenshots, videos, and console logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCollectPlaywright(cmd, root, runID, artifactDir)
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "local run ID")
	cmd.Flags().StringVar(&artifactDir, "dir", "", "Playwright artifact directory")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func runCollectPlaywright(cmd *cobra.Command, root string, runID string, artifactDir string) error {
	cfg, err := project.LoadConfig(root)
	if err != nil {
		return err
	}
	if strings.TrimSpace(artifactDir) == "" {
		artifactDir = cfg.Playwright.ArtifactDir
	}
	if strings.TrimSpace(artifactDir) == "" {
		return fmt.Errorf("no Playwright artifact directory configured; pass --dir or set playwright.artifact_dir")
	}
	if !filepath.IsAbs(artifactDir) {
		artifactDir = filepath.Join(root, artifactDir)
	}
	run, err := project.NewStore(root).EnsureRun(strings.TrimSpace(runID))
	if err != nil {
		return err
	}
	destRoot := run.EvidencePath(project.PlaywrightDirName)
	written := 0
	err = filepath.WalkDir(artifactDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !isPlaywrightArtifact(path) {
			return nil
		}
		rel, err := filepath.Rel(artifactDir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destRoot, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
		written++
		if err := run.RecordArtifact(fmt.Sprintf("%s_%d", project.ArtifactPlaywrightArtifact, written), "playwright_artifact", dest, time.Now().UTC()); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", run.RelativePath(dest))
		return nil
	})
	if err != nil {
		return err
	}
	if written == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: no Playwright artifacts found in %s\n", artifactDir)
	}
	return nil
}

func isPlaywrightArtifact(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".log", ".json", ".png", ".jpg", ".jpeg", ".webm", ".zip":
		return true
	default:
		return false
	}
}

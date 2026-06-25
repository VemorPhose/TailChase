package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/VemorPhose/TailChase/internal/project"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create .tailchase config and goal files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runInit(cmd, root)
		},
	}
}

func runInit(cmd *cobra.Command, root string) error {
	store := project.NewStore(root)
	if err := store.EnsureProjectDir(); err != nil {
		return err
	}

	configData, err := project.MarshalConfig(project.DefaultConfig())
	if err != nil {
		return err
	}
	goalData, err := project.MarshalGoal(project.DefaultGoal())
	if err != nil {
		return err
	}

	created := []string{}
	if err := writeNewFile(project.ConfigPath(root), configData); err != nil {
		return err
	}
	created = append(created, relPath(root, project.ConfigPath(root)))

	if err := writeNewFile(project.GoalPath(root), goalData); err != nil {
		return err
	}
	created = append(created, relPath(root, project.GoalPath(root)))

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", created[0])
	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", created[1])
	return nil
}

func writeNewFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", path)
		}
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

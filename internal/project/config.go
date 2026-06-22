package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

const (
	DirName        = ".tailchase"
	ConfigFileName = "config.yml"
	SchemaVersion  = 1
)

type Config struct {
	Version           int          `yaml:"version"`
	Collectors        []string     `yaml:"collectors"`
	GitHub            GitHubConfig `yaml:"github"`
	FailedJobsOnly    bool         `yaml:"failed_jobs_only"`
	MaxLogLinesPerJob int          `yaml:"max_log_lines_per_job"`
	PromptTarget      string       `yaml:"prompt_target"`
	PromptSizeLimit   int          `yaml:"prompt_size_limit"`
	Safety            SafetyConfig `yaml:"safety"`
}

type GitHubConfig struct {
	Repo string `yaml:"repo,omitempty"`
}

type SafetyConfig struct {
	Mode   string   `yaml:"mode"`
	StopOn []string `yaml:"stop_on,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Version:           SchemaVersion,
		Collectors:        []string{"github_actions"},
		FailedJobsOnly:    true,
		MaxLogLinesPerJob: 1200,
		PromptTarget:      "stdout",
		PromptSizeLimit:   12000,
		Safety: SafetyConfig{
			Mode: "manual",
			StopOn: []string{
				"test_weakening",
				"suspicious_path_edit",
			},
		},
	}
}

func ConfigPath(root string) string {
	return filepath.Join(root, DirName, ConfigFileName)
}

func LoadConfig(root string) (Config, error) {
	cfg := DefaultConfig()
	if err := loadYAML(ConfigPath(root), &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Version == 0 {
		c.Version = SchemaVersion
	}
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if len(c.Collectors) == 0 {
		return errors.New("collectors must not be empty")
	}
	for _, collector := range c.Collectors {
		if !slices.Contains([]string{"github_actions", "local_go_test", "local_shell"}, collector) {
			return fmt.Errorf("unsupported collector %q", collector)
		}
	}
	if c.MaxLogLinesPerJob <= 0 {
		return errors.New("max_log_lines_per_job must be greater than zero")
	}
	if c.PromptSizeLimit <= 0 {
		return errors.New("prompt_size_limit must be greater than zero")
	}
	if !slices.Contains([]string{"stdout", "file"}, c.PromptTarget) {
		return errors.New("prompt_target must be stdout or file")
	}
	if c.Safety.Mode == "" {
		c.Safety.Mode = "manual"
	}
	if c.Safety.Mode != "manual" {
		return errors.New("safety.mode must be manual")
	}
	return nil
}

func MarshalConfig(cfg Config) ([]byte, error) {
	if cfg.Version == 0 {
		cfg.Version = SchemaVersion
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return yaml.Marshal(cfg)
}

func loadYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist; run tailchase init first", path)
		}
		return err
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

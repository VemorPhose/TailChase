package project

import (
	"errors"
	"fmt"
	"net/url"
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
	Version           int              `yaml:"version"`
	Collectors        []string         `yaml:"collectors"`
	GitHub            GitHubConfig     `yaml:"github"`
	GitLab            GitLabConfig     `yaml:"gitlab,omitempty"`
	FailedJobsOnly    bool             `yaml:"failed_jobs_only"`
	MaxLogLinesPerJob int              `yaml:"max_log_lines_per_job"`
	PromptTarget      string           `yaml:"prompt_target"`
	PromptSizeLimit   int              `yaml:"prompt_size_limit"`
	Prompt            PromptConfig     `yaml:"prompt"`
	Model             ModelConfig      `yaml:"model,omitempty"`
	ReportGlobs       []string         `yaml:"report_globs,omitempty"`
	Compose           ComposeConfig    `yaml:"compose,omitempty"`
	Playwright        PlaywrightConfig `yaml:"playwright,omitempty"`
	Adapters          []AdapterConfig  `yaml:"adapters,omitempty"`
	Safety            SafetyConfig     `yaml:"safety"`
}

type GitHubConfig struct {
	Repo string `yaml:"repo,omitempty"`
}

type GitLabConfig struct {
	Project string `yaml:"project,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`
}

type PromptConfig struct {
	Mode string `yaml:"mode"`
}

type ModelConfig struct {
	Provider  string `yaml:"provider,omitempty"`
	BaseURL   string `yaml:"base_url,omitempty"`
	Model     string `yaml:"model,omitempty"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

type SafetyConfig struct {
	Mode   string   `yaml:"mode"`
	StopOn []string `yaml:"stop_on,omitempty"`
}

type ComposeConfig struct {
	Services  []string `yaml:"services,omitempty"`
	TailLines int      `yaml:"tail_lines,omitempty"`
}

type PlaywrightConfig struct {
	ArtifactDir string `yaml:"artifact_dir,omitempty"`
}

type AdapterConfig struct {
	Target     string `yaml:"target"`
	Capability string `yaml:"capability"`
}

func DefaultConfig() Config {
	return Config{
		Version:           SchemaVersion,
		Collectors:        []string{"github_actions"},
		FailedJobsOnly:    true,
		MaxLogLinesPerJob: 1200,
		GitLab:            GitLabConfig{BaseURL: "https://gitlab.com"},
		PromptTarget:      "stdout",
		PromptSizeLimit:   12000,
		Prompt:            PromptConfig{Mode: "heuristic"},
		Model: ModelConfig{
			Provider:  "openai_compatible",
			APIKeyEnv: "OPENAI_API_KEY",
		},
		Compose: ComposeConfig{TailLines: 300},
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
		if !slices.Contains([]string{"github_actions", "gitlab_ci", "local_go_test", "local_shell"}, collector) {
			return fmt.Errorf("unsupported collector %q", collector)
		}
	}
	if c.GitLab.BaseURL != "" {
		parsed, err := url.Parse(c.GitLab.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.New("gitlab.base_url must be an absolute URL")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("gitlab.base_url must use http or https")
		}
	}
	if slices.Contains(c.Collectors, "gitlab_ci") && c.GitLab.Project == "" {
		return errors.New("gitlab.project is required when collectors includes gitlab_ci")
	}
	if c.MaxLogLinesPerJob <= 0 {
		return errors.New("max_log_lines_per_job must be greater than zero")
	}
	if c.PromptSizeLimit <= 0 {
		return errors.New("prompt_size_limit must be greater than zero")
	}
	if c.Compose.TailLines < 0 {
		return errors.New("compose.tail_lines must not be negative")
	}
	if !slices.Contains([]string{"stdout", "file"}, c.PromptTarget) {
		return errors.New("prompt_target must be stdout or file")
	}
	if c.Prompt.Mode == "" {
		c.Prompt.Mode = "heuristic"
	}
	if !slices.Contains([]string{"heuristic", "model"}, c.Prompt.Mode) {
		return errors.New("prompt.mode must be heuristic or model")
	}
	if c.Prompt.Mode == "model" {
		if c.Model.Provider != "openai_compatible" {
			return errors.New("model.provider must be openai_compatible")
		}
		if c.Model.BaseURL == "" {
			return errors.New("model.base_url is required when prompt.mode is model")
		}
		if c.Model.Model == "" {
			return errors.New("model.model is required when prompt.mode is model")
		}
		if c.Model.APIKeyEnv == "" {
			return errors.New("model.api_key_env is required when prompt.mode is model")
		}
	}
	if c.Safety.Mode == "" {
		c.Safety.Mode = "manual"
	}
	if c.Safety.Mode != "manual" {
		return errors.New("safety.mode must be manual")
	}
	for _, adapter := range c.Adapters {
		if !slices.Contains([]string{"codex", "claude-code", "copilot", "cursor-vscode", "generic"}, adapter.Target) {
			return fmt.Errorf("unsupported adapter target %q", adapter.Target)
		}
		if !slices.Contains([]string{"artifact", "queued", "checkpoint", "hook_mcp", "wrapper"}, adapter.Capability) {
			return fmt.Errorf("unsupported adapter capability %q", adapter.Capability)
		}
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

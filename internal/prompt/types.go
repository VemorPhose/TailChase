package prompt

import (
	"time"

	"github.com/VemorPhose/TailChase/internal/project"
)

type Options struct {
	SizeLimit      int
	Delta          bool
	AttemptHistory project.AttemptHistory
}

type Result struct {
	Content       string
	Truncated     bool
	ModelMetadata *ModelMetadata
}

type ModelMetadata struct {
	Version          int               `yaml:"version"`
	Provider         string            `yaml:"provider"`
	Model            string            `yaml:"model"`
	PromptMode       string            `yaml:"prompt_mode"`
	Delta            bool              `yaml:"delta"`
	GeneratedAt      time.Time         `yaml:"generated_at"`
	PromptBytes      int               `yaml:"prompt_bytes"`
	Truncated        bool              `yaml:"truncated"`
	ResponseMetadata map[string]string `yaml:"response_metadata,omitempty"`
}

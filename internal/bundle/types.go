package bundle

import "time"

const schemaVersion = 1

type EvidenceSource struct {
	Source string `yaml:"source"`
	Path   string `yaml:"path"`
	Job    string `yaml:"job,omitempty"`
	JobID  int64  `yaml:"job_id,omitempty"`
}

type Signal struct {
	Type           string `yaml:"type"`
	Source         string `yaml:"source"`
	Job            string `yaml:"job,omitempty"`
	Message        string `yaml:"message"`
	File           string `yaml:"file,omitempty"`
	Line           int    `yaml:"line,omitempty"`
	Confidence     string `yaml:"confidence"`
	RawExcerpt     string `yaml:"raw_excerpt,omitempty"`
	RawExcerptPath string `yaml:"raw_excerpt_path,omitempty"`
}

type NormalizedEvidence struct {
	Version     int              `yaml:"version"`
	GeneratedAt time.Time        `yaml:"generated_at"`
	Sources     []EvidenceSource `yaml:"sources"`
	Signals     []Signal         `yaml:"signals"`
	Warnings    []string         `yaml:"warnings,omitempty"`
}

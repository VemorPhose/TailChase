package bundle

import "time"

const schemaVersion = 1

type EvidenceSource struct {
	Source string `yaml:"source"`
	Path   string `yaml:"path"`
	Job    string `yaml:"job,omitempty"`
	JobID  int64  `yaml:"job_id,omitempty"`
}

type RunMetadata struct {
	Source      string `yaml:"source"`
	Repository  string `yaml:"repository,omitempty"`
	RunID       string `yaml:"run_id,omitempty"`
	CollectedAt string `yaml:"collected_at,omitempty"`
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
	Run         RunMetadata      `yaml:"run"`
	Sources     []EvidenceSource `yaml:"sources"`
	Signals     []Signal         `yaml:"signals"`
	Warnings    []string         `yaml:"warnings,omitempty"`
}

type GoalContract struct {
	Goal            string   `yaml:"goal"`
	NonGoals        []string `yaml:"non_goals,omitempty"`
	MustPreserve    []string `yaml:"must_preserve,omitempty"`
	DoneConditions  []string `yaml:"done_conditions,omitempty"`
	SuspiciousPaths []string `yaml:"suspicious_paths,omitempty"`
}

type Artifact struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type FailureBundle struct {
	Version             int              `yaml:"version"`
	GeneratedAt         time.Time        `yaml:"generated_at"`
	Run                 RunMetadata      `yaml:"run"`
	Goal                GoalContract     `yaml:"goal"`
	Sources             []EvidenceSource `yaml:"sources"`
	RootErrorCandidates []Signal         `yaml:"root_error_candidates"`
	DownstreamSymptoms  []Signal         `yaml:"downstream_symptoms,omitempty"`
	Artifacts           []Artifact       `yaml:"artifacts"`
	Warnings            []string         `yaml:"warnings,omitempty"`
}

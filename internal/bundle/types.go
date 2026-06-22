package bundle

import "time"

const SchemaVersion = 1

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
	ExpectedPaths   []string `yaml:"expected_paths,omitempty"`
	SuspiciousPaths []string `yaml:"suspicious_paths,omitempty"`
	StopRules       []string `yaml:"stop_rules,omitempty"`
}

type Artifact struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type AttemptContext struct {
	SameRootErrorSeenBefore bool  `yaml:"same_root_error_seen_before"`
	MatchingAttemptNumbers  []int `yaml:"matching_attempt_numbers,omitempty"`
}

type BudgetMetadata struct {
	RawEvidenceBytes        int64 `yaml:"raw_evidence_bytes"`
	IncludedExcerptBytes    int64 `yaml:"included_excerpt_bytes"`
	RepeatedBlocksCollapsed int   `yaml:"repeated_blocks_collapsed"`
	EstimatedPromptBytes    int64 `yaml:"estimated_prompt_bytes"`
}

type SafetyFinding struct {
	Rule     string `yaml:"rule"`
	Decision string `yaml:"decision"`
	Message  string `yaml:"message"`
	Path     string `yaml:"path,omitempty"`
}

type FailureBundle struct {
	Version             int              `yaml:"version"`
	GeneratedAt         time.Time        `yaml:"generated_at"`
	Run                 RunMetadata      `yaml:"run"`
	Goal                GoalContract     `yaml:"goal"`
	Sources             []EvidenceSource `yaml:"sources"`
	AttemptContext      AttemptContext   `yaml:"attempt_context"`
	Budget              BudgetMetadata   `yaml:"budget"`
	RootErrorCandidates []Signal         `yaml:"root_error_candidates"`
	DownstreamSymptoms  []Signal         `yaml:"downstream_symptoms,omitempty"`
	SafetyFindings      []SafetyFinding  `yaml:"safety_findings,omitempty"`
	Artifacts           []Artifact       `yaml:"artifacts"`
	Warnings            []string         `yaml:"warnings,omitempty"`
}

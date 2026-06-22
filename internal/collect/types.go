package collect

import (
	"context"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
)

type ProviderMetadata struct {
	Name string
	Kind string
}

type ProviderCollector[Options any] interface {
	ProviderMetadata() ProviderMetadata
	Collect(ctx context.Context, run project.Run, opts Options) (Result, error)
}

type GitHubActionsOptions struct {
	Owner             string
	Repo              string
	RunID             int64
	FailedJobsOnly    bool
	MaxLogLinesPerJob int
}

type GitLabCIOptions struct {
	Project           string
	PipelineID        int64
	FailedJobsOnly    bool
	MaxLogLinesPerJob int
}

type Result struct {
	Repository   string
	RunID        int64
	Provider     ProviderMetadata
	EvidencePath string
	Sources      []bundle.EvidenceSource
	Signals      []bundle.Signal
	Jobs         []JobResult
	Warnings     []string
	CollectedAt  time.Time
}

type JobResult struct {
	ID           int64
	Name         string
	Status       string
	Conclusion   string
	HTMLURL      string
	LinesWritten int
	Truncated    bool
}

func EvidenceSource(source string, provider ProviderMetadata, path string) bundle.EvidenceSource {
	return bundle.EvidenceSource{
		Source:       source,
		Provider:     provider.Name,
		ProviderKind: provider.Kind,
		Path:         path,
	}
}

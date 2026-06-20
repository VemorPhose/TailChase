package collect

import "time"

type GitHubActionsOptions struct {
	Owner             string
	Repo              string
	RunID             int64
	FailedJobsOnly    bool
	MaxLogLinesPerJob int
}

type Result struct {
	Repository   string
	RunID        int64
	EvidencePath string
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

package guard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VemorPhose/TailChase/internal/bundle"
	"github.com/VemorPhose/TailChase/internal/project"
	"gopkg.in/yaml.v3"
)

const SchemaVersion = 1

type Input struct {
	Goal           project.Goal
	FailureBundle  bundle.FailureBundle
	EditedPaths    []string
	CommandHistory []string
	CommandOutput  string
	Now            func() time.Time
}

type Finding struct {
	Rule     string `yaml:"rule"`
	Decision string `yaml:"decision"`
	Message  string `yaml:"message"`
	Path     string `yaml:"path,omitempty"`
}

type EventLog struct {
	Version int     `yaml:"version"`
	Events  []Event `yaml:"events,omitempty"`
}

type Event struct {
	CreatedAt      time.Time `yaml:"created_at"`
	Type           string    `yaml:"type"`
	Message        string    `yaml:"message"`
	EditedPaths    []string  `yaml:"edited_paths,omitempty"`
	Commands       []string  `yaml:"commands,omitempty"`
	Findings       []Finding `yaml:"findings,omitempty"`
	FailureRunID   string    `yaml:"failure_run_id,omitempty"`
	FailureSummary string    `yaml:"failure_summary,omitempty"`
}

func Analyze(input Input) []Finding {
	var findings []Finding
	findings = append(findings, suspiciousPathFindings(input.Goal, input.EditedPaths)...)
	findings = append(findings, repeatedCommandFindings(input.CommandHistory)...)
	findings = append(findings, knownFailureFindings(input.FailureBundle, input.CommandOutput)...)
	return findings
}

func BuildEvent(input Input, findings []Finding) Event {
	now := time.Now().UTC()
	if input.Now != nil {
		now = input.Now().UTC()
	}
	return Event{
		CreatedAt:      now,
		Type:           "guard_check",
		Message:        fmt.Sprintf("guard produced %d finding(s)", len(findings)),
		EditedPaths:    append([]string{}, input.EditedPaths...),
		Commands:       append([]string{}, input.CommandHistory...),
		Findings:       findings,
		FailureRunID:   input.FailureBundle.Run.RunID,
		FailureSummary: firstRootMessage(input.FailureBundle),
	}
}

func AppendEvent(run project.Run, event Event) (EventLog, error) {
	log, err := ReadEventLog(run)
	if err != nil {
		return EventLog{}, err
	}
	log.Version = SchemaVersion
	log.Events = append(log.Events, event)
	data, err := yaml.Marshal(log)
	if err != nil {
		return EventLog{}, err
	}
	if err := run.WriteArtifactFile(project.SteeringEventsName, project.ArtifactSteeringEvents, "steering_events", data); err != nil {
		return EventLog{}, err
	}
	return log, nil
}

func ReadEventLog(run project.Run) (EventLog, error) {
	data, err := run.ReadArtifactFile(project.SteeringEventsName)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "is missing") {
			return EventLog{Version: SchemaVersion}, nil
		}
		return EventLog{}, err
	}
	var log EventLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return EventLog{}, err
	}
	if log.Version == 0 {
		log.Version = SchemaVersion
	}
	if log.Version != SchemaVersion {
		return EventLog{}, fmt.Errorf("unsupported steering events version %d", log.Version)
	}
	return log, nil
}

func ParseCommandLog(data string) []string {
	var commands []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "$ "):
			commands = append(commands, strings.TrimSpace(strings.TrimPrefix(line, "$ ")))
		case strings.HasPrefix(line, "> "):
			commands = append(commands, strings.TrimSpace(strings.TrimPrefix(line, "> ")))
		}
	}
	return commands
}

func suspiciousPathFindings(goal project.Goal, editedPaths []string) []Finding {
	var findings []Finding
	for _, editedPath := range editedPaths {
		for _, suspicious := range goal.SuspiciousPaths {
			if pathMatches(suspicious, editedPath) {
				findings = append(findings, Finding{
					Rule:     "suspicious_path_edit",
					Decision: "warn",
					Message:  fmt.Sprintf("edited path %q matches suspicious path %q", editedPath, suspicious),
					Path:     editedPath,
				})
			}
		}
	}
	return findings
}

func repeatedCommandFindings(commands []string) []Finding {
	counts := map[string]int{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		counts[command]++
		if counts[command] == 3 {
			return []Finding{{
				Rule:     "repeated_command_loop",
				Decision: "warn",
				Message:  fmt.Sprintf("command %q was observed 3 times", command),
			}}
		}
	}
	return nil
}

func knownFailureFindings(failureBundle bundle.FailureBundle, output string) []Finding {
	output = strings.ToLower(output)
	for _, signal := range failureBundle.RootErrorCandidates {
		message := strings.TrimSpace(signal.Message)
		if message == "" {
			continue
		}
		if strings.Contains(output, strings.ToLower(message)) {
			return []Finding{{
				Rule:     "known_failure_repeated",
				Decision: "warn",
				Message:  fmt.Sprintf("command output still contains known root failure %q", message),
				Path:     signal.File,
			}}
		}
	}
	return nil
}

func firstRootMessage(failureBundle bundle.FailureBundle) string {
	if len(failureBundle.RootErrorCandidates) == 0 {
		return ""
	}
	return failureBundle.RootErrorCandidates[0].Message
}

func pathMatches(pattern string, path string) bool {
	pattern = filepath.Clean(strings.TrimSpace(pattern))
	path = filepath.Clean(strings.TrimSpace(path))
	if pattern == "." || pattern == "" || path == "" {
		return false
	}
	return path == pattern || strings.HasPrefix(path, pattern+string(filepath.Separator)) || strings.HasPrefix(path, filepath.ToSlash(pattern)+"/")
}

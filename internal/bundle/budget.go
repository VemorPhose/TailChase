package bundle

import (
	"fmt"
	"os"
	"strings"

	"github.com/VemorPhose/TailChase/internal/project"
)

func compactSignalExcerpts(signals []Signal) ([]Signal, int) {
	compacted := append([]Signal(nil), signals...)
	collapsed := 0
	for i := range compacted {
		excerpt, count := collapseRepeatedLogBlocks(compacted[i].RawExcerpt)
		compacted[i].RawExcerpt = excerpt
		collapsed += count
	}
	return compacted, collapsed
}

func collapseRepeatedLogBlocks(excerpt string) (string, int) {
	excerpt = strings.TrimSpace(excerpt)
	if excerpt == "" {
		return "", 0
	}

	lines := strings.Split(excerpt, "\n")
	var output []string
	collapsed := 0
	for i := 0; i < len(lines); {
		size, repeats := repeatedBlock(lines, i)
		if repeats > 1 {
			output = append(output, lines[i:i+size]...)
			output = append(output, fmt.Sprintf("[tailchase] repeated previous %d-line block %d more time(s)", size, repeats-1))
			collapsed += repeats - 1
			i += size * repeats
			continue
		}
		output = append(output, lines[i])
		i++
	}

	compacted := strings.TrimSpace(strings.Join(output, "\n"))
	if len(compacted) >= len(excerpt) {
		return excerpt, 0
	}
	return compacted, collapsed
}

func repeatedBlock(lines []string, start int) (int, int) {
	remaining := len(lines) - start
	maxSize := remaining / 2
	if maxSize > 8 {
		maxSize = 8
	}
	for size := 1; size <= maxSize; size++ {
		repeats := 1
		for start+(repeats+1)*size <= len(lines) &&
			equalLineBlock(lines[start:start+size], lines[start+repeats*size:start+(repeats+1)*size]) {
			repeats++
		}
		if repeats > 1 && (size > 1 || repeats > 2) {
			return size, repeats
		}
	}
	return 0, 1
}

func equalLineBlock(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if strings.TrimSpace(left[i]) != strings.TrimSpace(right[i]) {
			return false
		}
	}
	return true
}

func rawEvidenceBytes(run project.Run, sources []EvidenceSource) int64 {
	seen := map[string]bool{}
	var total int64
	for _, source := range sources {
		if strings.TrimSpace(source.Path) == "" {
			continue
		}
		path := run.AbsolutePath(source.Path)
		if seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
	}
	return total
}

func includedExcerptBytes(signals ...[]Signal) int64 {
	var total int64
	for _, group := range signals {
		for _, signal := range group {
			total += int64(len([]byte(signal.RawExcerpt)))
		}
	}
	return total
}

func estimatePromptBytes(bundle FailureBundle) int64 {
	total := int64(2048)
	total += int64(len(bundle.Run.Repository) + len(bundle.Run.RunID) + len(bundle.Run.Source))
	total += stringSliceBytes(bundle.Goal.NonGoals)
	total += stringSliceBytes(bundle.Goal.MustPreserve)
	total += stringSliceBytes(bundle.Goal.DoneConditions)
	total += int64(len(bundle.Goal.Goal))
	total += signalPromptBytes(bundle.RootErrorCandidates)
	total += signalPromptBytes(bundle.DownstreamSymptoms)
	total += stringSliceBytes(bundle.Warnings)
	for _, artifact := range bundle.Artifacts {
		total += int64(len(artifact.Name) + len(artifact.Path) + 8)
	}
	return total
}

func signalPromptBytes(signals []Signal) int64 {
	var total int64
	for _, signal := range signals {
		total += int64(len(signal.Type) + len(signal.Job) + len(signal.Message) + len(signal.File) + len(signal.Confidence))
		total += int64(len(signal.RawExcerpt) + len(signal.RawExcerptPath) + 64)
	}
	return total
}

func stringSliceBytes(values []string) int64 {
	var total int64
	for _, value := range values {
		total += int64(len(value) + 4)
	}
	return total
}

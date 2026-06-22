package bundle

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/VemorPhose/TailChase/internal/project"
)

var rootLocationPattern = regexp.MustCompile(`(?:^|\s)(?:[A-Za-z]:)?[A-Za-z0-9_./\\-]+\.(?:go|ts|tsx|js|jsx|py|rb|rs|java|kt|cs|php|c|cc|cpp|h|hpp|sql|yaml|yml):\d+(?::\d+)?:?`)

func attemptContext(rootCandidates []Signal, history project.AttemptHistory) AttemptContext {
	current := map[string]bool{}
	for _, signal := range rootCandidates {
		fingerprint := RootErrorFingerprint(signal.Message)
		if fingerprint != "" {
			current[fingerprint] = true
		}
	}
	if len(current) == 0 {
		return AttemptContext{}
	}

	matches := map[int]bool{}
	for _, attempt := range history.Attempts {
		for _, candidate := range attempt.RootErrorCandidates {
			if current[RootErrorFingerprint(candidate)] {
				matches[attempt.Number] = true
				break
			}
		}
	}
	if len(matches) == 0 {
		return AttemptContext{}
	}

	numbers := make([]int, 0, len(matches))
	for number := range matches {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	return AttemptContext{SameRootErrorSeenBefore: true, MatchingAttemptNumbers: numbers}
}

func RootErrorFingerprint(message string) string {
	message = strings.TrimSpace(strings.ToLower(message))
	if message == "" {
		return ""
	}
	message = strings.ReplaceAll(message, "\\", "/")
	message = rootLocationPattern.ReplaceAllString(message, " ")
	message = strings.Trim(message, "`'\" ")
	return strings.Join(strings.Fields(message), " ")
}

func repeatedRootWarning(context AttemptContext) string {
	if !context.SameRootErrorSeenBefore {
		return ""
	}
	return fmt.Sprintf("same root error seen before in attempt(s): %s", joinAttemptNumbers(context.MatchingAttemptNumbers))
}

func joinAttemptNumbers(numbers []int) string {
	parts := make([]string, 0, len(numbers))
	for _, number := range numbers {
		parts = append(parts, strconv.Itoa(number))
	}
	return strings.Join(parts, ", ")
}

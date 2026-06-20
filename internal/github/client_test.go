package github

import "testing"

func TestParseRepository(t *testing.T) {
	tests := map[string]string{
		"owner/repo":                          "owner/repo",
		"https://github.com/owner/repo":       "owner/repo",
		"https://github.com/owner/repo.git":   "owner/repo",
		"git@github.com:owner/repo.git":       "owner/repo",
		"ssh://git@github.com/owner/repo.git": "owner/repo",
	}

	for input, want := range tests {
		got, err := ParseRepository(input)
		if err != nil {
			t.Fatalf("ParseRepository(%q) error = %v", input, err)
		}
		if got.String() != want {
			t.Fatalf("ParseRepository(%q) = %q, want %q", input, got.String(), want)
		}
	}
}

func TestParseRepositoryRejectsUnsupportedInput(t *testing.T) {
	tests := []string{
		"",
		"repo-only",
		"https://example.com/owner/repo",
		"owner/repo/extra",
		"owner /repo",
	}

	for _, input := range tests {
		if _, err := ParseRepository(input); err == nil {
			t.Fatalf("ParseRepository(%q) error = nil, want error", input)
		}
	}
}

func TestResolveRepositoryPrecedence(t *testing.T) {
	repo, source, err := ResolveRepository(t.TempDir(), "flag/repo", "config/repo")
	if err != nil {
		t.Fatalf("ResolveRepository() error = %v", err)
	}
	if repo.String() != "flag/repo" || source != "flag" {
		t.Fatalf("ResolveRepository() = %q from %q, want flag/repo from flag", repo.String(), source)
	}
}

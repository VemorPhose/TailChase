package github

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	gh "github.com/google/go-github/v72/github"
)

type Repository struct {
	Owner string
	Name  string
}

func (r Repository) String() string {
	if r.Owner == "" || r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

func NewClient(token string) *gh.Client {
	client := gh.NewClient(nil)
	if strings.TrimSpace(token) == "" {
		return client
	}
	return client.WithAuthToken(token)
}

func TokenFromEnv() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GH_TOKEN"))
}

func ResolveRepository(root string, explicit string, configured string) (Repository, string, error) {
	if strings.TrimSpace(explicit) != "" {
		repo, err := ParseRepository(explicit)
		return repo, "flag", err
	}
	if strings.TrimSpace(configured) != "" {
		repo, err := ParseRepository(configured)
		return repo, "config", err
	}

	remote, err := remoteOriginURL(root)
	if err != nil {
		return Repository{}, "", errors.New("repository is required; pass --repo owner/name or set github.repo in .tailchase/config.yml")
	}
	repo, err := ParseRepository(remote)
	return repo, "git remote origin", err
}

func ParseRepository(value string) (Repository, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return Repository{}, errors.New("repository must not be empty")
	}

	switch {
	case strings.HasPrefix(raw, "git@github.com:"):
		raw = strings.TrimPrefix(raw, "git@github.com:")
	case strings.Contains(raw, "://"):
		parsed, err := url.Parse(raw)
		if err != nil {
			return Repository{}, fmt.Errorf("parse repository URL: %w", err)
		}
		if !strings.EqualFold(parsed.Hostname(), "github.com") {
			return Repository{}, fmt.Errorf("repository URL host must be github.com, got %q", parsed.Hostname())
		}
		raw = strings.TrimPrefix(parsed.Path, "/")
	}

	raw = strings.TrimSuffix(raw, ".git")
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Repository{}, fmt.Errorf("repository %q must be owner/name", value)
	}
	if strings.ContainsAny(parts[0]+parts[1], " \t\r\n") {
		return Repository{}, fmt.Errorf("repository %q must not contain whitespace", value)
	}
	return Repository{Owner: parts[0], Name: parts[1]}, nil
}

func remoteOriginURL(root string) (string, error) {
	cmd := exec.Command("git", "-C", root, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v72/github"
)

type issueCommentClient interface {
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *gh.IssueComment) (*gh.IssueComment, *gh.Response, error)
}

type PullRequestCommenter struct {
	Issues issueCommentClient
}

func NewPullRequestCommenter(client *gh.Client) PullRequestCommenter {
	if client == nil {
		return PullRequestCommenter{}
	}
	return PullRequestCommenter{Issues: client.Issues}
}

func (c PullRequestCommenter) Post(ctx context.Context, repo Repository, prNumber int, body string) error {
	if c.Issues == nil {
		return fmt.Errorf("github issue comment client is required")
	}
	if repo.Owner == "" || repo.Name == "" {
		return fmt.Errorf("github repository is required")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be greater than zero")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("comment body must not be empty")
	}
	if _, _, err := c.Issues.CreateComment(ctx, repo.Owner, repo.Name, prNumber, &gh.IssueComment{Body: gh.Ptr(body)}); err != nil {
		return fmt.Errorf("post comment to %s#%d: %w", repo.String(), prNumber, err)
	}
	return nil
}

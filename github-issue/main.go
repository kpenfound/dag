// A generated module for GithubIssue functions

package main

import (
	"context"
	"dagger/github-issue/internal/dagger"
	"fmt"
	"strings"

	"github.com/google/go-github/v59/github"
)

type GithubIssue struct {
	// Github Token
	// +private
	Token *dagger.Secret
}

// Crate a new GithubIssue object
func New(
	// Github authentication token
	token *dagger.Secret,
) *GithubIssue {
	return &GithubIssue{Token: token}
}

type GithubIssueData struct {
	IssueNumber int
	Title       string
	Body        string
}

// Returns a container that echoes whatever string argument is provided
func (m *GithubIssue) Read(ctx context.Context, repo string, issueID int) (*GithubIssueData, error) {
	return loadGithubIssue(ctx, m.Token, repo, issueID)
}

// TODO: Not yet implemented
func (m *GithubIssue) Write(ctx context.Context, repo, title, body string) (*GithubIssueData, error) {
	return nil, nil
}

// // TODO
// func (m *GithubIssue) ReadComments(ctx context.Context) {}

// // TODO
// func (m *GithubIssue) WriteComment(ctx context.Context) {}

func loadGithubIssue(ctx context.Context, token *dagger.Secret, repo string, id int) (*GithubIssueData, error) {
	// Strip .git suffix if present
	repo = strings.TrimSuffix(repo, ".git")

	// Remove https:// or http:// prefix if present
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")

	// Remove github.com/ prefix if present
	repo = strings.TrimPrefix(repo, "github.com/")

	// Split remaining string into owner/repo
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repo)
	}

	owner := parts[0]
	repo = parts[1]

	ghClient, err := githubClient(ctx, token)
	if err != nil {
		return nil, err
	}

	issue, _, err := ghClient.Issues.Get(ctx, owner, repo, id)
	if err != nil {
		return nil, err
	}

	ghi := &GithubIssueData{IssueNumber: id}
	if issue.Title != nil {
		ghi.Title = *issue.Title
	}
	if issue.Body != nil {
		ghi.Body = *issue.Body
	}

	return ghi, nil
}

func githubClient(ctx context.Context, token *dagger.Secret) (*github.Client, error) {
	plaintoken, err := token.Plaintext(ctx)
	if err != nil {
		return nil, err
	}
	return github.NewClient(nil).WithAuthToken(plaintoken), nil
}

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
	HeadRef     string
	BaseRef     string
}

// Returns a container that echoes whatever string argument is provided
func (m *GithubIssue) Read(ctx context.Context, repo string, issueID int) (*GithubIssueData, error) {
	return loadGithubIssueData(ctx, m.Token, repo, issueID)
}

// TODO: Not yet implemented
func (m *GithubIssue) Write(ctx context.Context, repo, title, body string) (*GithubIssueData, error) {
	return nil, nil
}

// // TODO
// func (m *GithubIssue) ReadComments(ctx context.Context) {}

// Write a comment on a Github issue
func (m *GithubIssue) WriteComment(ctx context.Context, repo string, issueID int, body string) error {
	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return err
	}
	issue, err := loadGithubIssue(ctx, m.Token, repo, issueID)
	if err != nil {
		return err
	}

	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return err
	}

	if issue.IsPullRequest() {
		_, _, err = ghClient.PullRequests.CreateComment(ctx, owner, repoName, issueID, &github.PullRequestComment{
			Body: &body,
		})
		if err != nil {
			return err
		}
	} else {
		_, _, err = ghClient.Issues.CreateComment(ctx, owner, repoName, issueID, &github.IssueComment{
			Body: &body,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Write a comment on a Github issue
func (m *GithubIssue) WritePullRequestCodeComment(
	ctx context.Context,
	repo string,
	issueID int,
	commit string,
	body string,
	path string,
	side string,
	line int,
) error {
	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return err
	}
	issue, err := loadGithubIssue(ctx, m.Token, repo, issueID)
	if err != nil {
		return err
	}

	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return err
	}

	if !issue.IsPullRequest() {
		return fmt.Errorf("issue is not a pull request")
	}
	_, _, err = ghClient.PullRequests.CreateComment(ctx, owner, repoName, issueID, &github.PullRequestComment{
		Body:     &body,
		CommitID: &commit,
		Path:     &path,
		Side:     &side,
		Line:     &line,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *GithubIssue) GetPrForCommit(ctx context.Context, repo string, commit string) (int, error) {
	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return 0, err
	}
	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return 0, err
	}
	pulls, _, err := ghClient.PullRequests.ListPullRequestsWithCommit(ctx, owner, repoName, commit, nil)
	if err != nil {
		return 0, err
	}
	if len(pulls) == 0 {
		return 0, fmt.Errorf("no pull requests found for commit %s", commit)
	}
	return *pulls[0].Number, nil
}

func parseOwnerAndRepo(repo string) (string, string, error) {
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
		return "", "", fmt.Errorf("invalid repository format: %s", repo)
	}

	return parts[0], parts[1], nil
}

func loadGithubIssue(ctx context.Context, token *dagger.Secret, repo string, id int) (*github.Issue, error) {
	owner, repo, err := parseOwnerAndRepo(repo)
	if err != nil {
		return nil, err
	}

	ghClient, err := githubClient(ctx, token)
	if err != nil {
		return nil, err
	}

	issue, _, err := ghClient.Issues.Get(ctx, owner, repo, id)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func loadGithubIssueData(ctx context.Context, token *dagger.Secret, repo string, id int) (*GithubIssueData, error) {
	issue, err := loadGithubIssue(ctx, token, repo, id)
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

	ghClient, err := githubClient(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if issue is pull request
	if issue.IsPullRequest() {
		pr, _, err := ghClient.PullRequests.Get(ctx, issue.Repository.Owner.GetName(), repo, id)
		if err != nil {
			return nil, err
		}
		ghi.HeadRef = pr.Head.GetRef() // FIXME: this wont work on forks
		ghi.BaseRef = pr.Base.GetRef()
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

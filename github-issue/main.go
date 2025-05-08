// A generated module for GithubIssue functions

package main

import (
	"context"
	"dagger/github-issue/internal/dagger"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v59/github"
)

const GITHUB_CLI = "2.72.0"

type GithubIssue struct {
	// Github Token
	// +private
	Token *dagger.Secret
}

// Crate a new GithubIssue object
func New(
	// Github authentication token
	// +optional
	token *dagger.Secret,
) *GithubIssue {
	return &GithubIssue{Token: token}
}

type GithubIssueData struct {
	IssueNumber int
	// Issue title
	Title string
	// Issue body content
	Body string
	// Head ref for a pull request
	HeadRef string
	// Base ref for a pull request
	BaseRef string
}

// List Github issues for a repository
func (m *GithubIssue) List(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Page of issues to read. There are 10 per page
	// +default=1
	page int,
) ([]*GithubIssueData, error) {
	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return nil, err
	}

	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return nil, err
	}

	issues, _, err := ghClient.Issues.ListByRepo(
		ctx,
		owner,
		repoName,
		&github.IssueListByRepoOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 10,
			},
		},
	)

	res := []*GithubIssueData{}
	for _, i := range issues {
		ghd := &GithubIssueData{
			IssueNumber: i.GetNumber(),
			Title:       i.GetTitle(),
			Body:        i.GetBody(),
		}
		res = append(res, ghd)
	}
	return res, nil
}

// List Github issues for a repository and return a readable output for llms
func (m *GithubIssue) ListUnified(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Page of issues to read. There are 10 per page
	// +default=1
	page int,
) (string, error) {
	issues, err := m.List(ctx, repo, page)
	if err != nil {
		return "", err
	}

	res := fmt.Sprintf("# Issues for %s\n\n", repo)

	for _, i := range issues {
		res += fmt.Sprintf("## Issue %d: %s\n%s\n\n", i.IssueNumber, i.Title, i.Body)
	}
	return res, nil
}

// Read a Github issue from a repository
func (m *GithubIssue) Read(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Issue ID
	issueID int,
) (*GithubIssueData, error) {
	return loadGithubIssueData(ctx, m.Token, repo, issueID)
}

// Create a github issue in a repository TODO: NOT YET IMPLEMENTED
func (m *GithubIssue) Write(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo,
	// Issue title
	title,
	// Issue body
	body string,
) (*GithubIssueData, error) {
	return nil, nil
}

// A Github issue comment
type Comment struct {
	Author    string
	Body      string
	CreatedAt string
}

// List all of the comments on a Github issue or pull request
func (m *GithubIssue) ListComments(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Issue or Pull Request number
	issueID int,
) ([]Comment, error) {
	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return nil, err
	}

	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return nil, err
	}

	ghComments, _, err := ghClient.Issues.ListComments(ctx, owner, repoName, issueID, nil)
	if err != nil {
		return nil, err
	}

	comments := []Comment{}

	for _, c := range ghComments {
		user := c.GetUser().GetLogin()
		body := c.GetBody()
		createdAt := c.GetCreatedAt().String()

		comments = append(comments, Comment{Author: user, Body: body, CreatedAt: createdAt})
	}

	return comments, nil
}

// List all of the comments on a Github issue or pull request and return a readable output for llms
func (m *GithubIssue) ListCommentsUnified(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Issue or Pull Request number
	issueID int,
) (string, error) {
	comments, err := m.ListComments(ctx, repo, issueID)
	if err != nil {
		return "", err
	}
	original, err := m.Read(ctx, repo, issueID)
	if err != nil {
		return "", err
	}
	res := fmt.Sprintf("# Comments on issue %d\n\n## Issue Body\n%s\n\n", issueID, original.Body)

	for _, c := range comments {
		res += fmt.Sprintf("## %s at %s says:\n%s\n\n", c.Author, c.CreatedAt, c.Body)
	}
	return res, nil
}

// Write a comment on a Github issue
func (m *GithubIssue) WriteComment(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Issue or Pull Request number
	issueID int,
	// Comment body
	body string,
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

// Write a code suggestion on a Github pull request
func (m *GithubIssue) WritePullRequestCodeComment(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Pull request number
	issueID int,
	// Git commit sha
	commit string,
	// Comment body, e.g. the suggestion
	body string,
	// File to suggest a change on
	path string,
	// Side of the diff to suggest a change on
	side string,
	// Line number to suggest a change on
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

// Gets the pull request number for a commit
func (m *GithubIssue) GetPrForCommit(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// Git commit sha
	commit string,
) (int, error) {
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

// Create a Github Issue
// func (m *GithubIssue) CreateIssue(
// 	ctx context.Context,
// 	// Github repo, e.g https://github.com/owner/repo
// 	repo string,
// 	// title of issue
// 	title string,
// 	// body of issue
// 	body string,
// ) (*GithubIssueData, error) {}

// Create a Github Pull Request
func (m *GithubIssue) CreatePullRequest(
	ctx context.Context,
	// Github repo, e.g https://github.com/owner/repo
	repo string,
	// title of pull request
	title string,
	// body of pull request
	body string,
	// code to commit
	source *dagger.Directory,
	// branch name for pull request
	// +optional
	branch string,
	// base branch for pull request
	// +default="main"
	base string,
) (*GithubIssueData, error) {
	// determine branch name if unset
	if branch == "" {
		branch = strings.ToValidUTF8(branch, "")
		branch = strings.ReplaceAll(branch, "'", "")
		branch = strings.ReplaceAll(branch, "\"", "")
		branch = strings.ReplaceAll(branch, ":", "")
		branch = strings.ToLower(title)
		branch = strings.ReplaceAll(branch, " ", "_")
	}
	// push source as remote branch
	gitContainer := dag.Container().From("alpine/git:v2.47.1")
	_, err := dag.Github(GITHUB_CLI).
		Container(gitContainer).
		WithSecretVariable("GITHUB_TOKEN", m.Token).
		WithExec([]string{"gh", "auth", "setup-git"}).
		WithExec([]string{"git", "config", "--global", "user.name", "Dagger"}).
		WithExec([]string{"git", "config", "--global", "user.email", "noreply@dagger.io"}).
		WithEnvVariable("CACHE_BUSTER", time.Now().String()).
		WithWorkdir("/src").
		WithExec([]string{"gh", "repo", "clone", repo, ".", "--", "--depth=1"}).
		WithExec([]string{"git", "checkout", "-b", branch}).
		WithDirectory("/src", source.WithoutDirectory(".git")).
		WithExec([]string{"git", "add", "."}).
		WithExec([]string{"git", "commit", "-m", branch}).
		WithExec([]string{"git", "push", "origin", branch}).
		Sync(ctx)
	if err != nil {
		return nil, err
	}
	// create pull request with remote branch
	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return nil, err
	}
	ghClient, err := githubClient(ctx, m.Token)
	if err != nil {
		return nil, err
	}
	prData := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(branch),
		Base:                github.String(base),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}
	pr, _, err := ghClient.PullRequests.Create(ctx, owner, repoName, prData)
	if err != nil {
		return nil, err
	}
	// return issue data for new pull request
	return loadGithubIssueData(ctx, m.Token, repo, int(pr.GetID()))
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
	if token == nil {
		return nil, errors.New("github token is required")
	}
	owner, repoName, err := parseOwnerAndRepo(repo)
	if err != nil {
		return nil, err
	}

	ghClient, err := githubClient(ctx, token)
	if err != nil {
		return nil, err
	}

	issue, _, err := ghClient.Issues.Get(ctx, owner, repoName, id)
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

type Template struct {
	Name    string
	Content string
}

// Retruns the github issue templates
func (m *GithubIssue) Templates(ctx context.Context,
	// The name of the github repository in a owner/repo format
	repo string,
) ([]*Template, error) {

	var issueTemplateDir *dagger.Directory
	if td, err := dag.Git("https://github.com/" + repo).Head().Tree().Directory(".github/ISSUE_TEMPLATE").Sync(ctx); err != nil {
		base := strings.Split(repo, "/")[0]
		if td, err := dag.Git("https://github.com/" + base + "/.github").Head().Tree().Directory(".github/ISSUE_TEMPLATE").Sync(ctx); err != nil {
			return nil, errors.New("issue templates could not be found")
		} else {
			issueTemplateDir = td
		}
	} else {
		issueTemplateDir = td
	}

	tplFiles, err := issueTemplateDir.Entries(ctx)

	if err != nil {
		return nil, err
	}

	templates := []*Template{}

	for _, tf := range tplFiles {
		if tc, err := issueTemplateDir.File(tf).Contents(ctx); err != nil {
			continue
		} else {
			templates = append(templates, &Template{
				Name:    tf,
				Content: tc,
			})
		}

	}
	return templates, nil
}

// Retruns the github issue templates in a readable output
func (m *GithubIssue) TemplatesUnified(ctx context.Context,
	// The name of the github repository in a owner/repo format
	repo string,
) (string, error) {
	templates, err := m.Templates(ctx, repo)
	if err != nil {
		return "", err
	}

	res, err := json.Marshal(templates)
	if err != nil {
		return "", err
	}
	return string(res), nil

}

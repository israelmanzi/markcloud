package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type GitHubClient struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubClient(token, repoFullName string) (*GitHubClient, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid github_repo %q: expected owner/repo", repoFullName)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	return &GitHubClient{
		client: github.NewClient(tc),
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

func (g *GitHubClient) ListDir(ctx context.Context, path string) ([]*github.RepositoryContent, error) {
	_, dirContent, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", path, err)
	}
	return dirContent, nil
}

func (g *GitHubClient) GetFile(ctx context.Context, path string) (string, error) {
	fileContent, _, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("getting %s: %w", path, err)
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", err
	}
	return content, nil
}

func (g *GitHubClient) CreateOrUpdateFile(ctx context.Context, path string, content []byte, message string) error {
	existing, _, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, nil)

	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: content,
	}

	if err == nil && existing != nil {
		opts.SHA = existing.SHA
	}

	_, _, err = g.client.Repositories.CreateFile(ctx, g.owner, g.repo, path, opts)
	return err
}

func (g *GitHubClient) DeleteFile(ctx context.Context, path string, message string) error {
	existing, _, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, nil)
	if err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	_, _, err = g.client.Repositories.DeleteFile(ctx, g.owner, g.repo, path, &github.RepositoryContentFileOptions{
		Message: github.String(message),
		SHA:     existing.SHA,
	})
	return err
}

func (g *GitHubClient) GetLatestWorkflowRun(ctx context.Context) (*github.WorkflowRun, error) {
	runs, _, err := g.client.Actions.ListRepositoryWorkflowRuns(ctx, g.owner, g.repo, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 1},
	})
	if err != nil {
		return nil, err
	}
	if len(runs.WorkflowRuns) == 0 {
		return nil, fmt.Errorf("no workflow runs found")
	}
	return runs.WorkflowRuns[0], nil
}

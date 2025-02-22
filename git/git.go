package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v69/github"
	"github.com/pokt-network/poktroll/pkg/polylog"

	"github.com/buildwithgrove/gdi/config"
)

type GitProvider struct {
	logger polylog.Logger
	*github.Client
	owner string
}

func NewGitProvider(logger polylog.Logger, cfg config.GitConfig) (*GitProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid git config: %w", err)
	}

	return &GitProvider{
		logger: logger,
		Client: github.NewClient(nil).WithAuthToken(cfg.PersonalAccessToken),
		owner:  cfg.OwnerName,
	}, nil
}

type PullRequestConfig struct {
	TargetBranch string
	Title        string
	Body         string
	Draft        bool
	Issue        int
}

func (cfg PullRequestConfig) IsValid() error {
	if cfg.TargetBranch == "" {
		return fmt.Errorf("target branch is required")
	}
	if cfg.Title == "" {
		return fmt.Errorf("title is required")
	}
	if cfg.Body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// CreatePullRequest creates a new pull request on GitHub.
func (p *GitProvider) CreatePullRequest(ctx context.Context, cfg PullRequestConfig) (*github.PullRequest, error) {
	if err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid pull request config: %w", err)
	}

	repoName, err := p.getCurrentRepoName()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo name: %w", err)
	}

	currentBranchName, err := p.getCurrentBranchName()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch name: %w", err)
	}

	newPR := &github.NewPullRequest{
		Head:  github.Ptr(currentBranchName),
		Base:  github.Ptr(cfg.TargetBranch),
		Title: github.Ptr(cfg.Title),
		Body:  github.Ptr(cfg.Body),
		Draft: github.Ptr(cfg.Draft),
	}
	if cfg.Issue != 0 {
		newPR.Issue = github.Ptr(cfg.Issue)
	}

	pr, _, err := p.PullRequests.Create(ctx, p.owner, repoName, newPR)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to create pull request")
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	p.logger.Info().Str("url", pr.GetHTMLURL()).Msg("created pull request")
	return pr, nil
}

/*
Generate unified diff command:
1. git -C %s --no-pager diff %s --unified=0 -- .
  - Runs 'git diff' in the repository located at repoRoot against the targetBranch.
  - The '--unified=0' flag shows only the changed lines with zero context.

2. grep -vE '^(diff --git|index |@@)'
  - Filters out metadata lines from the raw diff output: lines starting with 'diff --git', 'index ', or '@@' (hunk headers).

3. sed -E 's/^--- a\//Old File: /; s/^\+\+\+ b\//New File: /'
  - Reformats file header lines: changing '--- a/' to 'Old File: ' and '+++ b/' to 'New File: ', providing clearer context for changes.

4. grep -vE '^$'
  - Removes any empty lines from the output to minimize noise.
*/
const GenDiffCmdTemplate = `git -C %s --no-pager diff %s --unified=0 -- . | grep -vE '^(diff --git|index |@@)' | sed -E 's/^--- a\//Old File: /; s/^\+\+\+ b\//New File: /' | grep -vE '^$'`

// GenerateDiff generates a diff between the current branch and the target branch.
func (p *GitProvider) GenerateDiff(ctx context.Context, targetBranch string) (string, error) {
	// Get the absolute path of the repository root
	repoRoot, err := p.getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Generate the unified diff command
	genDiffCmd := fmt.Sprintf(GenDiffCmdTemplate, repoRoot, targetBranch)

	cmd := exec.Command("bash", "-c", genDiffCmd)
	diffOutput, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute diff command: %v\nOutput: %s", err, string(diffOutput))
	}

	wrappedOutput := fmt.Sprintf("```diff\n%s\n```", string(diffOutput))
	return wrappedOutput, nil
}

// getRepoRoot returns the absolute path of the repository root.
func (p *GitProvider) getRepoRoot() (string, error) {
	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err
	}

	// Get the repository's worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// Get the absolute path of the worktree root
	repoRoot := worktree.Filesystem.Root()

	return repoRoot, nil
}

// getCurrentRepoName returns the name of the Git repository in the current directory.
func (p *GitProvider) getCurrentRepoName() (string, error) {
	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err
	}

	// Get the repository's worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// Get the base name of the worktree directory
	repoName := filepath.Base(worktree.Filesystem.Root())

	return repoName, nil
}

// getCurrentBranchName returns the name of the current branch in the Git repository.
func (p *GitProvider) getCurrentBranchName() (string, error) {
	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err
	}

	// Get the HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	// Extract the branch name from the reference
	branchName := ref.Name().Short()

	return branchName, nil
}

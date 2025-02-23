package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v69/github"
	"github.com/pokt-network/poktroll/pkg/polylog"

	gitCfg "github.com/buildwithgrove/gdi/config/git"
)

const repoOwner = "buildwithgrove"

type Provider struct {
	logger polylog.Logger
	*github.Client
}

func NewGitProvider(logger polylog.Logger, cfg gitCfg.Config) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid git config: %w", err)
	}

	return &Provider{
		logger: logger,
		Client: github.NewClient(nil).WithAuthToken(cfg.PersonalAccessToken),
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
		return fmt.Errorf("pull request config error: target branch is required")
	}
	if cfg.Title == "" {
		return fmt.Errorf("pull request config error: title is required")
	}
	if cfg.Body == "" {
		return fmt.Errorf("pull request config error: body is required")
	}
	return nil
}

// CreatePullRequest creates a new pull request on GitHub.
func (p *Provider) CreatePullRequest(ctx context.Context, cfg PullRequestConfig) (*github.PullRequest, error) {
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
	// Push branch to remove to ensure the branch exists on the remote and is up to date
	err = p.PushBranchToRemote(currentBranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to push branch to remote: %w", err)
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

	pr, _, err := p.PullRequests.Create(ctx, repoOwner, repoName, newPR)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to create pull request")
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	p.logger.Info().Str("url", pr.GetHTMLURL()).Msg("created pull request")
	return pr, nil
}

// PushBranchToRemote pushes the specified branch to the remote repository using the stored PAT.
func (p *Provider) PushBranchToRemote(branchName string) error {
	// Construct the git push command
	cmd := exec.Command("git", "push", "origin", branchName)

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch to remote: %w\nOutput: %s", err, string(output))
	}

	p.logger.Info().Msgf("Branch %s pushed to remote successfully", branchName)
	return nil
}

// The below commands are used to generate a diff between the current branch and the target branch.
const (
	// gitDiffCmdTemplate generates a diff between the current branch and the target branch
	// using the --unified=0 flag to show only the changed lines with zero context.
	// It runs the command in the specified repository root.
	gitDiffCmdTemplate = `git -C %s --no-pager diff %s --unified=0 -- .`
	// grepCmd filters out metadata lines from the diff output.
	// It removes lines starting with 'diff --git', 'index ', and '@@' (hunk headers).
	grepCmd = `grep -vE '^(diff --git|index |@@)'`
	// sedCmd reformats file header lines in the diff output.
	// It changes '--- a/' to 'Old File: ' and '+++ b/' to 'New File: ',
	// providing clearer context for changes.
	sedCmd = `sed -E 's/^--- a\//Old File: /; s/^\+\+\+ b\//New File: /'`
	// finalGrepCmd removes any empty lines from the output to minimize noise.
	finalGrepCmd = `grep -vE '^$'`

	// Public variable that combines all commands.
	// It is unused in this package but sent to the LLM to provide context for how the diff was generated.
	CombinedDiffCmd = gitDiffCmdTemplate + " | " + grepCmd + " | " + sedCmd + " | " + finalGrepCmd
)

// Update GenerateDiff to use the private global variables
func (p *Provider) GenerateDiff(ctx context.Context, targetBranch string) (string, error) {
	// Get the absolute path of the repository root
	repoRoot, err := p.getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Step 1: Run git diff
	gitDiffCmd := fmt.Sprintf(gitDiffCmdTemplate, repoRoot, targetBranch)
	p.logger.Info().Msgf("Executing git diff command: %s", gitDiffCmd)
	gitDiffOutput, err := exec.Command("bash", "-c", gitDiffCmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute git diff command: %v\nOutput: %s", err, string(gitDiffOutput))
	}

	// Step 2: Apply grep filter
	grepCmd := exec.Command("bash", "-c", grepCmd)
	grepCmd.Stdin = bytes.NewReader(gitDiffOutput)
	grepOutput, err := grepCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute grep command: %v\nOutput: %s", err, string(grepOutput))
	}

	// Step 3: Apply sed transformation
	sedCmd := exec.Command("bash", "-c", sedCmd)
	sedCmd.Stdin = bytes.NewReader(grepOutput)
	sedOutput, err := sedCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute sed command: %v\nOutput: %s", err, string(sedOutput))
	}

	// Step 4: Remove empty lines
	finalGrepCmd := exec.Command("bash", "-c", finalGrepCmd)
	finalGrepCmd.Stdin = bytes.NewReader(sedOutput)
	finalOutput, err := finalGrepCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute final grep command: %v\nOutput: %s", err, string(finalOutput))
	}

	wrappedOutput := fmt.Sprintf("```diff\n%s\n```", string(finalOutput))
	return wrappedOutput, nil
}

// getRepoRoot returns the absolute path of the repository root.
func (p *Provider) getRepoRoot() (string, error) {
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
func (p *Provider) getCurrentRepoName() (string, error) {
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
func (p *Provider) getCurrentBranchName() (string, error) {
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

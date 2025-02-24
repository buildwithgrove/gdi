// ---------------------------------------------------------------------------
// File: createpr.go
// Package: git
//
// Purpose:
//   This command automatically generates a GitHub Pull Request (PR) description
//   using a language model (LLM) and then creates a PR on GitHub. It does so by
//   computing a diff from the local git repository, generating a prompt for the LLM,
//   and incorporating the LLM's response into a PR description template. The command
//   also supports a dummy mode where the PR is not created but instead its description
//   is printed and copied to the clipboard.
//
// Features:
//   - Uses LLM to generate a PR description from a git diff.
//   - Supports overriding the default LLM provider and model using flags.
//   - Validates required flags (e.g. a non-empty PR title).
//   - If in dummy mode, does not create the PR and copies the PR description to
//     the clipboard instead.
//   - Logs relevant information with a logger from the polyzero library.
//   - Supports linking an issue number to the PR, and creating the PR as a draft if
//     the title contains "DRAFT".
// ---------------------------------------------------------------------------

package git

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"

	"github.com/buildwithgrove/gdi/config"
	llmCfg "github.com/buildwithgrove/gdi/config/llm"
	"github.com/buildwithgrove/gdi/git"
	"github.com/buildwithgrove/gdi/llm"
)

// Git config flags
var prTitle string
var targetBranch string
var issue int
var dummy bool

// LLM config flags
var llmProviderOverride string
var llmModelOverride string

func init() {
	// Initialize Git-related flags.
	createprCmd.Flags().StringVarP(&prTitle, "pr-title", "t", "", "PR title to open the PR with. Should be descriptive and contain relevant tags. Will open a draft PR if the string contains [DRAFT] or [WIP]. [REQUIRED]")
	createprCmd.Flags().StringVarP(&targetBranch, "target-branch", "b", "main", "Target branch to open the PR on. [OPTIONAL, defaults to 'main']")
	createprCmd.Flags().IntVarP(&issue, "issue", "i", 0, "Issue number to assign to the PR. [OPTIONAL]")
	createprCmd.Flags().BoolVarP(&dummy, "dummy", "d", false, "Dummy mode. If true, the command will not create a PR but will only print the PR description and copy to clipboard [OPTIONAL, defaults to 'false']")
	// Initialize LLM-related flags.
	createprCmd.Flags().StringVarP(&llmProviderOverride, "provider-override", "p", "", "LLM provider override. If set the default provider in the config will be overridden. [OPTIONAL]")
	createprCmd.Flags().StringVarP(&llmModelOverride, "model-override", "m", "", "LLM model override. If set the default model in the config will be overridden. [OPTIONAL]")
}

// createprCmd represents the createpr command
var createprCmd = &cobra.Command{
	Use:   "createpr",
	Short: "Automatically generate a PR description and open it on GitHub.",
	Long: `Automatically generate a PR description and open it on GitHub.

This command automatically generates a PR description by using an LLM to summarize a git diff.
It first computes a diff against a target branch (default "main"), builds a prompt incorporating
the diff and the PR title, and sends that prompt to the LLM. The LLM's response is then embedded
into a PR description template that includes a sanity checklist. Finally, the command creates a 
PR on GitHub using the generated description. 

Flags:
  --pr-title (-t)   : Required PR title.
  --target-branch (-b): Target branch (default "main").
  --issue (-i)      : Optional issue number to assign to the PR.
  --dummy (-d)      : If set, the PR is not created; instead, the description is printed and copied.
  --provider-override (-p): Optional LLM provider override.
  --model-override (-m)   : Optional LLM model override.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Initialize logger.
		logger := polyzero.NewLogger()

		// Validate required flags.
		if prTitle == "" {
			log.Fatalf("PR title is required")
		}

		// Load configuration from the config YAML file.
		config, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		// Initialize the Git provider using loaded Git config.
		gitProvider, err := git.NewGitProvider(logger, *config.Git)
		if err != nil {
			log.Fatalf("failed to create git provider: %v", err)
		}

		// Get additional provider flags based on any overrides set.
		providerFlags := getProviderFlags()
		// Initialize the LLM provider with potential provider overrides.
		llmProvider, err := llmCfg.NewLLMProvider(logger, config.LLMs, providerFlags...)
		if err != nil {
			log.Fatalf("failed to get LLM provider: %v", err)
		}

		logger.Info().
			Str("dummy", fmt.Sprintf("%t", dummy)).Msg("Initialization successful. Running Create PR command.")

		// Generate a diff from Git against the target branch.
		diff, err := gitProvider.GenerateDiff(context.Background(), targetBranch)
		if err != nil {
			log.Fatalf("failed to generate diff: %v", err)
		}

		// Build the prompt by merging PR title and generated diff.
		prompt := buildPrompt(prTitle, diff)

		// Get prompt flags based on any model override.
		promptFlags := getPromptFlags()
		// Send the prompt to the LLM provider.
		response, err := llmProvider.SendPrompt(context.Background(), prompt, promptFlags...)
		if err != nil {
			log.Fatalf("failed to send prompt: %v", err)
		}

		// Build the final PR description by adding a sanity checklist.
		prDescription := buildPRDescription(response)

		// In dummy mode, print the PR description and copy it to the clipboard.
		if dummy {
			err := clipboard.Init()
			if err != nil {
				log.Fatalf("failed to initialize clipboard: %v", err)
			}
			fmt.Printf("PR Description:\n\n%s\n", prDescription)
			clipboard.Write(clipboard.FmtText, []byte(prDescription))
			logger.Info().Msg("PR description copied to clipboard. PR not created.")
			return
		}

		// Construct the pull request configuration.
		pullRequestConfig := git.PullRequestConfig{
			TargetBranch: targetBranch,
			Title:        prTitle,
			Body:         prDescription,
			Draft:        isDraft(prTitle),
		}
		// If an issue number is provided, add it to the configuration.
		if issue != 0 {
			pullRequestConfig.Issue = issue
		}
		// Create the pull request using the Git provider.
		pullRequest, err := gitProvider.CreatePullRequest(context.Background(), pullRequestConfig)
		if err != nil {
			log.Fatalf("failed to create pull request: %v", err)
		}

		// Output the URL of the created PR.
		fmt.Printf("Pull request created: %s\n", *pullRequest.HTMLURL)
	},
}

/*--------- LLM Provider and Prompt Flags ---------*/

// getProviderFlags returns the LLM provider flags, modifying the LLM configuration.
func getProviderFlags() []llmCfg.ProviderFlag {
	var flags []llmCfg.ProviderFlag

	// Append provider override flag if specified.
	if llmProviderOverride != "" {
		flags = append(flags, llmCfg.WithProviderOverride(llmCfg.ProviderType(llmProviderOverride)))
	}

	return flags
}

// getPromptFlags returns the LLM prompt flags, modifying the LLM prompt configuration.
func getPromptFlags() []llm.PromptFlag {
	var flags []llm.PromptFlag

	// Append model override flag if provided.
	if llmModelOverride != "" {
		flags = append(flags, llm.WithLLMModelOverride(llmModelOverride))
	}

	return flags
}

/*--------- Prompt Construction ---------*/

// buildPrompt constructs the LLM prompt by merging the PR title and git diff.
// The prompt follows a specific template to guide the LLM in generating a PR description.
func buildPrompt(prTitle, diff string) string {
	return fmt.Sprintf(promptIntro, git.CombinedDiffCmd, prTitle, diff)
}

// isDraft checks if the PR title contains "DRAFT" (case-insensitive).
// If it does, the PR will be created as a draft.
func isDraft(prTitle string) bool {
	return strings.Contains(strings.ToUpper(prTitle), "DRAFT") ||
		strings.Contains(strings.ToUpper(prTitle), "WIP")
}

// buildPRDescription builds the final PR description from the LLM response.
// It adds a sanity checklist to the generated description.
func buildPRDescription(summary string) string {
	return fmt.Sprintf(prDescription, summary)
}

/*--------- LLM Prompt Templates ---------*/

// promptIntro provides the instructions and template structure for the LLM.
// It guides the LLM on how to generate the PR description.
const promptIntro = `Please generate a GitHub PR description from the following diff output. The diff was generated using this command:

		%s

		Where the first 's' is the repo root and the second 's' is the target branch.

		The generated description should be in the following template format.
		It should output ONLY the template inside the --- BEGIN PR TEMPLATE --- and --- END PR TEMPLATE --- tags.
		Use as many Primary and Secondary changes as you feel are valid to capture the changes in the diff.

		--- BEGIN PR TEMPLATE (do not include this line in the output) ---

		## 🌿 Summary

		< One line summary >

		### 🌱 Primary Changes:
		- < core changes # 1 >
		- < core changes # 2 >
		- ...

		### 🍃 Secondary changes:
		- < secondary changes # 1 >
		- < secondary changes # 2 >

		--- END PR TEMPLATE (do not include this line in the output) ---

		Focus:
		The PR title should be used to focus the summary and primary/secondary changes as it describes the "why" of the PR.
		Do not simply use it as the Summary but use it as a simple guideline for how to structure the the template.

		PR Title: %s

		Considerations:
		- Keep the bullet points concise
		- Escape key terms with backticks
		- Primary changes are what the PR is all about
		- Secondary changes include misc changes (e.g. documentation updates, etc)
		- Limit the number of bullets to 3-5

		Diff:

		%s`

// prDescription defines the final PR description template, including a sanity checklist.
const prDescription = `%s

## 🛠️ Type of change

Select one or more from the following:

- [ ] New feature, functionality or library
- [ ] Bug fix
- [ ] Code health or cleanup
- [ ] Documentation
- [ ] Other (specify)

## 🤯 Sanity Checklist

- [ ] I have updated the GitHub Issue 'assignees', 'reviewers', 'labels', 'project', 'iteration' and 'milestone'
- [ ] For docs, I have run 'make docusaurus_start'
- [ ] For code, I have run 'make test_all'
- [ ] For configurations, I have update the documentation
- [ ] I added TODOs where applicable
`

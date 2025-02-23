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
	// Git config flags
	createprCmd.Flags().StringVarP(&prTitle, "pr-title", "t", "", "PR title to open the PR with. Should be descriptive and contain relevant tags. [REQUIRED]")
	createprCmd.Flags().StringVarP(&targetBranch, "target-branch", "b", "main", "Target branch to open the PR on. [OPTIONAL, defaults to 'main']")
	createprCmd.Flags().IntVarP(&issue, "issue", "i", 0, "Issue number to assign to the PR. [OPTIONAL]")
	createprCmd.Flags().BoolVarP(&dummy, "dummy", "d", false, "Dummy mode. If true, the command will not create a PR but will only print the PR description and copy to clipboard [OPTIONAL, defaults to 'false']")
	// LLM config flags
	createprCmd.Flags().StringVarP(&llmProviderOverride, "provider-override", "p", "", "LLM provider override. If set the default provider in the config will be overridden. [OPTIONAL]")
	createprCmd.Flags().StringVarP(&llmModelOverride, "model-override", "m", "", "LLM model override. If set the default model in the config will be overridden. [OPTIONAL]")

}

// createprCmd represents the createpr command
var createprCmd = &cobra.Command{
	Use:   "createpr",
	Short: "Automatically generate a PR description and open it on GitHub.",
	Long: `Automatically generate a PR description and open it on GitHub.

This command will automatically generate a PR description and open it on GitHub.
It will use the LLM to generate a PR description based on your git diff and then 
open a PR on GitHub to the main branch or a specified target branch.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logger.
		logger := polyzero.NewLogger()

		// Validate required flags.
		if prTitle == "" {
			log.Fatalf("PR title is required")
		}

		// Load config from config YAML file.
		config, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		// Initialize Git provider.
		gitProvider, err := git.NewGitProvider(logger, *config.Git)
		if err != nil {
			log.Fatalf("failed to create git provider: %v", err)
		}

		// Initialize LLM provider including the provider override flag.
		providerFlags := getProviderFlags()
		llmProvider, err := llmCfg.NewLLMProvider(logger, config.LLMs, providerFlags...)
		if err != nil {
			log.Fatalf("failed to get LLM provider: %v", err)
		}

		logger.Info().
			Str("dummy", fmt.Sprintf("%t", dummy)).Msg("Initialization successful. Running Create PR command.")

		// Generate diff.
		diff, err := gitProvider.GenerateDiff(context.Background(), targetBranch)
		if err != nil {
			log.Fatalf("failed to generate diff: %v", err)
		}

		// Add the diff to the prompt.
		prompt := buildPrompt(prTitle, diff)

		// Send prompt to LLM provider, including the model override flag.
		promptFlags := getPromptFlags()
		response, err := llmProvider.SendPrompt(context.Background(), prompt, promptFlags...)
		if err != nil {
			log.Fatalf("failed to send prompt: %v", err)
		}

		// Add the sanity checklist to the PR description.
		prDescription := buildPRDescription(response)

		// If dummy is true, print the PR description to the console
		// and copy to clipboard instead of creating a PR.
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

		// If dummy is false (which is the default), create the PR on Github.
		pullRequestConfig := git.PullRequestConfig{
			TargetBranch: targetBranch,
			Title:        prTitle,
			Body:         prDescription,
			Draft:        isDraft(prTitle),
		}
		if issue != 0 {
			pullRequestConfig.Issue = issue
		}
		pullRequest, err := gitProvider.CreatePullRequest(context.Background(), pullRequestConfig)
		if err != nil {
			log.Fatalf("failed to create pull request: %v", err)
		}

		// Log the PR URL.
		fmt.Printf("Pull request created: %s\n", *pullRequest.HTMLURL)
	},
}

/*--------- LLM Provider and Prompt Flags ---------*/

// GetProviderFlags returns the LLM provider flags, which will modify the LLM config.
func getProviderFlags() []llmCfg.ProviderFlag {
	var flags []llmCfg.ProviderFlag

	if llmProviderOverride != "" {
		flags = append(flags, llmCfg.WithProviderOverride(llmCfg.ProviderType(llmProviderOverride)))
	}

	return flags
}

// GetPromptFlags returns the LLM prompt flags, which will modify the LLM prompt config.
func getPromptFlags() []llm.PromptFlag {
	var flags []llm.PromptFlag

	if llmModelOverride != "" {
		flags = append(flags, llm.WithLLMModelOverride(llmModelOverride))
	}

	return flags
}

/*--------- Prompt Construction ---------*/

// BuildPrompt builds the prompt for the LLM.
// It appends the diff to the prompt and returns the prompt as a string.
func buildPrompt(prTitle, diff string) string {
	return fmt.Sprintf(promptIntro, git.CombinedDiffCmd, prTitle, diff)
}

// IsDraft checks if the PR title contains the word "DRAFT".
// If so the PR will be created as a draft.
func isDraft(prTitle string) bool {
	return strings.Contains(strings.ToUpper(prTitle), "DRAFT")
}

// BuildPRDescription builds the PR description from the LLM response.
// It adds a sanity checklist to the PR description and returns the PR description as a string.
func buildPRDescription(summary string) string {
	return fmt.Sprintf(prDescription, summary)
}

// Prompt intro provides the LLM with the instructions for the PR description.
// It includes the template for the PR description and the instructions for the LLM.
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

// prDescription is the template for the PR description after the prompt has been sent.
// It adds a sanity checklist to the PR description.
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

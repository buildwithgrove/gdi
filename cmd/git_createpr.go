package cmd

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
	gitPkg "github.com/buildwithgrove/gdi/git"
	llmPkg "github.com/buildwithgrove/gdi/llm"
)

var prTitle string
var targetBranch string
var issue int
var dummy bool

var llmProviderOverride string
var llmModelOverride string

func init() {
	// Git config flags
	createprCmd.Flags().StringVarP(&prTitle, "pr-title", "t", "", "PR title [REQUIRED]")
	createprCmd.Flags().StringVarP(&targetBranch, "target-branch", "b", "main", "Target branch [OPTIONAL, defaults to 'main']")
	createprCmd.Flags().IntVarP(&issue, "issue", "i", 0, "Issue number [OPTIONAL]")
	createprCmd.Flags().BoolVarP(&dummy, "dummy", "d", false, "Dummy mode. If true, the command will not create a PR but will only print the PR description and copy to clipboard [OPTIONAL, defaults to 'false']")
	// LLM config flags
	createprCmd.Flags().StringVarP(&llmProviderOverride, "provider-override", "p", "", "LLM provider override [OPTIONAL]")
	createprCmd.Flags().StringVarP(&llmModelOverride, "model-override", "m", "", "LLM model override [OPTIONAL]")

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
		gitProvider, err := gitPkg.NewGitProvider(logger, *config.Git)
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
		diff, err := gitProvider.GenerateDiff(context.Background(), "llm-provider-and-config")
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

		if dummy {
			err := clipboard.Init()
			if err != nil {
				log.Fatalf("failed to initialize clipboard: %v", err)
			}
			fmt.Printf("PR Description:\n\n%s", prDescription)
			clipboard.Write(clipboard.FmtText, []byte(prDescription))
			return
		}

		pullRequestConfig := gitPkg.PullRequestConfig{
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

		fmt.Printf("Pull request created: %s\n", *pullRequest.HTMLURL)
	},
}

func getProviderFlags() []llmCfg.ProviderFlag {
	var flags []llmCfg.ProviderFlag

	if llmProviderOverride != "" {
		flags = append(flags, llmCfg.WithLLMProviderOverride(llmCfg.LLMProviderType(llmProviderOverride)))
	}

	return flags
}

func getPromptFlags() []llmPkg.PromptFlag {
	var flags []llmPkg.PromptFlag

	if llmModelOverride != "" {
		flags = append(flags, llmPkg.WithLLMModelOverride(llmModelOverride))
	}

	return flags
}

/*--------- Prompt Construction ---------*/

func buildPrompt(prTitle, diff string) string {
	return fmt.Sprintf(promptIntro, gitPkg.GenDiffCmdTemplate, prTitle, diff)
}

func isDraft(prTitle string) bool {
	return strings.Contains(strings.ToUpper(prTitle), "DRAFT")
}

func buildPRDescription(summary string) string {
	return fmt.Sprintf(prDescription, summary)
}

// Prompt intro provides the LLM with the instructions for the PR description.
// It includes the template for the PR description and the instructions for the LLM.
const promptIntro = `Please generate a GitHub PR description from the following diff output. The diff was generated using this command:

		%s

		Where the first %\s is the repo root and the second %\s is the target branch.

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

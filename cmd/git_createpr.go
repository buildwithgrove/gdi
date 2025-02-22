package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/spf13/cobra"

	"github.com/buildwithgrove/gdi/cmd/llm"
	"github.com/buildwithgrove/gdi/config"
	llmPkg "github.com/buildwithgrove/gdi/llm"
)

var llmProviderOverride string
var llmModelOverride string

func init() {
	createprCmd.Flags().StringVarP(&llmProviderOverride, "provider-override", "p", "", "LLM provider override")
	createprCmd.Flags().StringVarP(&llmModelOverride, "model-override", "m", "", "LLM model override")
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
		logger := polyzero.NewLogger()

		// Load config from config YAML file.
		config, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		// Get LLM Provider including the provider override flag.
		providerFlags := getProviderFlags()
		llmProvider, err := llm.GetLLMProvider(logger, config.LLMs, providerFlags...)
		if err != nil {
			log.Fatalf("failed to get LLM provider: %v", err)
		}

		// TODO_NEXT(@commoddity): Add Git package for git client. https://github.com/google/go-github
		// TODO_NEXT(@commoddity): Complete PR creation code.

		// Send prompt to LLM provider, including the model override flag.
		promptFlags := getPromptFlags()
		response, err := llmProvider.SendPrompt(context.Background(), "Who is Aesop Rock?", promptFlags...)
		if err != nil {
			log.Fatalf("failed to send prompt: %v", err)
		}

		fmt.Println(response)
	},
}

func getProviderFlags() []llm.ProviderFlag {
	var flags []llm.ProviderFlag

	if llmProviderOverride != "" {
		flags = append(flags, llm.WithLLMProviderOverride(config.LLMProviderType(llmProviderOverride)))
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

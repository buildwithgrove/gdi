// ---------------------------------------------------------------------------
// File: setup.go
// Package: config
//
// Purpose:
//
//	This command implements the first-time setup configuration for the
//	Grove Developer Interface (GDI). It runs when no configuration file exists
//	and guides the user through initial setup, including:
//	  - Git configuration (optional)
//	  - LLM provider selection and configuration
//	  - API key setup for selected providers
//
// Features:
//   - Interactive prompts with colorized output and emojis
//   - Hidden input for sensitive values (API keys, PATs)
//   - Enum-based selection for LLM providers and models
//   - Validation of required fields
//   - Clear terminal between prompts for better readability
//   - Automatic creation of configuration file after setup
//
// Usage:
//
//	This command runs automatically when no configuration file is found.
//	It can be triggered manually by deleting ~/.config.gdi.yaml and running GDI.
//
// ---------------------------------------------------------------------------
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	cfgPkg "github.com/buildwithgrove/gdi/config"
	cfgEditor "github.com/buildwithgrove/gdi/config/editor"
	log "github.com/buildwithgrove/gdi/log"
	"gopkg.in/yaml.v3"
)

// ConfigExists checks if the configuration file exists.
// It is used to determine if we should run the first-time setup.
func ConfigExists() bool {
	_, err := os.Stat(cfgPkg.ConfigFilePath)
	return err == nil
}

// RunFirstTimeSetup performs an interactive configuration when the config file does not exist.
func RunFirstTimeSetup() error {
	if err := cfgPkg.InitEmptyConfig(); err != nil {
		fmt.Printf("Failed to initialize empty config: %v", err)
		os.Exit(1)
	}

	schema, err := cfgPkg.LoadSchema()
	if err != nil {
		fmt.Printf("Failed to load schema: %v", err)
		os.Exit(1)
	}

	yamlEditor, err := cfgEditor.NewYAMLEditor(
		"gdi",
		cfgPkg.ConfigFilePath,
		schema,
	)
	if err != nil {
		fmt.Printf("Failed to create editor: %v", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	cfgEditor.ClearTerminal()
	fmt.Println(log.Green + "🌿 Welcome to the Grove Developer Interface (GDI)! It looks like this is the first time you're using it." + log.ResetColor)

	fmt.Print(log.Blue + "❓ Would you like to configure it now? (y/n): " + log.ResetColor)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	cfgEditor.ClearTerminal()
	if answer != "y" {
		fmt.Println(log.Yellow + "👋 Exiting GDI configuration. You can configure later using 'gdi config'." + log.ResetColor)
		os.Exit(0)
	}

	// Prepare configuration data
	configNode := yamlEditor.GetConfigNode()

	// Configure git_config (mandatory)
	fmt.Print(log.Green + "🔑 Please enter your GitHub Personal Access Token (PAT) for private repository access. It must have at least `write:repo` scope." + log.ResetColor + "\n")
	token := cfgEditor.ReadHiddenInput(log.Blue + "📝 Enter your valid GitHub Personal Access Token (input hidden): " + log.ResetColor)
	cfgEditor.ClearTerminal()
	gitConfigNode := cfgEditor.GetOrCreateMappingNode(configNode, "git_config")
	gitConfigNode.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "personal_access_token"},
		{Kind: yaml.ScalarNode, Value: token},
	}

	llmConfigNode := cfgEditor.GetOrCreateMappingNode(configNode, "llm_config")
	allowedProviders := yamlEditor.GetEnumOptionsForPath("llm_config.default_llm_provider")
	if len(allowedProviders) == 0 {
		fmt.Printf(log.Red + "No allowed LLM providers found in schema." + log.ResetColor)
		os.Exit(1)
	}

	fmt.Println(log.Green + "🛠️ Configure LLM: You must set a default LLM provider.\n" + log.ResetColor + "(You may override this provider per-request with the `-p|--provider` flag.)\n" + log.Blue + "Choose one of the following options:" + log.ResetColor)
	for i, p := range allowedProviders {
		fmt.Printf("%d. "+log.Purple+"%s"+log.ResetColor+"\n", i+1, p)
	}
	var defaultProvider string
	for {
		fmt.Print(log.Blue + "📝 Enter choice (number): " + log.ResetColor)
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)
		cfgEditor.ClearTerminal()
		choice, err := strconv.Atoi(choiceStr)
		if err != nil || choice < 1 || choice > len(allowedProviders) {
			fmt.Println(log.Red + "Invalid choice. Please try again." + log.ResetColor)
			continue
		}
		defaultProvider = allowedProviders[choice-1]
		break
	}
	llmConfigNode.Content = cfgEditor.SetScalarValue(llmConfigNode.Content, "default_llm_provider", defaultProvider)

	// Configure default provider details
	llmProvidersNode := cfgEditor.GetOrCreateMappingNode(llmConfigNode, "llm_providers")
	fmt.Printf(log.Green+"🛠️ Configuring provider '%s'.\n"+log.ResetColor, defaultProvider)
	providerDetails := promptForProviderConfiguration(yamlEditor, reader, defaultProvider)
	llmProvidersNode.Content = cfgEditor.SetMappingValue(llmProvidersNode.Content, defaultProvider, providerDetails)

	// Ask about additional providers
	fmt.Print(log.Green + "❓ Would you like to configure any other LLM providers? (y/n): " + log.ResetColor)
	answer, _ = reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	cfgEditor.ClearTerminal()
	if answer == "y" {
		for {
			// List remaining providers
			remaining := []string{}
			for _, p := range allowedProviders {
				found := false
				for i := 0; i < len(llmProvidersNode.Content); i += 2 {
					if llmProvidersNode.Content[i].Value == p {
						found = true
						break
					}
				}
				if !found {
					remaining = append(remaining, p)
				}
			}
			if len(remaining) == 0 {
				fmt.Println(log.Yellow + "✅ All LLM providers have been configured. You can edit the configuration later using 'gdi config'." + log.ResetColor)
				break
			}
			fmt.Println(log.Green + "🛠️ Select an LLM provider to configure from the following:" + log.ResetColor)
			for i, p := range remaining {
				fmt.Printf("%d. "+log.Purple+"%s"+log.ResetColor+"\n", i+1, p)
			}
			fmt.Print(log.Blue + "📝 Enter choice (number): " + log.ResetColor)
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)
			cfgEditor.ClearTerminal()
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(remaining) {
				fmt.Println(log.Red + "Invalid choice. Please try again." + log.ResetColor)
				continue
			}
			selectedProvider := remaining[choice-1]
			fmt.Printf(log.Green+"🛠️ Configuring LLM provider '%s'.\n"+log.ResetColor, selectedProvider)
			details := promptForProviderConfiguration(yamlEditor, reader, selectedProvider)
			llmProvidersNode.Content = cfgEditor.SetMappingValue(llmProvidersNode.Content, selectedProvider, details)

			fmt.Print(log.Green + "❓ Would you like to configure another LLM provider? (y/n): " + log.ResetColor)
			cont, _ := reader.ReadString('\n')
			cont = strings.TrimSpace(strings.ToLower(cont))
			cfgEditor.ClearTerminal()
			if cont != "y" {
				break
			}
		}
	}

	fmt.Println(log.Green + "🌿 Configuration completed and saved. You may edit the configuration later by running 'gdi config'." + log.ResetColor)

	yamlEditor.SaveConfigAndExit()

	return nil
}

// promptForProviderConfiguration prompts the user for provider details (api_key and client_model) and returns them as a *yaml.Node.
func promptForProviderConfiguration(yamlEditor *cfgEditor.YAMLEditor, reader *bufio.Reader, provider string) *yaml.Node {
	providerNode := &yaml.Node{Kind: yaml.MappingNode}

	apiKey := cfgEditor.ReadHiddenInput(log.Blue + "🔑 Enter API key for " + provider + " (input hidden): " + log.ResetColor)
	cfgEditor.ClearTerminal()
	providerNode.Content = append(providerNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "api_key"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: apiKey},
	)

	allowedModels := yamlEditor.GetEnumOptionsForPath("llm_config.llm_providers." + provider + ".client_model")
	if len(allowedModels) > 0 {
		fmt.Printf(log.Green+"🤖 Select a default client model for %s:\n"+log.ResetColor+"(You may override this model per-request with the `-m|--model` flag.)\n", provider)
		for i, model := range allowedModels {
			fmt.Printf("%d. "+log.Purple+"%s"+log.ResetColor+"\n", i+1, model)
		}
		var clientModel string
		for {
			fmt.Print(log.Blue + "📝 Enter choice (number): " + log.ResetColor)
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)
			cfgEditor.ClearTerminal()
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(allowedModels) {
				fmt.Println(log.Red + "Invalid choice. Please try again." + log.ResetColor)
				continue
			}
			clientModel = allowedModels[choice-1]
			break
		}
		providerNode.Content = append(providerNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "client_model"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: clientModel},
		)
	} else {
		fmt.Printf(log.Green+"🤖 Enter client model for %s: "+log.ResetColor, provider)
		clientModel, _ := reader.ReadString('\n')
		clientModel = strings.TrimSpace(clientModel)
		cfgEditor.ClearTerminal()
		providerNode.Content = append(providerNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "client_model"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: clientModel},
		)
	}

	return providerNode
}

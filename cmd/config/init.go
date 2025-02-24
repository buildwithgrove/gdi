package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	configPkg "github.com/buildwithgrove/gdi/config"
)

// ConfigExists checks if the configuration file exists.
// It is used to determine if we should run the first-time setup.
func ConfigExists() bool {
	_, err := os.Stat(configPkg.ConfigFilePath)
	return err == nil
}

// RunFirstTimeSetup performs an interactive configuration when the config file does not exist.
func RunFirstTimeSetup() error {
	reader := bufio.NewReader(os.Stdin)
	clearTerminal()
	fmt.Println(ColorGreen + "🌿 Welcome to the Grove Developer Interface (GDI)! It looks like this is the first time you're using it." + ColorReset)

	fmt.Print(ColorBlue + "❓ Would you like to configure it now? (y/n): " + ColorReset)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	clearTerminal()
	if answer != "y" {
		fmt.Println(ColorYellow + "👋 Exiting GDI configuration. You can configure later using 'gdi config'." + ColorReset)
		os.Exit(0)
	}

	// Prepare configuration data
	configData := make(map[string]interface{})

	// Optionally configure git_config
	fmt.Print(ColorGreen + "❓ Would you like to configure git configuration?" + ColorReset + "\nThis is only necessary for private repositories. It must be a valid PAT with at least `write:repo` scope." + ColorBlue + "\nYou may edit this choice later using 'gdi config'." + ColorReset + "\n(y/n): ")
	answer, _ = reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	clearTerminal()
	if answer == "y" {
		token := readHiddenInput(ColorBlue + "📝 Enter your valid GitHub Personal Access Token (input hidden): " + ColorReset)
		clearTerminal()
		configData["git_config"] = map[string]interface{}{
			"personal_access_token": token,
		}
	}

	// Load embedded schema. schemaMap var is already defined in root.go
	configPkg.LoadSchema(&schemaMap)

	llmConfig := make(map[string]interface{})
	allowedProviders := getEnumOptionsForPath("llm_config.default_llm_provider")
	if len(allowedProviders) == 0 {
		log.Fatalf(ColorRed + "No allowed LLM providers found in schema." + ColorReset)
	}

	fmt.Println(ColorGreen + "🛠️ Configure LLM: You must set a default LLM provider.\n" + ColorReset + "(You may override this provider per-request with the `-p|--provider` flag.)\n" + ColorBlue + "Choose one of the following options:" + ColorReset)
	for i, p := range allowedProviders {
		fmt.Printf("%d. "+ColorPurple+"%s"+ColorReset+"\n", i+1, p)
	}
	var defaultProvider string
	for {
		fmt.Print(ColorBlue + "📝 Enter choice (number): " + ColorReset)
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)
		clearTerminal()
		choice, err := strconv.Atoi(choiceStr)
		if err != nil || choice < 1 || choice > len(allowedProviders) {
			fmt.Println(ColorRed + "Invalid choice. Please try again." + ColorReset)
			continue
		}
		defaultProvider = allowedProviders[choice-1]
		break
	}
	llmConfig["default_llm_provider"] = defaultProvider

	// Configure default provider details
	llmProviders := make(map[string]interface{})
	fmt.Printf(ColorGreen+"🛠️ Configuring provider '%s'.\n"+ColorReset, defaultProvider)
	providerDetails := promptForProviderConfiguration(reader, defaultProvider)
	llmProviders[defaultProvider] = providerDetails

	// Ask about additional providers
	fmt.Print(ColorGreen + "❓ Would you like to configure any other LLM providers? (y/n): " + ColorReset)
	answer, _ = reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	clearTerminal()
	if answer == "y" {
		for {
			// List remaining providers
			remaining := []string{}
			for _, p := range allowedProviders {
				if _, exists := llmProviders[p]; !exists {
					remaining = append(remaining, p)
				}
			}
			if len(remaining) == 0 {
				fmt.Println(ColorYellow + "✅ All LLM providers have been configured. You can edit the configuration later using 'gdi config'." + ColorReset)
				break
			}
			fmt.Println(ColorGreen + "🛠️ Select an LLM provider to configure from the following:" + ColorReset)
			for i, p := range remaining {
				fmt.Printf("%d. "+ColorPurple+"%s"+ColorReset+"\n", i+1, p)
			}
			fmt.Print(ColorBlue + "📝 Enter choice (number): " + ColorReset)
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)
			clearTerminal()
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(remaining) {
				fmt.Println(ColorRed + "Invalid choice. Please try again." + ColorReset)
				continue
			}
			selectedProvider := remaining[choice-1]
			fmt.Printf(ColorGreen+"🛠️ Configuring LLM provider '%s'.\n"+ColorReset, selectedProvider)
			details := promptForProviderConfiguration(reader, selectedProvider)
			llmProviders[selectedProvider] = details

			fmt.Print(ColorGreen + "❓ Would you like to configure another LLM provider? (y/n): " + ColorReset)
			cont, _ := reader.ReadString('\n')
			cont = strings.TrimSpace(strings.ToLower(cont))
			clearTerminal()
			if cont != "y" {
				break
			}
		}
	}
	llmConfig["llm_providers"] = llmProviders
	configData["llm_config"] = llmConfig

	// Save configuration to YAML file
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to marshal configuration to YAML: %v"+ColorReset, err)
	}
	err = os.WriteFile(configPkg.ConfigFilePath, yamlData, 0644)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to write configuration file: %v"+ColorReset, err)
	}

	fmt.Println(ColorGreen + "🌿 Configuration completed and saved. You may edit the configuration later by running 'gdi config'." + ColorReset)

	return nil
}

// promptForProviderConfiguration prompts the user for provider details (api_key and client_model) and returns them as a map.
func promptForProviderConfiguration(reader *bufio.Reader, provider string) map[string]interface{} {
	details := make(map[string]interface{})
	apiKey := readHiddenInput(ColorBlue + "🔑 Enter API key for " + provider + " (input hidden): " + ColorReset)
	clearTerminal()
	details["api_key"] = apiKey

	allowedModels := getEnumOptionsForPath("llm_config.llm_providers." + provider + ".client_model")
	if len(allowedModels) > 0 {
		fmt.Printf(ColorGreen+"🤖 Select a default client model for %s:\n"+ColorReset+"(You may override this model per-request with the `-m|--model` flag.)\n", provider)
		for i, model := range allowedModels {
			fmt.Printf("%d. "+ColorPurple+"%s"+ColorReset+"\n", i+1, model)
		}
		var clientModel string
		for {
			fmt.Print(ColorBlue + "📝 Enter choice (number): " + ColorReset)
			choiceStr, _ := reader.ReadString('\n')
			choiceStr = strings.TrimSpace(choiceStr)
			clearTerminal()
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(allowedModels) {
				fmt.Println(ColorRed + "Invalid choice. Please try again." + ColorReset)
				continue
			}
			clientModel = allowedModels[choice-1]
			break
		}
		details["client_model"] = clientModel
	} else {
		fmt.Printf(ColorGreen+"🤖 Enter client model for %s: "+ColorReset, provider)
		clientModel, _ := reader.ReadString('\n')
		clientModel = strings.TrimSpace(clientModel)
		clearTerminal()
		details["client_model"] = clientModel
	}
	return details
}

// Add helper function to read hidden input using golang.org/x/term
func readHiddenInput(prompt string) string {
	fmt.Print(prompt)
	byteInput, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf(ColorRed+"Failed to read hidden input: %v"+ColorReset, err)
	}
	fmt.Println("")
	return strings.TrimSpace(string(byteInput))
}

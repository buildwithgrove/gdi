// ---------------------------------------------------------------------------
// File: root.go
// Package: config
//
// Purpose:
//
//	This command implements an interactive configuration editor for the
//	Grove Developer Interface (GDI). It allows the user to traverse and edit
//	the YAML configuration file (~/.config.gdi.yaml) interactively, based on the
//	schema defined in ./config/config.schema.yaml. The command supports editing
//	of nested fields, enum selection with allowed values (displayed in purple),
//	and provider-specific validation (e.g., ensuring that a default LLM provider
//	is properly configured before it can be selected).
//
// Features:
//   - Interactive traversal of config fields with options to "go up" a level.
//   - Dynamic prompts that display the field's schema description.
//   - Enum-based selections with allowed values.
//   - Provider validation: if a default LLM provider is selected which
//     lacks configuration (api_key or client_model), the user is prompted to fill
//     in the necessary details. The client_model field uses enum options.
//   - Colorized output and emojis for improved readability and guidance.
//   - The ability to save and exit from any prompt by typing 's' (save option) in yellow.
//   - Clear text prompts for errors, field names, and schema descriptions.
//
// Usage:
//
//	Running the "gdi config" command will launch the interactive configuration editor.
//	It supports flags:
//	   --show (-s): Show the current configuration.
//	   --editor (-e): Open the configuration in a text editor instead of interactive mode.
//
// ---------------------------------------------------------------------------
package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/config"
)

// ANSI color definitions used for colored output.
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m" // For prompts/questions and success messages.
	ColorBlue   = "\033[34m" // For YAML field names and "go up" text.
	ColorPurple = "\033[35m" // For enum option values.
	ColorWhite  = "\033[37m" // For schema descriptions and "generic" white text.
	ColorYellow = "\033[33m" // For save option (always printed as 's').
	ColorRed    = "\033[31m" // For error messages.
	ColorCyan   = "\033[36m" // Used for the full "Enter choice" prompt.
)

var (
	show      bool
	editor    string
	schemaMap map[string]interface{}
)

// init sets up flags for the config command.
func init() {
	ConfigCmd.Flags().BoolVarP(&show, "show", "s", false, "Show the configuration.")
	ConfigCmd.Flags().StringVarP(&editor, "editor", "e", "", "Edit the configuration in the given text editor.")
}

// ConfigCmd represents the interactive configuration command.
// The Long description provides detailed usage information.
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit the configuration for the application.",
	Long: `Edit the configuration for the application.

This command is used to modify the YAML configuration file for the Grove Developer Interface.
It uses an interactive command-line interface to traverse and update configuration fields,
using the schema defined in ./config/config.schema.yaml. You can navigate through nested fields,
edit values (with enum validation where applicable), and ensure that required fields for providers
(such as LLM configurations) are appropriately set. You may also choose to save and exit at any
time by entering the save command.
	  
Flags:
  --show (-s)   : Show the current config file.
  --editor (-e) : Open the config file in a specified text editor. For example, 'gdi config --editor nano' will open the config file in nano.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle the --show flag: print the configuration file.
		if show {
			showConfig()
			return
		}
		// Handle the --editor flag: open the file in the given text editor.
		if editor != "" {
			editConfig(editor)
			return
		}
		// Otherwise, start the interactive configuration editor.
		interactiveEditConfigV3()
	},
}

// showConfig prints the current configuration file to stdout.
func showConfig() {
	data, err := os.ReadFile(config.ConfigFilePath)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to read config file: %v"+ColorReset, err)
	}
	fmt.Println(string(data))
}

// editConfig opens the configuration file in the user's preferred text editor.
func editConfig(editor string) {
	cmd := exec.Command(editor, config.ConfigFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// interactiveEditConfigV3 launches the interactive editor.
// It loads the user's configuration, the schema, and continuously prompts for edits.
func interactiveEditConfigV3() {
	data, err := os.ReadFile(config.ConfigFilePath)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to read config file: %v"+ColorReset, err)
	}

	var configMap map[string]interface{}
	err = yaml.Unmarshal(data, &configMap)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to unmarshal YAML: %v"+ColorReset, err)
	}

	// Load the configuration schema from the embedded schema.
	config.LoadSchema(&schemaMap)

	// Interactive editing loop.
	for {
		clearTerminal()
		editFieldRecursive(configMap, "", configMap)

		// Write out current configuration changes.
		saveConfigV3(configMap)
		fmt.Print(ColorGreen + "Do you want to continue editing? (" + ColorWhite + "y/n" + ColorGreen + "/" + ColorYellow + "s" + ColorGreen + " for save and exit): " + ColorReset)
		var cont string
		fmt.Scan(&cont)
		cont = strings.ToLower(cont)
		if cont == "s" || cont == "n" {
			log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
			os.Exit(0)
		} else if cont == "y" {
			continue
		} else {
			log.Println(ColorRed + "Invalid input, please try again." + ColorReset)
		}
	}
}

// clearTerminal clears the console for a fresh prompt display.
func clearTerminal() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// getEnumOptionsForPath traverses the schema using a dot-delimited field path
// and returns the allowed enum options (if any) for that field.
func getEnumOptionsForPath(fieldPath string) []string {
	parts := strings.Split(fieldPath, ".")
	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		return nil
	}
	currentNode := properties
	for i, part := range parts {
		if i == len(parts)-1 {
			propertyDef, ok := currentNode[part].(map[string]interface{})
			if !ok {
				return nil
			}
			if enumVal, exists := propertyDef["enum"]; exists {
				if enumList, ok := enumVal.([]interface{}); ok {
					options := make([]string, 0, len(enumList))
					for _, option := range enumList {
						options = append(options, fmt.Sprintf("%v", option))
					}
					return options
				}
			}
			return nil
		}
		propertyDef, ok := currentNode[part].(map[string]interface{})
		if !ok {
			return nil
		}
		next, ok := propertyDef["properties"].(map[string]interface{})
		if !ok {
			return nil
		}
		currentNode = next
	}
	return nil
}

// getDescriptionForPath retrieves the description for a given field path from the schema.
func getDescriptionForPath(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		return ""
	}
	currentNode := properties
	for i, part := range parts {
		propertyDef, ok := currentNode[part].(map[string]interface{})
		if !ok {
			return ""
		}
		if i == len(parts)-1 {
			if desc, exists := propertyDef["description"]; exists {
				return fmt.Sprintf("%v", desc)
			}
			return ""
		}
		next, ok := propertyDef["properties"].(map[string]interface{})
		if !ok {
			return ""
		}
		currentNode = next
	}
	return ""
}

func isSensitive(fieldPath string) bool {
	return strings.Contains(fieldPath, "api_key") || strings.Contains(fieldPath, "personal_access_token")
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

// editFieldRecursive interactively allows the user to navigate through and edit fields.
// It prints prompts with colorized output and emojis, provides a save and exit option, and
// validates provider configuration if necessary.
func editFieldRecursive(currentMap map[string]interface{}, path string, topMap map[string]interface{}) {
	for {
		clearTerminal()
		// Print the main field selection prompt with a question emoji.
		fmt.Println(ColorGreen + "❓ Which field would you like to edit? (or type " + ColorYellow + "s" + ColorGreen + " to save and exit)" + ColorReset)

		// If in a nested level, provide an uncolored option to go up one level.
		if path != "" {
			fmt.Println("0. Go up one level")
		}

		// Build a list of keys within the current map.
		keys := make([]string, 0, len(currentMap))
		for k := range currentMap {
			keys = append(keys, k)
		}

		// Display each key along with its schema description.
		for i, key := range keys {
			fullPath := key
			if path != "" {
				fullPath = path + key
			}
			desc := getDescriptionForPath(fullPath)
			if desc != "" {
				// Field name in blue, description in white.
				fmt.Printf("%d. "+ColorBlue+"%s"+ColorReset+" - "+ColorWhite+"%s"+ColorReset+"\n", i+1, key, desc)
			} else {
				fmt.Printf("%d. "+ColorBlue+"%s"+ColorReset+"\n", i+1, key)
			}
		}
		// Prompt for user input using the new emoji.
		fmt.Print(ColorCyan + "📝 Enter choice (or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
		var choiceInput string
		fmt.Scan(&choiceInput)
		choiceInput = strings.ToLower(choiceInput)
		if choiceInput == "s" {
			log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
			saveConfigV3(topMap)
			os.Exit(0)
		}
		choice, err := strconv.Atoi(choiceInput)
		if err != nil {
			fmt.Println(ColorRed + "Invalid input, please try again." + ColorReset)
			continue
		}

		// If at nested level and the user selects 0, go up one level.
		if path != "" && choice == 0 {
			return
		}

		// Validate the user's numeric choice.
		if choice < 1 || choice > len(keys) {
			fmt.Println(ColorRed + "Invalid choice, please try again." + ColorReset)
			continue
		}

		// Determine which key was chosen.
		selectedKey := keys[choice-1]
		selectedValue := currentMap[selectedKey]
		newPath := path + selectedKey

		switch v := selectedValue.(type) {
		case map[string]interface{}:
			// Recurse into nested objects.
			editFieldRecursive(v, newPath+".", topMap)
		case string, int, bool:
			// Terminal fields: if enums are defined, display selectable options.
			enumOptions := getEnumOptionsForPath(newPath)
			var newValue string
			if len(enumOptions) > 0 {
				fmt.Printf(ColorGreen+"Select one of the following options for "+ColorBlue+"%s"+ColorGreen+" (or type "+ColorYellow+"s"+ColorGreen+" to save and exit):"+ColorReset+"\n", newPath)
				for i, option := range enumOptions {
					fmt.Printf("%d. "+ColorPurple+"%s"+ColorReset+"\n", i+1, option)
				}
				fmt.Print(ColorCyan + "📝 Enter choice (number or " + ColorYellow + "s" + ColorCyan + "): " + ColorReset)
				var enumInput string
				fmt.Scan(&enumInput)
				enumInput = strings.ToLower(enumInput)
				if enumInput == "s" {
					log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
					saveConfigV3(topMap)
					os.Exit(0)
				}
				choiceNum, err := strconv.Atoi(enumInput)
				if err != nil || choiceNum < 1 || choiceNum > len(enumOptions) {
					fmt.Println(ColorRed + "Invalid choice, input ignored." + ColorReset)
					newValue = fmt.Sprintf("%v", currentMap[selectedKey])
				} else {
					newValue = enumOptions[choiceNum-1]
				}
			} else {
				// Freeform input for terminal fields
				if isSensitive(newPath) {
					newValue = readHiddenInput(ColorCyan + "📝 Enter new value for " + ColorBlue + newPath + ColorCyan + " (input hidden, or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
				} else {
					fmt.Print(ColorCyan + "📝 Enter new value for " + ColorBlue + newPath + ColorCyan + " (or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
					fmt.Scan(&newValue)
				}
				if strings.ToLower(newValue) == "s" {
					log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
					saveConfigV3(topMap)
					os.Exit(0)
				}
			}
			if newValue != "" {
				currentMap[selectedKey] = newValue
			}

			// Provider validation: for llm_config.default_llm_provider, ensure provider details are set.
			if newPath == "llm_config.default_llm_provider" {
				llmConfRaw, ok := topMap["llm_config"]
				if !ok {
					log.Println(ColorRed + "llm_config is missing." + ColorReset)
				} else if llmConf, ok := llmConfRaw.(map[string]interface{}); ok {
					providersRaw, ok := llmConf["llm_providers"]
					if !ok {
						log.Println(ColorRed + "llm_providers is missing." + ColorReset)
					} else if providers, ok := providersRaw.(map[string]interface{}); ok {
						providerConfRaw, exists := providers[newValue]
						if !exists {
							providerConfRaw = map[string]interface{}{
								"api_key":      "",
								"client_model": "",
							}
							providers[newValue] = providerConfRaw
						}
						if provConf, ok := providerConfRaw.(map[string]interface{}); ok {
							apiKey, _ := provConf["api_key"].(string)
							clientModel, _ := provConf["client_model"].(string)
							// If either api_key or client_model is missing, prompt the user.
							if apiKey == "" || clientModel == "" {
								fmt.Printf(ColorRed+"Provider %s is not fully configured."+ColorReset+"\n", newValue)
								// Prompt for api_key if missing.
								if apiKey == "" {
									apiKeyInput := readHiddenInput(ColorCyan + "📝 Enter api_key for " + ColorBlue + newValue + ColorCyan + " (input hidden, or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
									if strings.ToLower(apiKeyInput) == "s" {
										log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
										saveConfigV3(topMap)
										os.Exit(0)
									}
									if apiKeyInput == "" {
										fmt.Println(ColorRed + "Invalid api_key. Provider not configured. Please select another default provider." + ColorReset)
										continue
									}
									provConf["api_key"] = apiKeyInput
								}
								// Prompt for client_model using enum options if missing.
								if clientModel == "" {
									fieldPath := "llm_config.llm_providers." + newValue + ".client_model"
									enumOptions := getEnumOptionsForPath(fieldPath)
									var clientModelInput string
									if len(enumOptions) > 0 {
										fmt.Printf(ColorGreen+"Select one of the following options for "+ColorBlue+"%s"+ColorGreen+" (or type "+ColorYellow+"s"+ColorGreen+" to save and exit):"+ColorReset+"\n", fieldPath)
										for i, option := range enumOptions {
											fmt.Printf("%d. "+ColorPurple+"%s"+ColorReset+"\n", i+1, option)
										}
										fmt.Print(ColorCyan + "📝 Enter choice (number or " + ColorYellow + "s" + ColorCyan + "): " + ColorReset)
										var cmInput string
										fmt.Scan(&cmInput)
										cmInput = strings.ToLower(cmInput)
										if cmInput == "s" {
											log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
											saveConfigV3(topMap)
											os.Exit(0)
										}
										choiceNum, err := strconv.Atoi(cmInput)
										if err != nil || choiceNum < 1 || choiceNum > len(enumOptions) {
											fmt.Println(ColorRed + "Invalid choice, provider not configured. Please select another default provider." + ColorReset)
											continue
										}
										clientModelInput = enumOptions[choiceNum-1]
									} else {
										fmt.Print(ColorCyan + "📝 Enter client_model for " + ColorBlue + newValue + ColorCyan + " (or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
										fmt.Scan(&clientModelInput)
										if strings.ToLower(clientModelInput) == "s" {
											log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
											saveConfigV3(topMap)
											os.Exit(0)
										}
										if clientModelInput == "" {
											fmt.Println(ColorRed + "Invalid client_model. Provider not configured. Please select another default provider." + ColorReset)
											continue
										}
									}
									provConf["client_model"] = clientModelInput
								}
								// Save after provider configuration is complete.
								saveConfigV3(topMap)
							}
						} else {
							log.Println(ColorRed + "Provider configuration is invalid." + ColorReset)
						}
					} else {
						log.Println(ColorRed + "Invalid llm_providers structure." + ColorReset)
					}
				} else {
					log.Println(ColorRed + "Invalid llm_config structure." + ColorReset)
				}
			}

			// Save config after any terminal field edit and prompt for further action.
			saveConfigV3(topMap)
			fmt.Print(ColorGreen + "Do you want to continue editing? (" + ColorWhite + "y/n" + ColorGreen + "/" + ColorYellow + "s" + ColorGreen + " for save and exit): " + ColorReset)
			var cont string
			fmt.Scan(&cont)
			cont = strings.ToLower(cont)
			if cont == "s" || cont == "n" {
				log.Println(ColorGreen + "✅ All changes saved. You can view the config file by running 'gdi config --show'." + ColorReset)
				os.Exit(0)
			} else if cont != "y" {
				log.Println(ColorRed + "Invalid input, returning to main prompt." + ColorReset)
				return
			}
		default:
			fmt.Println(ColorRed + "Unsupported field type." + ColorReset)
			return
		}
	}
}

// saveConfigV3 writes the updated configuration map to the config file.
func saveConfigV3(configMap map[string]interface{}) {
	file, err := os.Create(config.ConfigFilePath)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to open config file for writing: %v"+ColorReset, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(configMap)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to encode config to YAML: %v"+ColorReset, err)
	}
}

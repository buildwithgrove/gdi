package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/config"
)

// ANSI color definitions
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorWhite  = "\033[37m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorCyan   = "\033[36m"
)

var (
	show      bool
	editor    string
	schemaMap map[string]interface{}
)

func init() {
	ConfigCmd.Flags().BoolVarP(&show, "show", "s", false, "Show the configuration.")
	ConfigCmd.Flags().StringVarP(&editor, "editor", "e", "", "Edit the configuration in the given text editor.")
}

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit the configuration for the application.",
	Long: `Edit the configuration for the application.

This command will help you edit the configuration YAML file for the application.

The configuration file is located at ~/.config.gdi.yaml.

You can use the --editor flag to open the configuration file in your default text editor.

You can use the --show flag to print the configuration to the console.`,
	Run: func(cmd *cobra.Command, args []string) {
		if show {
			showConfig()
			return
		}
		if editor != "" {
			editConfig(editor)
			return
		}
		interactiveEditConfigV3()
	},
}

func showConfig() {
	data, err := os.ReadFile(config.ConfigFilePath)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to read config file: %v"+ColorReset, err)
	}
	fmt.Println(string(data))
}

func editConfig(editor string) {
	cmd := exec.Command(editor, config.ConfigFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Run()
}

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

	loadSchema()

	for {
		clearTerminal()
		editFieldRecursive(configMap, "", configMap)

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

func clearTerminal() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func loadSchema() {
	schemaFile := "./config/config.schema.yaml"
	data, err := os.ReadFile(schemaFile)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to read schema file at %s: %v"+ColorReset, schemaFile, err)
	}
	err = yaml.Unmarshal(data, &schemaMap)
	if err != nil {
		log.Fatalf(ColorRed+"Failed to unmarshal schema YAML: %v"+ColorReset, err)
	}
}

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

func editFieldRecursive(currentMap map[string]interface{}, path string, topMap map[string]interface{}) {
	for {
		clearTerminal()
		fmt.Println(ColorGreen + "❓ Which field would you like to edit? (or type " + ColorYellow + "s" + ColorGreen + " to save and exit)" + ColorReset)

		if path != "" {
			fmt.Println("0. Go up one level")
		}

		keys := make([]string, 0, len(currentMap))
		for k := range currentMap {
			keys = append(keys, k)
		}

		for i, key := range keys {
			fullPath := key
			if path != "" {
				fullPath = path + key
			}
			desc := getDescriptionForPath(fullPath)
			if desc != "" {
				fmt.Printf("%d. "+ColorBlue+"%s"+ColorReset+" - "+ColorWhite+"%s"+ColorReset+"\n", i+1, key, desc)
			} else {
				fmt.Printf("%d. "+ColorBlue+"%s"+ColorReset+"\n", i+1, key)
			}
		}
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

		if path != "" && choice == 0 {
			return
		}

		if choice < 1 || choice > len(keys) {
			fmt.Println(ColorRed + "Invalid choice, please try again." + ColorReset)
			continue
		}

		selectedKey := keys[choice-1]
		selectedValue := currentMap[selectedKey]
		newPath := path + selectedKey

		switch v := selectedValue.(type) {
		case map[string]interface{}:
			editFieldRecursive(v, newPath+".", topMap)
		case string, int, bool:
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
				fmt.Print(ColorCyan + "📝 Enter new value for " + ColorBlue + newPath + ColorCyan + " (or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
				fmt.Scan(&newValue)
				if strings.ToLower(newValue) == "s" {
					log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
					saveConfigV3(topMap)
					os.Exit(0)
				}
			}
			if newValue != "" {
				currentMap[selectedKey] = newValue
			}

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
							if apiKey == "" || clientModel == "" {
								fmt.Printf(ColorRed+"Provider %s is not fully configured."+ColorReset+"\n", newValue)
								if apiKey == "" {
									fmt.Print(ColorCyan + "📝 Enter api_key for " + ColorBlue + newValue + ColorCyan + " (or " + ColorYellow + "s" + ColorCyan + " to save and exit): " + ColorReset)
									var apiKeyInput string
									fmt.Scan(&apiKeyInput)
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

			saveConfigV3(topMap)
			fmt.Print(ColorGreen + "Do you want to continue editing? (" + ColorWhite + "y/n" + ColorGreen + "/" + ColorYellow + "s" + ColorGreen + " for save and exit): " + ColorReset)
			var cont string
			fmt.Scan(&cont)
			cont = strings.ToLower(cont)
			if cont == "s" || cont == "n" {
				log.Println(ColorGreen + "✅ All changes saved." + ColorReset)
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

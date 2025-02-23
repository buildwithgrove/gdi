package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/config"
)

var (
	show   bool
	editor string
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
		log.Fatalf("Failed to read config file: %v", err)
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
		log.Fatalf("Failed to read config file: %v", err)
	}

	var configMap map[string]interface{}
	err = yaml.Unmarshal(data, &configMap)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	for {
		clearTerminal()
		editFieldRecursive(configMap, "", configMap)

		saveConfigV3(configMap)
		fmt.Print("Do you want to continue editing? (y/n): ")
		var cont string
		fmt.Scan(&cont)
		if cont != "y" && cont != "n" {
			log.Println("Invalid input, please try again.")
			continue
		}
		if cont == "n" {
			log.Println("All changes saved.")
			os.Exit(0)
		}
		if cont == "y" {
			continue
		}
	}
}

func clearTerminal() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func editFieldRecursive(currentMap map[string]interface{}, path string, topMap map[string]interface{}) {
	for {
		clearTerminal()
		fmt.Println("Which field would you like to edit:")

		// If at a nested level (path is not empty), provide an option to go up one level.
		if path != "" {
			fmt.Println("0. Go up one level")
		}

		keys := make([]string, 0, len(currentMap))
		for k := range currentMap {
			keys = append(keys, k)
		}

		// List out the keys with index starting at 1.
		for i, key := range keys {
			fmt.Printf("%d. %s\n", i+1, key)
		}
		fmt.Print("Enter choice: ")

		var choice int
		fmt.Scan(&choice)

		// If at a nested level and the user selects 0, go up one level.
		if path != "" && choice == 0 {
			return
		}

		if choice < 1 || choice > len(keys) {
			fmt.Println("Invalid choice, please try again.")
			continue
		}

		selectedKey := keys[choice-1]
		selectedValue := currentMap[selectedKey]
		newPath := path + selectedKey

		switch v := selectedValue.(type) {
		case map[string]interface{}:
			editFieldRecursive(v, newPath+".", topMap)
		case string, int, bool:
			fmt.Printf("Enter new value for %s: ", newPath)
			var newValue string
			fmt.Scan(&newValue)
			if newValue != "" {
				currentMap[selectedKey] = newValue
			}
			saveConfigV3(topMap)
			fmt.Print("Do you want to continue editing? (y/n): ")
			var cont string
			fmt.Scan(&cont)
			if cont == "n" {
				log.Println("All changes saved.")
				os.Exit(0)
			} else if cont != "y" {
				log.Println("Invalid input, returning to main prompt.")
				return
			}
		default:
			fmt.Println("Unsupported field type.")
			return
		}
	}
}

func saveConfigV3(configMap map[string]interface{}) {
	file, err := os.Create(config.ConfigFilePath)
	if err != nil {
		log.Fatalf("Failed to open config file for writing: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(configMap)
	if err != nil {
		log.Fatalf("Failed to encode config to YAML: %v", err)
	}
}

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

	"github.com/spf13/cobra"

	"github.com/buildwithgrove/gdi/config"
	cfgEditor "github.com/buildwithgrove/gdi/config/editor"
)

var (
	show   bool
	editor string
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
		schema, err := config.LoadSchema()
		if err != nil {
			log.Fatalf("Failed to load schema: %v", err)
		}

		yamlEditor, err := cfgEditor.NewYAMLEditor(config.ConfigFilePath, schema)
		if err != nil {
			log.Fatalf("Failed to create editor: %v", err)
		}

		yamlEditor.InteractiveEditConfigV3()
	},
}

// showConfig prints the current configuration file to stdout.
func showConfig() {
	data, err := os.ReadFile(config.ConfigFilePath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
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

package editor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/log"
)

func init() {
	// Set up a channel to listen for interrupt signals.
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to handle the signal.
	go func() {
		<-signalChannel
		clearTerminal()
		os.Exit(0)
	}()
}

type YAMLEditor struct {
	configNode          *yaml.Node
	configFilePath      string
	schemaNode          *yaml.Node
	customFieldHandlers map[string]func()
}

// NewYAMLEditor loads a YAML document into a Node and sets up the editor.
func NewYAMLEditor(configFilePath string, schemaNode *yaml.Node) (*YAMLEditor, error) {
	configNode, err := loadConfigNode(configFilePath)
	if err != nil {
		return nil, err
	}

	return &YAMLEditor{
		configNode:     configNode,
		configFilePath: configFilePath,
		schemaNode:     schemaNode,
	}, nil
}

// loadConfigNode reads the YAML file and unmarshals it into a yaml.Node.
func loadConfigNode(configFilePath string) (*yaml.Node, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	err = yaml.Unmarshal(data, &node)
	if err != nil {
		return nil, err
	}
	// The top-level YAML document is the first content node.
	if len(node.Content) > 0 {
		return node.Content[0], nil
	}
	return &node, nil
}

// InteractiveEditConfigV3 runs the interactive loop for editing the YAML config.
func (e *YAMLEditor) InteractiveEditConfigV3() {
	for {
		clearTerminal()
		e.editFieldRecursive(e.configNode, "")

		// Ask for confirmation to continue editing.
		cont := readInput(log.Green + "Do you want to continue editing? (y for yes, n to save and exit): " + log.ResetColor)
		cont = strings.ToLower(cont)
		if cont == "n" {
			e.saveConfigAndExit()
		} else if cont != "y" {
			fmt.Println(log.Red + "Invalid input, returning to main prompt." + log.ResetColor)
		}
		// if "y", loop continues.
	}
}

// editFieldRecursive recursively traverses the YAML node tree for interactive editing.
// It allows users to navigate through the YAML structure, edit scalar values, and save changes.
func (e *YAMLEditor) editFieldRecursive(currentNode *yaml.Node, path string) {
	// Check if the current node is a mapping node (i.e., a dictionary-like structure).
	if currentNode.Kind == yaml.MappingNode {
		for {
			clearTerminal() // Clear the terminal for a fresh display.
			fmt.Println(log.Green + "❓ Which field would you like to edit? (or type " + log.Yellow + "s" + log.Green + " to save and exit)" + log.ResetColor)

			// If we're at a nested level, provide an option to go up one level.
			if path != "" {
				fmt.Println("0. Go up one level")
			}

			// Extract keys from the current mapping node for display.
			keys := extractMappingKeys(currentNode)
			printMappingOptions(keys, path, e) // Display the keys with descriptions.

			// Prompt the user for a choice.
			choiceInput := readInput(log.Cyan + "📝 Enter choice (number, or " + log.Yellow + "s" + log.Cyan + " to save and exit): " + log.ResetColor)
			choiceInput = strings.ToLower(choiceInput)
			if choiceInput == "s" {
				e.saveConfigAndExit()
			}

			choice, err := strconv.Atoi(choiceInput)
			if err != nil {
				fmt.Println(log.Red + "Invalid input, please try again." + log.ResetColor)
				continue
			}

			// If we're in a nested level and input is 0, then go up one level.
			if path != "" && choice == 0 {
				return
			}

			// Validate the user's choice.
			if choice < 1 || choice > len(keys) {
				fmt.Println(log.Red + "Invalid choice, please try again." + log.ResetColor)
				continue
			}

			// Calculate the index of the selected key in the node's content.
			index := (choice-1)*2 + 1
			selectedKey := keys[choice-1]
			selectedValueNode := currentNode.Content[index]
			newPath := path + selectedKey

			// Handle the selected YAML node based on its type.
			switch selectedValueNode.Kind {
			case yaml.MappingNode:
				// If the selected node is a mapping, recurse into it.
				e.editFieldRecursive(selectedValueNode, newPath+".")

			case yaml.ScalarNode:
				// If the selected node is a scalar, handle its editing.
				e.handleScalarEditing(selectedValueNode, newPath)

			default:
				fmt.Println(log.Red + "Unsupported node type." + log.ResetColor)
				return
			}
		}
	} else {
		// If the current node is a standalone primitive, handle its editing directly.
		e.handlePrimitiveNodeEditing(currentNode, path)
	}
}

// extractMappingKeys returns the keys (assumed at even indices) for a mapping node.
func extractMappingKeys(node *yaml.Node) []string {
	keys := []string{}
	for i := 0; i < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	return keys
}

// printMappingOptions prints the keys and schema descriptions for a mapping node.
func printMappingOptions(keys []string, path string, e *YAMLEditor) {
	for i, key := range keys {
		fullPath := key
		if path != "" {
			fullPath = path + key
		}
		desc := e.getDescriptionForPath(fullPath)
		if desc != "" {
			fmt.Printf("%d. "+log.Blue+"%s"+log.ResetColor+" - "+log.White+"%s"+log.ResetColor+"\n", i+1, key, desc)
		} else {
			fmt.Printf("%d. "+log.Blue+"%s"+log.ResetColor+"\n", i+1, key)
		}
	}
}

// handleScalarEditing manages editing of a scalar node (leaf) with optional enum options.
func (e *YAMLEditor) handleScalarEditing(node *yaml.Node, path string) {
	enumOptions := e.getEnumOptionsForPath(path)
	var newValue string
	if len(enumOptions) > 0 {
		newValue = e.handleEnumSelection(path, enumOptions, node.Value)
	} else {
		newValue, _ = handleDirectScalarInput(path, isSensitive(path))
	}
	if newValue != "" {
		node.Value = newValue
	}
	// After editing the scalar, ask if the user wants to continue editing.
	contLevel := readInput(log.Green + "Do you want to continue editing the config?" + log.Cyan + " (y for yes, n to save and exit): " + log.ResetColor)
	contLevel = strings.ToLower(contLevel)
	if contLevel == "n" {
		e.saveConfigAndExit()
	} else if contLevel != "y" {
		return
	}
}

// handlePrimitiveNodeEditing handles editing when the current node is a non-mapping primitive.
func (e *YAMLEditor) handlePrimitiveNodeEditing(node *yaml.Node, path string) {
	clearTerminal()
	if !isSensitive(path) {
		fmt.Printf(log.Cyan+"Current value for %s: %s\n"+log.ResetColor, path, node.Value)
	}
	fmt.Println(log.White + "0. Go up one level" + log.ResetColor)
	newValue, goBack := handleDirectScalarInput(path, isSensitive(path))
	if goBack {
		return
	}
	node.Value = newValue
}

// handleEnumSelection manages enum selection from given options.
// Returns the selected value.
func (e *YAMLEditor) handleEnumSelection(path string, options []string, currentVal string) string {
	clearTerminal()
	fmt.Printf(log.Green+"Select one of the following options for "+log.Blue+"%s"+log.Green+" (or type "+log.Yellow+"s"+log.Green+" to save and exit):"+log.ResetColor+"\n", path)
	fmt.Println(log.White + "0. Go back" + log.ResetColor)
	// List options
	for i, option := range options {
		fmt.Printf("%d. "+log.Purple+"%s"+log.ResetColor+"\n", i+1, option)
	}
	enumInput := readInput(log.Cyan + "📝 Enter choice: " + log.ResetColor)
	enumInput = strings.ToLower(enumInput)
	if enumInput == "s" {
		e.saveConfigAndExit()
	}
	choiceNum, err := strconv.Atoi(enumInput)
	if err != nil || choiceNum < 0 || choiceNum > len(options) {
		fmt.Println(log.Red + "Invalid choice, input ignored." + log.ResetColor)
		return currentVal
	} else if choiceNum == 0 {
		return ""
	} else {
		return options[choiceNum-1]
	}
}

// handleDirectScalarInput prompts the user directly for a new value for a scalar.
// It clears the terminal and prints a "0. Go up one level" option.
// Returns the new value and a bool indicating if the user wants to go back.
func handleDirectScalarInput(path string, sensitive bool) (string, bool) {
	clearTerminal()
	if !sensitive {
		fmt.Printf(log.Cyan+"Current value for %s: %s\n"+log.ResetColor, path, "")
	}
	fmt.Println(log.White + "0. Go up one level" + log.ResetColor)

	newValue := promptForValue(path, sensitive)

	for newValue == "" {
		fmt.Println(log.Red + "Input cannot be empty. Please enter a valid newValue." + log.ResetColor)
		newValue = promptForValue(path, sensitive)
	}

	if newValue == "0" {
		return "", true
	}

	return newValue, false
}

// getEnumOptionsForPath traverses the schemaNode using a dot-delimited field path and returns the allowed enum options if available.
func (e *YAMLEditor) getEnumOptionsForPath(fieldPath string) []string {
	parts := strings.Split(fieldPath, ".")
	props := getMappingValue(e.schemaNode, "properties")
	if props == nil {
		return nil
	}

	current := props
	for i, part := range parts {
		node := getMappingValue(current, part)
		if node == nil {
			return nil
		}
		if i == len(parts)-1 {
			enumNode := getMappingValue(node, "enum")
			if enumNode == nil || enumNode.Kind != yaml.SequenceNode {
				return nil
			}
			var options []string
			for _, item := range enumNode.Content {
				options = append(options, item.Value)
			}
			return options
		}
		next := getMappingValue(node, "properties")
		if next == nil {
			return nil
		}
		current = next
	}
	return nil
}

// getDescriptionForPath retrieves the description for a given field path from the schemaNode.
func (e *YAMLEditor) getDescriptionForPath(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	props := getMappingValue(e.schemaNode, "properties")
	if props == nil {
		return ""
	}

	current := props
	for i, part := range parts {
		node := getMappingValue(current, part)
		if node == nil {
			return ""
		}
		if i == len(parts)-1 {
			descNode := getMappingValue(node, "description")
			if descNode != nil {
				return descNode.Value
			}
			return ""
		}
		next := getMappingValue(node, "properties")
		if next == nil {
			return ""
		}
		current = next
	}
	return ""
}

// Helper function: getMappingValue returns the value node corresponding to a key in a mapping node.
func getMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		v := node.Content[i+1]
		if k.Value == key {
			return v
		}
	}
	return nil
}

// saveConfig writes the updated YAML Node configuration to the config file,
// then clears the terminal, prints a success message, and exits.
func (e *YAMLEditor) saveConfigAndExit() {
	file, err := os.Create(e.configFilePath)
	if err != nil {
		fmt.Printf(log.Red+"Failed to open config file for writing: %v"+log.ResetColor, err)
	}
	defer file.Close()
	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(e.configNode)
	if err != nil {
		fmt.Printf(log.Red+"Failed to encode config to YAML: %v"+log.ResetColor, err)
	}

	clearTerminal()

	fmt.Println(log.Green + "✅ All changes saved successfully to " + e.configFilePath + ".\n" + log.Blue + "💡 To view the updated raw config, run `gdi config --show`." + log.ResetColor)

	os.Exit(0)
}

/* ------------ Helper functions ------------ */

// clearTerminal clears the console for a fresh prompt display.
func clearTerminal() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// isSensitive returns true if the field path indicates sensitive information.
func isSensitive(fieldPath string) bool {
	fieldPath = strings.ToLower(fieldPath)
	return strings.Contains(fieldPath, "api_key") ||
		strings.Contains(fieldPath, "personal_access_token") ||
		strings.Contains(fieldPath, "private_key") ||
		strings.Contains(fieldPath, "signature")
}

// promptForValue prompts the user for a new value for a field.
func promptForValue(path string, sensitive bool) string {
	var newValue string
	if sensitive {
		newValue = readHiddenInput(log.Cyan + "📝 Enter new value for " + log.Blue + path + log.Cyan + " (input hidden): " + log.ResetColor)
	} else {
		newValue = readInput(log.Cyan + "📝 Enter new value for " + log.Blue + path + log.Cyan + ": " + log.ResetColor)
	}
	return newValue
}

// readInput displays a prompt and returns the user's input.clear
func readInput(prompt string) string {
	fmt.Print(prompt)
	// Use bufio.NewReader to read the entire line including empty responses
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(log.Red + "Error reading input." + log.ResetColor)
		return readInput(prompt)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		fmt.Println(log.Red + "Input cannot be empty. Please enter a valid value." + log.ResetColor)
		// Print the prompt again on a new line
		return readInput(prompt)
	}
	return input
}

// readHiddenInput reads input with hidden echo (used for sensitive fields).
func readHiddenInput(prompt string) string {
	fmt.Print(prompt)
	byteInput, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf(log.Red+"Failed to read hidden input: %v"+log.ResetColor, err)
	}
	fmt.Println("")
	return strings.TrimSpace(string(byteInput))
}

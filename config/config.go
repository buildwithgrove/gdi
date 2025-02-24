package config

import (
	"fmt"
	"io"
	"os"

	_ "embed"

	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/config/git"
	"github.com/buildwithgrove/gdi/config/llm"
)

const configFileName = ".config.gdi.yaml"

// eg. /Users/greg/.config.gdi.yaml
var ConfigFilePath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), configFileName)

// Config represents the configuration for LLM providers and Git.
type Config struct {
	Git  *git.Config `yaml:"git_config"`
	LLMs *llm.Config `yaml:"llm_config"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig() (*Config, error) {
	file, err := os.Open(ConfigFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Embed the schema file in the binary for use in the config command.
// This is to allow the schema file to be loaded in the config command,
// regardless of where the binary is run from.
//
//go:embed config.schema.yaml
var schemaYaml []byte

// LoadSchema loads the embedded schema from the config.schema.yaml file.
func LoadSchema(schema *map[string]interface{}) error {
	if err := yaml.Unmarshal(schemaYaml, &schema); err != nil {
		return err
	}
	return nil
}

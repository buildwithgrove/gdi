package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/buildwithgrove/gdi/config/llm"
)

const configFileName = ".config.gdi.yaml"

// eg. /Users/greg/.config.gdi.yaml
var configFilePath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), configFileName)

// Config represents the configuration for LLM providers and Git.
type Config struct {
	LLMs *llm.LLMConfig `yaml:"llm_config"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig() (*Config, error) {
	file, err := os.Open(configFilePath)
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

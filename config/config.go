package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for LLM providers.
type Config struct {
	LLMs LLMConfig `yaml:"llms"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
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

	return &config, config.validate()
}

// Validate validates the configuration.
func (c *Config) validate() error {
	return c.LLMs.validate()
}

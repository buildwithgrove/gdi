package config

import (
	"fmt"

	"github.com/buildwithgrove/gdi/llm/anthropic"
	"github.com/buildwithgrove/gdi/llm/deepseek"
	"github.com/buildwithgrove/gdi/llm/openapi"
)

// Provider represents a valid LLM provider.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderDeepSeek  Provider = "deepseek"
	ProviderAnthropic Provider = "anthropic"
)

// isValid checks if the given provider is valid.
func (p Provider) isValid() bool {
	switch p {
	case ProviderOpenAI, ProviderDeepSeek, ProviderAnthropic:
		return true
	default:
		return false
	}
}

// Validate checks if the LLMConfig is valid.
func (c *LLMConfig) validate() error {
	if !c.DefaultLLMProvider.isValid() {
		return fmt.Errorf("invalid default LLM provider: %s", c.DefaultLLMProvider)
	}
	if !c.LLMProviders.OpenAI.ClientModel.IsValid() {
		return fmt.Errorf("invalid OpenAI client model: %s", c.LLMProviders.OpenAI.ClientModel)
	}
	if !c.LLMProviders.DeepSeek.ClientModel.IsValid() {
		return fmt.Errorf("invalid DeepSeek client model: %s", c.LLMProviders.DeepSeek.ClientModel)
	}
	if !c.LLMProviders.Anthropic.ClientModel.IsValid() {
		return fmt.Errorf("invalid Anthropic client model: %s", c.LLMProviders.Anthropic.ClientModel)
	}
	return nil
}

// LLMConfig represents the configuration for LLMs.
type LLMConfig struct {
	DefaultLLMProvider Provider           `yaml:"default_llm_provider"`
	LLMProviders       LLMProvidersConfig `yaml:"llm_providers"`
}

// LLMProvidersConfig represents the configuration for all LLM providers.
type LLMProvidersConfig struct {
	OpenAI    OpenAIConfig    `yaml:"openai"`
	DeepSeek  DeepSeekConfig  `yaml:"deepseek"`
	Anthropic AnthropicConfig `yaml:"anthropic"`
}

// OpenAIConfig represents the configuration for the OpenAI provider.
type OpenAIConfig struct {
	APIKey      string               `yaml:"api_key"`
	ClientModel openapi.OpenAPIModel `yaml:"client_model"`
}

// DeepSeekConfig represents the configuration for the DeepSeek provider.
type DeepSeekConfig struct {
	APIKey      string                 `yaml:"api_key"`
	ClientModel deepseek.DeepSeekModel `yaml:"client_model"`
}

// AnthropicConfig represents the configuration for the Anthropic provider.
type AnthropicConfig struct {
	APIKey      string                   `yaml:"api_key"`
	ClientModel anthropic.AnthropicModel `yaml:"client_model"`
}

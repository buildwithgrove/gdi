package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/buildwithgrove/gdi/llm/anthropic"
	"github.com/buildwithgrove/gdi/llm/deepseek"
	"github.com/buildwithgrove/gdi/llm/openai"
)

var (
	errLLMConfigNotFound         = errors.New("config error: LLM config not found")
	errInvalidDefaultLLMProvider = errors.New("config error: invalid default LLM provider: %s.\nValid providers:\n%s")

	errOpenAIConfigNotConfigured = errors.New("config error: OpenAI is not configured")
	errOpenAIAPIKeyNotConfigured = errors.New("config error: OpenAI API key is not configured")
	errInvalidOpenAIClientModel  = errors.New("config error: invalid OpenAI client model: %s.\nValid models: %s")

	errDeepSeekConfigNotConfigured = errors.New("config error: DeepSeek is not configured")
	errDeepSeekAPIKeyNotConfigured = errors.New("config error: DeepSeek API key is not configured")
	errInvalidDeepSeekClientModel  = errors.New("config error: invalid DeepSeek client model: %s.\nValid models: %s")

	errAnthropicConfigNotConfigured = errors.New("config error: Anthropic is not configured")
	errAnthropicAPIKeyNotConfigured = errors.New("config error: Anthropic API key is not configured")
	errInvalidAnthropicClientModel  = errors.New("config error: invalid Anthropic client model: %s.\nValid models: %s")
)

type LLMProviderType string

const (
	ProviderNameOpenAI    LLMProviderType = "openai"
	ProviderNameDeepSeek  LLMProviderType = "deepseek"
	ProviderNameAnthropic LLMProviderType = "anthropic"
)

func (t LLMProviderType) IsValid() bool {
	return t == ProviderNameOpenAI || t == ProviderNameDeepSeek || t == ProviderNameAnthropic
}

func validProvidersStr() string {
	var providers []string
	for _, provider := range []LLMProviderType{
		ProviderNameOpenAI,
		ProviderNameDeepSeek,
		ProviderNameAnthropic,
	} {
		providers = append(providers, string(provider))
	}
	return strings.Join(providers, "\n")
}

// LLMConfig represents the configuration for LLMs.
type LLMConfig struct {
	DefaultLLMProvider LLMProviderType    `yaml:"default_llm_provider"`
	LLMProviders       LLMProvidersConfig `yaml:"llm_providers"`
}

// LLMProvidersConfig represents the configuration for all LLM providers.
type LLMProvidersConfig struct {
	OpenAI    *OpenAIConfig    `yaml:"openai"`
	DeepSeek  *DeepSeekConfig  `yaml:"deepseek"`
	Anthropic *AnthropicConfig `yaml:"anthropic"`
}

// OpenAIConfig represents the configuration for the OpenAI provider.
type OpenAIConfig struct {
	APIKey      string             `yaml:"api_key"`
	ClientModel openai.OpenAIModel `yaml:"client_model"`
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

// Validate checks if the LLMConfig is valid.
func (c *LLMConfig) Validate() error {
	// Validate the LLMConfig is not nil.
	if c == nil {
		return errLLMConfigNotFound
	}

	// Validate the default LLM provider.
	if !c.DefaultLLMProvider.IsValid() {
		return fmt.Errorf(errInvalidDefaultLLMProvider.Error(), c.DefaultLLMProvider, validProvidersStr())
	}

	// Validate the default LLM provider's config.
	switch c.DefaultLLMProvider {
	case ProviderNameOpenAI:
		if c.LLMProviders.OpenAI == nil {
			return errOpenAIConfigNotConfigured
		}
		if c.LLMProviders.OpenAI.APIKey == "" {
			return errOpenAIAPIKeyNotConfigured
		}
		if !c.LLMProviders.OpenAI.ClientModel.IsValid() {
			return fmt.Errorf(
				errInvalidOpenAIClientModel.Error(), c.LLMProviders.OpenAI.ClientModel, openai.ListValidModelsStr(),
			)
		}
	case ProviderNameDeepSeek:
		if c.LLMProviders.DeepSeek == nil {
			return errDeepSeekConfigNotConfigured
		}
		if c.LLMProviders.DeepSeek.APIKey == "" {
			return errDeepSeekAPIKeyNotConfigured
		}
		if !c.LLMProviders.DeepSeek.ClientModel.IsValid() {
			return fmt.Errorf(
				errInvalidDeepSeekClientModel.Error(), c.LLMProviders.DeepSeek.ClientModel, deepseek.ListValidModelsStr(),
			)
		}
	case ProviderNameAnthropic:
		if c.LLMProviders.Anthropic == nil {
			return errAnthropicConfigNotConfigured
		}
		if c.LLMProviders.Anthropic.APIKey == "" {
			return errAnthropicAPIKeyNotConfigured
		}
		if !c.LLMProviders.Anthropic.ClientModel.IsValid() {
			return fmt.Errorf(
				errInvalidAnthropicClientModel.Error(), c.LLMProviders.Anthropic.ClientModel, anthropic.ListValidModelsStr(),
			)
		}
	}

	return nil
}

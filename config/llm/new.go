package llm

import (
	"fmt"

	"github.com/buildwithgrove/gdi/llm"
	"github.com/buildwithgrove/gdi/llm/anthropic"
	"github.com/buildwithgrove/gdi/llm/deepseek"
	"github.com/buildwithgrove/gdi/llm/openai"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

type ProviderFlag func(cfg *LLMConfig)

func NewLLMProvider(logger polylog.Logger, llmConfig *LLMConfig, flags ...ProviderFlag) (llm.LLMProvider, error) {
	for _, flag := range flags {
		flag(llmConfig)
	}

	if err := llmConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid LLM config: %w", err)
	}

	provider := llmConfig.DefaultLLMProvider

	switch provider {

	case ProviderNameOpenAI:
		return openai.NewOpenAIProvider(openai.Config{
			Logger:      logger,
			APIKey:      llmConfig.LLMProviders.OpenAI.APIKey,
			ClientModel: llmConfig.LLMProviders.OpenAI.ClientModel,
		}), nil

	case ProviderNameDeepSeek:
		return deepseek.NewDeepseekProvider(deepseek.Config{
			Logger:      logger,
			APIKey:      llmConfig.LLMProviders.DeepSeek.APIKey,
			ClientModel: llmConfig.LLMProviders.DeepSeek.ClientModel,
		}), nil

	case ProviderNameAnthropic:
		return anthropic.NewAnthropicProvider(anthropic.Config{
			Logger:      logger,
			APIKey:      llmConfig.LLMProviders.Anthropic.APIKey,
			ClientModel: llmConfig.LLMProviders.Anthropic.ClientModel,
		}), nil

	default:
		return nil, fmt.Errorf("invalid LLM provider: %s", provider)
	}
}

func WithLLMProviderOverride(provider LLMProviderType) ProviderFlag {
	return func(cfg *LLMConfig) {
		cfg.DefaultLLMProvider = provider
	}
}

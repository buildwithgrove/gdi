package llm

import (
	"context"
)

// TODO_IMPROVE(@commoddity): Add token length configuration to LLM packages to ensure prompt does not go over token limit.

type PromptFlag func(cfg *PromptConfig)

type LLMProvider interface {
	SendPrompt(ctx context.Context, prompt string, flags ...PromptFlag) (string, error)
}

type PromptConfig struct {
	Model string
}

func WithLLMModelOverride(model string) PromptFlag {
	return func(cfg *PromptConfig) {
		cfg.Model = model
	}
}

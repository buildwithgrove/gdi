package llm

import "context"

type LLMProvider interface {
	SendPrompt(ctx context.Context, prompt string, config ...PromptConfig) (string, error)
}

type PromptConfig struct {
	OverrideModel string
}

package openrouter

import (
	"strings"
)

type OpenRouterModel string

// This is a simplified list of models that are supported by the OpenAI API.
// This should cover any use case we have for now.
// More models can be added as needed from the OpenRouter API site:
// https://openrouter.ai/models
const (
	modelDeepseekChat            OpenRouterModel = OpenRouterModel("deepseek/deepseek-chat")
	modelDeepseekR1              OpenRouterModel = OpenRouterModel("deepseek/deepseek-r1")
	modelOpenAIGPT4o             OpenRouterModel = OpenRouterModel("openai/chatgpt-4o-latest")
	modelOpenAIO1                OpenRouterModel = OpenRouterModel("openai/o1")
	modelAnthropicClaude37Sonnet OpenRouterModel = OpenRouterModel("anthropic/claude-3.7-sonnet")
	modelAnthropicClaude35Haiku  OpenRouterModel = OpenRouterModel("anthropic/claude-3.5-haiku")
	modelAnthropicClaude3Opus    OpenRouterModel = OpenRouterModel("anthropic/claude-3-opus")
	modelQwenQwenTurbo           OpenRouterModel = OpenRouterModel("qwen/qwen-turbo")
)

func (m OpenRouterModel) IsValid() bool {
	switch m {
	case modelDeepseekChat,
		modelDeepseekR1,
		modelOpenAIGPT4o,
		modelOpenAIO1,
		modelAnthropicClaude37Sonnet,
		modelAnthropicClaude35Haiku,
		modelAnthropicClaude3Opus,
		modelQwenQwenTurbo:
		return true
	default:
		return false
	}
}

func ListValidModelsStr() string {
	var models []string
	for _, model := range []OpenRouterModel{
		modelDeepseekChat,
		modelDeepseekR1,
		modelOpenAIGPT4o,
		modelOpenAIO1,
		modelAnthropicClaude37Sonnet,
		modelAnthropicClaude35Haiku,
		modelAnthropicClaude3Opus,
		modelQwenQwenTurbo,
	} {
		models = append(models, string(model))
	}
	return strings.Join(models, "\n")
}

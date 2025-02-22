package anthropic

import (
	"slices"

	anthropic "github.com/liushuangls/go-anthropic/v2"
)

type AnthropicModel string

// This is a simplified list of models that are supported by the Anthropic API.
// This should cover any use case we have for now.
var validModels = []AnthropicModel{
	AnthropicModel(anthropic.ModelClaude3Dot5HaikuLatest),
	AnthropicModel(anthropic.ModelClaude3Dot5SonnetLatest),
	AnthropicModel(anthropic.ModelClaude3Opus20240229),
}

func (m AnthropicModel) IsValid() bool {
	switch {
	case slices.ContainsFunc(validModels, func(v AnthropicModel) bool {
		return v == m
	}):
		return true
	default:
		return false
	}
}

func getDefaultModel() AnthropicModel {
	return AnthropicModel(anthropic.ModelClaude3Dot5SonnetLatest)
}

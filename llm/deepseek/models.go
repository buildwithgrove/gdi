package deepseek

import (
	"slices"

	deepseek "github.com/cohesion-org/deepseek-go"
)

type DeepSeekModel string

// This is a simplified list of models that are supported by the DeepSeek API.
// This should cover any use case we have for now.
var validModels = []DeepSeekModel{
	deepseek.DeepSeekChat,
	deepseek.DeepSeekCoder,
	deepseek.DeepSeekReasoner,
}

func (m DeepSeekModel) IsValid() bool {
	switch {
	case slices.ContainsFunc(validModels, func(v DeepSeekModel) bool {
		return v == m
	}):
		return true
	default:
		return false
	}
}

func getDefaultModel() DeepSeekModel {
	return DeepSeekModel(deepseek.DeepSeekCoder)
}

package openapi

import (
	"slices"

	"github.com/sashabaranov/go-openai"
)

type OpenAPIModel string

// This is a simplified list of models that are supported by the OpenAI API.
// This should cover any use case we have for now.
var validModels = []OpenAPIModel{
	openai.O1Mini,
	openai.O1,
	openai.O3Mini,
	openai.GPT4o,
}

func (m OpenAPIModel) IsValid() bool {
	switch {
	case slices.ContainsFunc(validModels, func(v OpenAPIModel) bool {
		return v == m
	}):
		return true
	default:
		return false
	}
}

func getDefaultModel() OpenAPIModel {
	return openai.GPT4o
}

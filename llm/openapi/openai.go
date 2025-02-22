package openapi

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/sashabaranov/go-openai"

	"github.com/buildwithgrove/gdi/llm"
)

var _ llm.LLMProvider = &OpenAIProvider{}

type OpenAIProvider struct {
	logger      polylog.Logger
	client      *openai.Client
	clientModel OpenAPIModel
}

type Config struct {
	Logger      polylog.Logger
	APIKey      string
	ClientModel OpenAPIModel
}

func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	if !cfg.ClientModel.IsValid() {
		cfg.Logger.Warn().Msgf("invalid client model %s, using default model: %s", cfg.ClientModel, getDefaultModel())
		cfg.ClientModel = getDefaultModel()
	}

	return &OpenAIProvider{
		logger:      cfg.Logger,
		client:      openai.NewClient(cfg.APIKey),
		clientModel: cfg.ClientModel,
	}
}

func (p *OpenAIProvider) SendPrompt(ctx context.Context, prompt string, config ...llm.PromptConfig) (string, error) {
	model := p.clientModel
	if len(config) > 0 {
		openaiModel := OpenAPIModel(config[0].OverrideModel)
		if !openaiModel.IsValid() {
			return "", fmt.Errorf("invalid OpenAI model: %s", config[0].OverrideModel)
		}
		model = openaiModel
	}

	req := openai.ChatCompletionRequest{
		Model: string(model),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

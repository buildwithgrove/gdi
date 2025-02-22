package deepseek

import (
	"context"
	"fmt"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/cohesion-org/deepseek-go/constants"

	"github.com/buildwithgrove/gdi/llm"
)

var _ llm.LLMProvider = &DeepseekProvider{}

type DeepseekProvider struct {
	client      *deepseek.Client
	clientModel DeepSeekModel
}

type Config struct {
	APIKey      string
	ClientModel DeepSeekModel
}

func NewDeepseekProvider(cfg Config) *DeepseekProvider {
	if !cfg.ClientModel.IsValid() {
		cfg.ClientModel = getDefaultModel()
	}

	return &DeepseekProvider{
		client:      deepseek.NewClient(cfg.APIKey),
		clientModel: cfg.ClientModel,
	}
}

func (p *DeepseekProvider) SendPrompt(ctx context.Context, prompt string, config ...llm.PromptConfig) (string, error) {
	model := p.clientModel
	if len(config) > 0 {
		deepSeekModel := DeepSeekModel(config[0].OverrideModel)
		if !deepSeekModel.IsValid() {
			return "", fmt.Errorf("invalid DeepSeek model: %s", config[0].OverrideModel)
		}
		model = deepSeekModel
	}

	req := deepseek.ChatCompletionRequest{
		Model: string(model),
		Messages: []deepseek.ChatCompletionMessage{
			{
				Role:    constants.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := p.client.CreateChatCompletion(ctx, &req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", nil
	}

	return resp.Choices[0].Message.Content, nil
}

package anthropic

import (
	"context"
	"fmt"

	anthropicapi "github.com/liushuangls/go-anthropic/v2"

	"github.com/buildwithgrove/gdi/llm"
)

var _ llm.LLMProvider = &AnthropicProvider{}

type AnthropicProvider struct {
	client      *anthropicapi.Client
	clientModel AnthropicModel
}

type Config struct {
	APIKey      string
	ClientModel AnthropicModel
}

func NewAnthropicProvider(cfg Config) *AnthropicProvider {
	if !cfg.ClientModel.IsValid() {
		cfg.ClientModel = getDefaultModel()
	}

	return &AnthropicProvider{
		client:      anthropicapi.NewClient(cfg.APIKey),
		clientModel: cfg.ClientModel,
	}
}

func (p *AnthropicProvider) SendPrompt(ctx context.Context, prompt string, config ...llm.PromptConfig) (string, error) {
	model := p.clientModel
	if len(config) > 0 {
		anthropicModel := AnthropicModel(config[0].OverrideModel)
		if !anthropicModel.IsValid() {
			return "", fmt.Errorf("invalid Anthropic model: %s", config[0].OverrideModel)
		}
		model = anthropicModel
	}

	req := anthropicapi.MessagesRequest{
		Model:     anthropicapi.Model(model),
		MaxTokens: 1000,
		Messages: []anthropicapi.Message{
			anthropicapi.NewUserTextMessage(prompt),
		},
	}

	resp, err := p.client.CreateMessages(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Content) == 0 {
		return "", nil
	}

	return resp.Content[0].GetText(), nil
}

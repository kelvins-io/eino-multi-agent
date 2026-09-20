package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
)

func NewChatModel(ctx context.Context, cfg config.LLMConfig) (model.ToolCallingChatModel, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("llm.model is required")
	}
	switch cfg.Provider {
	case "ark":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("llm.api_key is required for ark")
		}
		timeout := cfg.Timeout
		opt := &ark.ChatModelConfig{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			Timeout: &timeout,
		}
		if cfg.BaseURL != "" {
			opt.BaseURL = cfg.BaseURL
		}
		return ark.NewChatModel(ctx, opt)
	case "ollama":
		base := cfg.BaseURL
		if base == "" {
			base = "http://127.0.0.1:11434"
		}
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: base,
			Timeout: cfg.Timeout,
			Model:   cfg.Model,
		})
	case "openai", "":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("llm.api_key is required for openai")
		}
		opt := &openai.ChatModelConfig{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			Timeout: cfg.Timeout,
			BaseURL: cfg.BaseURL,
		}
		return openai.NewChatModel(ctx, opt)
	default:
		return nil, fmt.Errorf("unsupported llm.provider: %s", cfg.Provider)
	}
}

func Ready(cfg config.LLMConfig) error {
	switch cfg.Provider {
	case "ollama":
		if cfg.Model == "" {
			return fmt.Errorf("llm.model is required")
		}
		return nil
	default:
		if cfg.APIKey == "" {
			return fmt.Errorf("尚未配置 LLM API Key，请在 config.yaml 或环境变量 EINO_LLM_API_KEY 中设置")
		}
		if cfg.Model == "" {
			return fmt.Errorf("llm.model is required")
		}
		return nil
	}
}

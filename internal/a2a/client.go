package a2a

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/a2a/client"
	einoa2a "github.com/cloudwego/eino-ext/a2a/extension/eino"
	"github.com/cloudwego/eino-ext/a2a/transport/jsonrpc"
	"github.com/cloudwego/eino/adk"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"go.uber.org/zap"
)

// LoadRemoteAgents 将配置中的远程 A2A Agent 封装为本地 ADK SubAgent。
func LoadRemoteAgents(ctx context.Context, remotes []config.A2ARemoteConfig) ([]adk.Agent, error) {
	if len(remotes) == 0 {
		return nil, nil
	}
	out := make([]adk.Agent, 0, len(remotes))
	log := logx.Named("a2a")
	for _, r := range remotes {
		base := strings.TrimSpace(r.BaseURL)
		if base == "" {
			return nil, fmt.Errorf("a2a remote %q: base_url is required", r.Name)
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = base
		}
		streaming := true
		if r.Streaming != nil {
			streaming = *r.Streaming
		}
		transport, err := jsonrpc.NewTransport(ctx, &jsonrpc.ClientConfig{
			BaseURL:                     base,
			HandlerPath:                 r.HandlerPath,
			HTTPClient:                  httpClientWithBearer(r.AuthToken),
			DisablePrevHeaderForwarding: true,
		})
		if err != nil {
			return nil, fmt.Errorf("a2a remote %q transport: %w", name, err)
		}
		cli, err := client.NewA2AClient(ctx, &client.Config{Transport: transport})
		if err != nil {
			return nil, fmt.Errorf("a2a remote %q client: %w", name, err)
		}
		cfg := einoa2a.AgentConfig{
			Client:    cli,
			Streaming: &streaming,
		}
		if n := strings.TrimSpace(r.Name); n != "" {
			cfg.Name = &n
		}
		if d := strings.TrimSpace(r.Description); d != "" {
			cfg.Description = &d
		}
		ag, err := einoa2a.NewAgent(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("a2a remote %q agent: %w", name, err)
		}
		out = append(out, ag)
		log.Info("remote a2a agent ready",
			zap.String("name", name),
			zap.String("base_url", base),
			zap.String("handler", r.HandlerPath),
			zap.Bool("streaming", streaming),
		)
	}
	return out, nil
}

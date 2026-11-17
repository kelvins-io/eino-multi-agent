package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"go.uber.org/zap"
)

// AgentCardURL 返回本实例对外可发现的 Agent Card 地址。
func AgentCardURL(cfg config.A2AConfig) string {
	base := strings.TrimRight(cfg.PublicURL, "/")
	u, err := url.JoinPath(base, ".well-known/agent-card.json")
	if err != nil {
		return base + "/.well-known/agent-card.json"
	}
	return u
}

// RegisterToRegistries 向配置的注册中心提交 wellKnownURI（启动成功后调用）。
func RegisterToRegistries(ctx context.Context, cfg config.A2AConfig) {
	if len(cfg.Registries) == 0 {
		logx.Named("a2a").Info("a2a auto-register: no registries configured; agent card is live",
			zap.String("agent_card", AgentCardURL(cfg)),
		)
		return
	}
	wellKnown := AgentCardURL(cfg)
	client := &http.Client{Timeout: 15 * time.Second}
	log := logx.Named("a2a")
	for _, reg := range cfg.Registries {
		endpoint := strings.TrimSpace(reg.URL)
		if endpoint == "" {
			continue
		}
		if err := postRegistry(ctx, client, endpoint, reg.AuthToken, wellKnown); err != nil {
			log.Warn("a2a registry register failed",
				zap.String("registry", endpoint),
				zap.String("agent_card", wellKnown),
				zap.Error(err),
			)
			continue
		}
		log.Info("a2a registered to registry",
			zap.String("registry", endpoint),
			zap.String("agent_card", wellKnown),
		)
	}
}

func postRegistry(ctx context.Context, client *http.Client, endpoint, token, wellKnown string) error {
	body, _ := json.Marshal(map[string]string{
		"wellKnownURI": wellKnown,
		"url":          wellKnown,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if t := strings.TrimSpace(token); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

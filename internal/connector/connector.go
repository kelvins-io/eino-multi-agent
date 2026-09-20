package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
	"go.uber.org/zap"
)

const (
	KindWebhook  = "webhook"
	KindLocalDir = "local_dir"
)

type Event struct {
	Task      *store.Task
	Artifacts []store.Artifact
}

type Registry struct {
	store *store.Store
	root  string
	http  *http.Client
}

func New(st *store.Store, workspaceRoot string) *Registry {
	return &Registry{
		store: st,
		root:  workspaceRoot,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *Registry) Notify(ctx context.Context, ev Event) {
	userID := ""
	if ev.Task != nil {
		userID = ev.Task.UserID
	}
	items, err := r.store.EnabledConnectors(ctx, userID)
	if err != nil {
		logx.Named("connector").Error("list connectors", zap.String("user_id", userID), zap.Error(err))
		return
	}
	for i := range items {
		item := items[i]
		taskID := ""
		if ev.Task != nil {
			taskID = ev.Task.ID
		}
		if err := r.Invoke(ctx, &item, ev); err != nil {
			logx.Named("connector").Warn("invoke failed",
				zap.String("connector_id", item.ID),
				zap.String("kind", item.Kind),
				zap.String("task_id", taskID),
				zap.Error(err),
			)
		} else {
			logx.Named("connector").Info("invoked",
				zap.String("connector_id", item.ID),
				zap.String("kind", item.Kind),
				zap.String("task_id", taskID),
			)
		}
	}
}

func (r *Registry) Invoke(ctx context.Context, item *store.Connector, ev Event) error {
	switch item.Kind {
	case KindWebhook:
		return invokeWebhook(ctx, r.http, item, ev)
	case KindLocalDir:
		return invokeLocalDir(r.root, item, ev)
	default:
		return fmt.Errorf("unknown connector kind %s", item.Kind)
	}
}

func invokeWebhook(ctx context.Context, client *http.Client, item *store.Connector, ev Event) error {
	url := item.Config["url"]
	if url == "" {
		return fmt.Errorf("webhook url is required")
	}
	body, _ := json.Marshal(map[string]any{
		"event":     "task.succeeded",
		"task":      ev.Task,
		"artifacts": ev.Artifacts,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret := item.Config["secret"]; secret != "" {
		req.Header.Set("X-Harness-Secret", secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %s", resp.Status)
	}
	return nil
}

func invokeLocalDir(workspaceRoot string, item *store.Connector, ev Event) error {
	if ev.Task == nil {
		return fmt.Errorf("task is required")
	}
	name := item.Config["path"]
	if name == "" {
		name = item.ID
	}
	name = filepath.Base(name)
	dest := filepath.Join(workspaceRoot, "exports", name, ev.Task.ID)
	src := filepath.Join(ev.Task.Workspace, "output")
	return workspace.CopyTree(src, dest)
}

func ValidateKind(kind string) error {
	switch kind {
	case KindWebhook, KindLocalDir:
		return nil
	default:
		return fmt.Errorf("kind must be webhook or local_dir")
	}
}

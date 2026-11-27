package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/confirm"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/eval"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
)

// LocalClient 进程内调用 store / runtime（HTTP MCP 使用），按 UserID 隔离。
type LocalClient struct {
	cfg        *config.Config
	store      *store.Store
	runtime    *harness.Runtime
	sched      *scheduler.Scheduler
	connectors *connector.Registry
	userID     string
	username   string
}

func NewLocalClient(
	cfg *config.Config,
	st *store.Store,
	rt *harness.Runtime,
	sched *scheduler.Scheduler,
	connectors *connector.Registry,
	userID, username string,
) *LocalClient {
	return &LocalClient{
		cfg:        cfg,
		store:      st,
		runtime:    rt,
		sched:      sched,
		connectors: connectors,
		userID:     userID,
		username:   username,
	}
}

func (c *LocalClient) audit(ctx context.Context, action, targetType, targetID, detail string) {
	actor := c.username
	if actor == "" {
		actor = "mcp"
	}
	_ = c.store.CreateAudit(ctx, &store.AuditLog{
		UserID:     c.userID,
		Actor:      actor,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
	})
}

func (c *LocalClient) GetMeta(ctx context.Context) (any, error) {
	type skillDTO struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	skills := make([]skillDTO, 0)
	if c.runtime != nil {
		for _, sk := range c.runtime.Skills() {
			skills = append(skills, skillDTO{Name: sk.Name, Description: sk.Description})
		}
	}
	return map[string]any{
		"llm": map[string]any{
			"provider": c.cfg.LLM.Provider,
			"model":    c.cfg.LLM.Model,
			"ready":    agent.Ready(c.cfg.LLM) == nil,
		},
		"confirm_policies": []string{"on_risk", "always", "never"},
		"skills":           skills,
		"auth_required":    true,
		"auth_mode":        "jwt",
		"mcp": map[string]any{
			"enabled": c.cfg.MCP.Enabled,
			"addr":    c.cfg.MCP.Addr,
			"path":    c.cfg.MCP.Path,
		},
		"a2a": map[string]any{
			"enabled":       c.cfg.A2A.Enabled,
			"auto_register": c.cfg.A2A.AutoRegister,
			"addr":          c.cfg.A2A.Addr,
			"public_url":    c.cfg.A2A.PublicURL,
			"handler_path":  c.cfg.A2A.HandlerPath,
			"agent_card":    strings.TrimRight(c.cfg.A2A.PublicURL, "/") + "/.well-known/agent-card.json",
			"auth":          strings.TrimSpace(c.cfg.A2A.AuthToken) != "",
			"registries":    len(c.cfg.A2A.Registries),
			"remotes":       len(c.cfg.A2A.Remotes),
		},
	}, nil
}

func (c *LocalClient) GetMe(ctx context.Context) (any, error) {
	user, err := c.store.GetUserByID(ctx, c.userID)
	if err != nil {
		return nil, err
	}
	display := user.DisplayName
	if display == "" {
		display = user.Username
	}
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": display,
		"created_at":   user.CreatedAt,
	}, nil
}

func (c *LocalClient) CreateTask(ctx context.Context, in CreateTaskInput) (any, error) {
	if strings.TrimSpace(in.Goal) == "" {
		return nil, fmt.Errorf("goal is required")
	}
	if in.ProjectID != "" {
		if _, err := c.store.GetProjectOwned(ctx, in.ProjectID, c.userID); err != nil {
			return nil, fmt.Errorf("project not found")
		}
	}
	var uploads []harness.Upload
	for _, p := range in.FilePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", p, err)
		}
		defer f.Close()
		uploads = append(uploads, harness.Upload{Name: filepath.Base(p), Reader: f})
	}
	task, err := c.runtime.Create(ctx, harness.CreateInput{
		Title:         in.Title,
		Goal:          in.Goal,
		ConfirmPolicy: in.ConfirmPolicy,
		Skills:        compactStrings(in.Skills),
		ProjectID:     in.ProjectID,
		UserID:        c.userID,
		Files:         uploads,
	})
	if err != nil {
		return nil, err
	}
	c.audit(ctx, "task.create", "task", task.ID, task.Title)
	return task, nil
}

func (c *LocalClient) ListTasks(ctx context.Context, limit int, projectID string) (any, error) {
	if limit <= 0 {
		limit = 50
	}
	items, err := c.store.ListTasks(ctx, limit, projectID, c.userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) GetTask(ctx context.Context, id string) (any, error) {
	task, err := c.store.GetTaskOwned(ctx, id, c.userID)
	if err != nil {
		return nil, err
	}
	events, _ := c.store.ListEvents(ctx, task.ID, 0)
	artifacts, _ := c.store.ListArtifacts(ctx, task.ID)
	return map[string]any{"task": task, "events": events, "artifacts": artifacts}, nil
}

func (c *LocalClient) CancelTask(ctx context.Context, id string) (any, error) {
	if _, err := c.store.GetTaskOwned(ctx, id, c.userID); err != nil {
		return nil, err
	}
	if err := c.runtime.Cancel(ctx, id); err != nil {
		return nil, err
	}
	c.audit(ctx, "task.cancel", "task", id, "")
	return map[string]any{"ok": true}, nil
}

func (c *LocalClient) ConfirmTask(ctx context.Context, id string, approved bool) (any, error) {
	if _, err := c.store.GetTaskOwned(ctx, id, c.userID); err != nil {
		return nil, err
	}
	if err := c.runtime.Confirm(ctx, id, approved); err != nil {
		return nil, err
	}
	detail := "rejected"
	if approved {
		detail = "approved"
	}
	c.audit(ctx, "task.confirm", "task", id, detail)
	return map[string]any{"ok": true}, nil
}

func (c *LocalClient) RetryTask(ctx context.Context, id string) (any, error) {
	if _, err := c.store.GetTaskOwned(ctx, id, c.userID); err != nil {
		return nil, err
	}
	task, err := c.runtime.Retry(ctx, id)
	if err != nil {
		return nil, err
	}
	c.audit(ctx, "task.retry", "task", task.ID, "")
	return task, nil
}

func (c *LocalClient) ListTaskArtifacts(ctx context.Context, taskID string) (any, error) {
	if _, err := c.store.GetTaskOwned(ctx, taskID, c.userID); err != nil {
		return nil, err
	}
	items, err := c.store.ListArtifacts(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) PreviewArtifact(ctx context.Context, taskID string, artifactID uint64) (any, error) {
	task, err := c.store.GetTaskOwned(ctx, taskID, c.userID)
	if err != nil {
		return nil, err
	}
	item, err := c.store.GetArtifact(ctx, task.ID, artifactID)
	if err != nil {
		return nil, err
	}
	if !harness.Previewable(item.Name, item.Mime) {
		return map[string]any{"previewable": false, "name": item.Name, "mime": item.Mime}, nil
	}
	sb, err := workspace.Open(task.Workspace)
	if err != nil {
		return nil, err
	}
	abs, err := sb.ReadArtifact(item.RelPath)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(item.Name))
	if strings.HasPrefix(item.Mime, "image/") || isImageExt(ext) {
		return map[string]any{
			"previewable": true,
			"kind":        "image",
			"name":        item.Name,
			"mime":        item.Mime,
			"note":        "image artifact; use REST download with inline=1 for bytes",
		}, nil
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(raw)
	truncated := false
	if len([]rune(text)) > 20000 {
		text = string([]rune(text)[:20000])
		truncated = true
	}
	return map[string]any{
		"previewable": true,
		"kind":        "text",
		"name":        item.Name,
		"mime":        item.Mime,
		"text":        text,
		"truncated":   truncated,
	}, nil
}

func (c *LocalClient) ListProjects(ctx context.Context) (any, error) {
	items, err := c.store.ListProjects(ctx, c.userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) CreateProject(ctx context.Context, name, description string) (any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	id := uuid.NewString()
	root, err := workspace.NewProject(c.cfg.Workspace.Root, id)
	if err != nil {
		return nil, err
	}
	p := &store.Project{ID: id, UserID: c.userID, Name: name, Description: description, Workspace: root}
	if err := c.store.CreateProject(ctx, p); err != nil {
		return nil, err
	}
	c.audit(ctx, "project.create", "project", p.ID, p.Name)
	return p, nil
}

func (c *LocalClient) GetProject(ctx context.Context, id string) (any, error) {
	p, err := c.store.GetProjectOwned(ctx, id, c.userID)
	if err != nil {
		return nil, err
	}
	files, _ := workspace.ListShared(p.Workspace)
	tasks, _ := c.store.ListTasks(ctx, 50, p.ID, c.userID)
	return map[string]any{"project": p, "files": files, "tasks": tasks}, nil
}

func (c *LocalClient) UploadProjectFiles(ctx context.Context, projectID string, filePaths []string) (any, error) {
	p, err := c.store.GetProjectOwned(ctx, projectID, c.userID)
	if err != nil {
		return nil, err
	}
	for _, path := range filePaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", path, err)
		}
		_, err = workspace.SaveShared(p.Workspace, filepath.Base(path), f)
		_ = f.Close()
		if err != nil {
			return nil, err
		}
	}
	files, _ := workspace.ListShared(p.Workspace)
	return map[string]any{"files": files}, nil
}

func (c *LocalClient) ListSchedules(ctx context.Context) (any, error) {
	items, err := c.store.ListSchedules(ctx, c.userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) CreateSchedule(ctx context.Context, in CreateScheduleInput) (any, error) {
	if strings.TrimSpace(in.Goal) == "" {
		return nil, fmt.Errorf("goal is required")
	}
	if in.ProjectID != "" {
		if _, err := c.store.GetProjectOwned(ctx, in.ProjectID, c.userID); err != nil {
			return nil, fmt.Errorf("project not found")
		}
	}
	tz := in.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc := scheduler.LoadLocation(tz)
	var runAt *time.Time
	if in.Kind == scheduler.KindOnce && in.RunAt != "" {
		t, err := time.ParseInLocation(time.RFC3339, in.RunAt, loc)
		if err != nil {
			return nil, fmt.Errorf("run_at must be RFC3339")
		}
		runAt = &t
	}
	next, err := scheduler.NextRun(time.Now(), loc, in.Kind, in.Weekday, in.Hour, in.Minute, runAt)
	if err != nil {
		return nil, err
	}
	title := in.Title
	if title == "" {
		title = in.Goal
		if len([]rune(title)) > 40 {
			title = string([]rune(title)[:40])
		}
	}
	item := &store.Schedule{
		ID:            uuid.NewString(),
		UserID:        c.userID,
		ProjectID:     in.ProjectID,
		Title:         title,
		Goal:          in.Goal,
		Skills:        compactStrings(in.Skills),
		ConfirmPolicy: confirm.Normalize(in.ConfirmPolicy),
		Kind:          in.Kind,
		Weekday:       in.Weekday,
		Hour:          in.Hour,
		Minute:        in.Minute,
		RunAt:         runAt,
		Timezone:      tz,
		Enabled:       true,
		NextRunAt:     &next,
	}
	if err := c.store.CreateSchedule(ctx, item); err != nil {
		return nil, err
	}
	c.audit(ctx, "schedule.create", "schedule", item.ID, item.Title)
	return item, nil
}

func (c *LocalClient) RunSchedule(ctx context.Context, id string) (any, error) {
	if c.sched == nil {
		return nil, fmt.Errorf("scheduler not ready")
	}
	if _, err := c.store.GetScheduleOwned(ctx, id, c.userID); err != nil {
		return nil, err
	}
	task, err := c.sched.FireNow(ctx, id)
	if err != nil {
		return nil, err
	}
	c.audit(ctx, "schedule.run", "schedule", id, task.ID)
	return task, nil
}

func (c *LocalClient) ToggleSchedule(ctx context.Context, id string) (any, error) {
	item, err := c.store.GetScheduleOwned(ctx, id, c.userID)
	if err != nil {
		return nil, err
	}
	item.Enabled = !item.Enabled
	if err := c.store.SaveSchedule(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *LocalClient) ListConnectors(ctx context.Context) (any, error) {
	items, err := c.store.ListConnectors(ctx, c.userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) CreateConnector(ctx context.Context, name, kind string, config map[string]string) (any, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := connector.ValidateKind(kind); err != nil {
		return nil, err
	}
	if config == nil {
		config = map[string]string{}
	}
	item := &store.Connector{
		ID:      uuid.NewString(),
		UserID:  c.userID,
		Name:    name,
		Kind:    kind,
		Config:  config,
		Enabled: true,
	}
	if err := c.store.CreateConnector(ctx, item); err != nil {
		return nil, err
	}
	c.audit(ctx, "connector.create", "connector", item.ID, item.Name)
	return item, nil
}

func (c *LocalClient) ToggleConnector(ctx context.Context, id string) (any, error) {
	item, err := c.store.GetConnectorOwned(ctx, id, c.userID)
	if err != nil {
		return nil, err
	}
	item.Enabled = !item.Enabled
	if err := c.store.SaveConnector(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *LocalClient) TestConnector(ctx context.Context, id string) (any, error) {
	item, err := c.store.GetConnectorOwned(ctx, id, c.userID)
	if err != nil {
		return nil, err
	}
	if c.connectors == nil {
		return nil, fmt.Errorf("connectors not ready")
	}
	ev := connector.Event{Task: &store.Task{
		ID:        "test",
		UserID:    c.userID,
		Title:     "连接器测试",
		Status:    store.StatusSucceeded,
		Workspace: c.cfg.Workspace.Root,
	}}
	if err := c.connectors.Invoke(ctx, item, ev); err != nil {
		return nil, err
	}
	c.audit(ctx, "connector.test", "connector", item.ID, item.Name)
	return map[string]any{"ok": true}, nil
}

func (c *LocalClient) ListAudit(ctx context.Context, limit int) (any, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := c.store.ListAudit(ctx, limit, c.userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func (c *LocalClient) GetEval(ctx context.Context, limit int) (any, error) {
	if limit <= 0 {
		limit = 200
	}
	report, err := eval.Run(ctx, c.store, limit, c.userID)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func compactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return true
	}
	return false
}

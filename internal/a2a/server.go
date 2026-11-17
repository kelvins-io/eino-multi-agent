package a2a

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/a2a/models"
	a2aserver "github.com/cloudwego/eino-ext/a2a/server"
	"github.com/cloudwego/eino-ext/a2a/transport/jsonrpc"
	"github.com/cloudwego/hertz/pkg/app"
	hertz_server "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/go-openapi/spec"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"go.uber.org/zap"
)

// Server 在独立 Hertz 端口上暴露 A2A JSON-RPC；入站 message 会创建 harness 任务。
type Server struct {
	cfg     *config.Config
	runtime *harness.Runtime
	store   *store.Store
	hz      *hertz_server.Hertz
}

func NewServer(cfg *config.Config, rt *harness.Runtime, st *store.Store) (*Server, error) {
	if cfg == nil || !cfg.A2A.Enabled {
		return nil, fmt.Errorf("a2a is disabled")
	}
	s := &Server{cfg: cfg, runtime: rt, store: st}
	hz := hertz_server.Default(
		hertz_server.WithHostPorts(cfg.A2A.Addr),
		hertz_server.WithSenseClientDisconnection(true),
	)
	cardPath := "/.well-known/agent-card.json"
	authMW := []app.HandlerFunc{bearerMiddleware(cfg.A2A.AuthToken)}
	reg, err := jsonrpc.NewRegistrar(context.Background(), &jsonrpc.ServerConfig{
		Router:               hz,
		HandlerPath:          cfg.A2A.HandlerPath,
		HandlerMiddleWares:   authMW,
		AgentCardPath:        &cardPath,
		AgentCardMiddleWares: authMW,
	})
	if err != nil {
		return nil, err
	}
	endpoint, err := url.JoinPath(strings.TrimRight(cfg.A2A.PublicURL, "/"), strings.TrimPrefix(cfg.A2A.HandlerPath, "/"))
	if err != nil {
		endpoint = strings.TrimRight(cfg.A2A.PublicURL, "/") + cfg.A2A.HandlerPath
	}
	skills := make([]models.AgentSkill, 0)
	for _, sk := range rt.Skills() {
		desc := sk.Description
		skills = append(skills, models.AgentSkill{
			ID:          sk.Name,
			Name:        sk.Name,
			Description: &desc,
			Tags:        []string{"skill"},
			InputModes:  []string{"text/plain"},
			OutputModes: []string{"text/plain"},
		})
	}
	securitySchemes := map[string]*spec.SecurityScheme{}
	var security []map[string][]string
	if strings.TrimSpace(cfg.A2A.AuthToken) != "" {
		// go-openapi/spec 为 OAS2：用 apiKey 描述 Authorization Bearer
		securitySchemes["bearer"] = spec.APIKeyAuth("Authorization", "header")
		security = []map[string][]string{{"bearer": {}}}
	}
	err = a2aserver.RegisterHandlers(context.Background(), reg, &a2aserver.Config{
		AgentCardConfig: a2aserver.AgentCardConfig{
			Name:               cfg.A2A.Name,
			Description:        cfg.A2A.Description,
			URL:                endpoint,
			Version:            cfg.A2A.Version,
			SecuritySchemes:    securitySchemes,
			Security:           security,
			DefaultInputModes:  []string{"text/plain"},
			DefaultOutputModes: []string{"text/plain"},
			Skills:             skills,
		},
		MessageStreamingHandler: s.handleStream,
		CancelTaskHandler:       s.handleCancel,
		TaskEventsConsolidator:  consolidateEvents,
		Logger: func(_ context.Context, format string, v ...any) {
			logx.Named("a2a").Sugar().Infof(format, v...)
		},
	})
	if err != nil {
		return nil, err
	}
	s.hz = hz
	return s, nil
}

// Start 拉起 A2A 端口；监听成功后执行 onReady（用于向注册中心登记）。
func (s *Server) Start(onReady func()) {
	log := logx.Named("a2a")
	s.hz.OnRun = append(s.hz.OnRun, func(ctx context.Context) error {
		log.Info("a2a listening",
			zap.String("addr", s.cfg.A2A.Addr),
			zap.String("handler", s.cfg.A2A.HandlerPath),
			zap.String("public_url", s.cfg.A2A.PublicURL),
			zap.String("agent_card", AgentCardURL(s.cfg.A2A)),
			zap.Bool("auth", strings.TrimSpace(s.cfg.A2A.AuthToken) != ""),
		)
		if onReady != nil {
			go onReady()
		}
		return nil
	})
	go func() {
		if err := s.hz.Run(); err != nil {
			log.Error("a2a server stopped", zap.Error(err))
		}
	}()
}

func (s *Server) handleStream(ctx context.Context, params *a2aserver.InputParams, writer a2aserver.ResponseEventWriter) error {
	taskID := params.Task.ID
	existing, err := s.store.GetTask(ctx, taskID)
	if err == nil && existing != nil {
		return s.watchExisting(ctx, existing, writer)
	}

	goal := textFromMessage(params.Input)
	if goal == "" {
		return fmt.Errorf("message text is required to create a harness task")
	}
	policy := s.cfg.A2A.ConfirmPolicy
	if v, ok := params.Metadata["confirm_policy"].(string); ok && strings.TrimSpace(v) != "" {
		policy = v
	}
	skills := skillsFromMetadata(params.Metadata)
	uploads := uploadsFromMessage(params.Input)

	_ = writer.Write(models.ResponseEvent{
		TaskStatusUpdateEventContent: &models.TaskStatusUpdateEventContent{
			Status: models.TaskStatus{
				State:     models.TaskStateSubmitted,
				Message:   agentMessage("已创建工作任务，准备执行"),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	})

	created, err := s.runtime.Create(ctx, harness.CreateInput{
		ID:            taskID,
		Title:         "A2A: " + truncate(goal, 36),
		Goal:          goal,
		ConfirmPolicy: policy,
		Skills:        skills,
		Files:         uploads,
	})
	if err != nil {
		return err
	}
	logx.Named("a2a").Info("inbound task created",
		zap.String("task_id", created.ID),
		zap.String("goal", truncate(goal, 80)),
	)
	return s.watchExisting(ctx, created, writer)
}

func (s *Server) watchExisting(ctx context.Context, task *store.Task, writer a2aserver.ResponseEventWriter) error {
	ch, unsub := s.runtime.Bus().Subscribe(task.ID)
	defer unsub()

	_ = writer.Write(models.ResponseEvent{
		TaskStatusUpdateEventContent: &models.TaskStatusUpdateEventContent{
			Status: models.TaskStatus{
				State:     mapStatus(task.Status),
				Message:   agentMessage("任务状态: " + task.Status),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	})

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-ch:
			if ok && strings.TrimSpace(ev.Message) != "" {
				_ = writer.Write(models.ResponseEvent{
					TaskStatusUpdateEventContent: &models.TaskStatusUpdateEventContent{
						Status: models.TaskStatus{
							State:     models.TaskStateWorking,
							Message:   agentMessage(ev.Message),
							Timestamp: time.Now().UTC().Format(time.RFC3339),
						},
					},
				})
			}
		case <-ticker.C:
		}
		cur, err := s.store.GetTask(ctx, task.ID)
		if err != nil {
			return err
		}
		if !isTerminal(cur.Status) {
			continue
		}
		arts, _ := s.store.ListArtifacts(ctx, cur.ID)
		content := taskContentFromHarness(cur, arts)
		_ = writer.Write(models.ResponseEvent{TaskContent: content})
		_ = writer.Write(models.ResponseEvent{
			TaskStatusUpdateEventContent: &models.TaskStatusUpdateEventContent{
				Status: content.Status,
				Final:  true,
			},
		})
		return nil
	}
}

func (s *Server) handleCancel(ctx context.Context, params *a2aserver.InputParams) (*models.TaskContent, error) {
	if err := s.runtime.Cancel(ctx, params.Task.ID); err != nil {
		// 若 harness 任务不存在，仍返回 canceled
		if !store.IsNotFound(err) {
			logx.Named("a2a").Warn("cancel harness task", zap.String("task_id", params.Task.ID), zap.Error(err))
		}
	}
	task, err := s.store.GetTask(ctx, params.Task.ID)
	if err != nil {
		return &models.TaskContent{
			Status: models.TaskStatus{State: models.TaskStateCanceled},
		}, nil
	}
	arts, _ := s.store.ListArtifacts(ctx, task.ID)
	return taskContentFromHarness(task, arts), nil
}

func consolidateEvents(_ context.Context, params *a2aserver.InputParams, events []models.ResponseEvent, _ error) *models.TaskContent {
	t := params.Task
	result := &models.TaskContent{
		Status:    t.Status,
		Artifacts: append([]*models.Artifact{}, t.Artifacts...),
		History:   append([]*models.Message{}, t.History...),
		Metadata:  t.Metadata,
	}
	if params.Input != nil {
		result.History = append(result.History, params.Input)
	}
	for _, e := range events {
		if e.TaskContent != nil {
			if result.Status.Message != nil {
				result.History = append(result.History, result.Status.Message)
			}
			result.Status = e.TaskContent.Status
			if e.TaskContent.Metadata != nil {
				result.Metadata = e.TaskContent.Metadata
			}
			result.Artifacts = append(result.Artifacts, e.TaskContent.Artifacts...)
			result.History = append(result.History, e.TaskContent.History...)
		} else if e.TaskStatusUpdateEventContent != nil {
			if result.Status.Message != nil {
				result.History = append(result.History, result.Status.Message)
			}
			result.Status = e.TaskStatusUpdateEventContent.Status
		} else if e.TaskArtifactUpdateEventContent != nil {
			if e.TaskArtifactUpdateEventContent.Append && len(result.Artifacts) > 0 {
				last := result.Artifacts[len(result.Artifacts)-1]
				last.Parts = append(last.Parts, e.TaskArtifactUpdateEventContent.Artifact.Parts...)
			} else {
				art := e.TaskArtifactUpdateEventContent.Artifact
				result.Artifacts = append(result.Artifacts, &art)
			}
		}
	}
	return result
}

func truncate(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

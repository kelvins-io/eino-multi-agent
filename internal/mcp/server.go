package mcp

import (
	"context"
	"net/http"
	"strings"

	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// StartHTTP 在独立端口上启动 Streamable HTTP MCP（与 A2A 类似）。
func StartHTTP(
	cfg *config.Config,
	st *store.Store,
	rt *harness.Runtime,
	sched *scheduler.Scheduler,
	connectors *connector.Registry,
) {
	if cfg == nil || !cfg.MCP.Enabled {
		return
	}
	path := cfg.MCP.Path
	if path == "" {
		path = "/mcp"
	}
	mux := http.NewServeMux()
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		userID, username, err := UserFromRequest(cfg, r)
		if err != nil {
			// Streamable handler 仍需要返回 Server；鉴权失败用空工具集 + 中间件拒绝更合适。
			// 这里返回带 auth 错误工具的 server 不可行；改为在外层包装。
			return NewServer(cfg.MCP.Name, cfg.MCP.Version, &authErrorClient{err: err})
		}
		local := NewLocalClient(cfg, st, rt, sched, connectors, userID, username)
		return NewServer(cfg.MCP.Name, cfg.MCP.Version, local)
	}, &mcpsdk.StreamableHTTPOptions{
		JSONResponse: true,
		Stateless:    true,
	})

	authWrap := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := UserFromRequest(cfg, r); err != nil {
			http.Error(w, `{"error":"`+escapeJSON(err.Error())+`"}`, http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})

	mux.Handle(path, authWrap)
	if !strings.HasSuffix(path, "/") {
		mux.Handle(path+"/", authWrap)
	}

	addr := cfg.MCP.Addr
	log := logx.Named("mcp")
	log.Info("mcp listening",
		zap.String("addr", addr),
		zap.String("path", path),
		zap.String("name", cfg.MCP.Name),
	)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Error("mcp server stopped", zap.Error(err))
		}
	}()
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// authErrorClient 所有调用返回鉴权错误（防御性；正常路径外层已 401）。
type authErrorClient struct{ err error }

func (c *authErrorClient) GetMeta(ctx context.Context) (any, error) { return nil, c.err }
func (c *authErrorClient) GetMe(ctx context.Context) (any, error)   { return nil, c.err }
func (c *authErrorClient) CreateTask(ctx context.Context, in CreateTaskInput) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ListTasks(ctx context.Context, limit int, projectID string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) GetTask(ctx context.Context, id string) (any, error) { return nil, c.err }
func (c *authErrorClient) CancelTask(ctx context.Context, id string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ConfirmTask(ctx context.Context, id string, approved bool) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) RetryTask(ctx context.Context, id string) (any, error) { return nil, c.err }
func (c *authErrorClient) ListTaskArtifacts(ctx context.Context, taskID string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) PreviewArtifact(ctx context.Context, taskID string, artifactID uint64) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ListProjects(ctx context.Context) (any, error) { return nil, c.err }
func (c *authErrorClient) CreateProject(ctx context.Context, name, description string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) GetProject(ctx context.Context, id string) (any, error) { return nil, c.err }
func (c *authErrorClient) UploadProjectFiles(ctx context.Context, projectID string, filePaths []string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ListSchedules(ctx context.Context) (any, error) { return nil, c.err }
func (c *authErrorClient) CreateSchedule(ctx context.Context, in CreateScheduleInput) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) RunSchedule(ctx context.Context, id string) (any, error) { return nil, c.err }
func (c *authErrorClient) ToggleSchedule(ctx context.Context, id string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ListConnectors(ctx context.Context) (any, error) { return nil, c.err }
func (c *authErrorClient) CreateConnector(ctx context.Context, name, kind string, config map[string]string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ToggleConnector(ctx context.Context, id string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) TestConnector(ctx context.Context, id string) (any, error) {
	return nil, c.err
}
func (c *authErrorClient) ListAudit(ctx context.Context, limit int) (any, error) { return nil, c.err }
func (c *authErrorClient) GetEval(ctx context.Context, limit int) (any, error)  { return nil, c.err }

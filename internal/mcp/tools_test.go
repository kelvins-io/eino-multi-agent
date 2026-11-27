package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeClient struct {
	meta any
	err  error
}

func (f *fakeClient) GetMeta(ctx context.Context) (any, error) { return f.meta, f.err }
func (f *fakeClient) GetMe(ctx context.Context) (any, error)   { return map[string]any{"id": "u1"}, f.err }
func (f *fakeClient) CreateTask(ctx context.Context, in CreateTaskInput) (any, error) {
	return map[string]any{"id": "t1", "goal": in.Goal}, f.err
}
func (f *fakeClient) ListTasks(ctx context.Context, limit int, projectID string) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) GetTask(ctx context.Context, id string) (any, error) {
	return map[string]any{"task": map[string]any{"id": id}}, f.err
}
func (f *fakeClient) CancelTask(ctx context.Context, id string) (any, error) {
	return map[string]any{"ok": true}, f.err
}
func (f *fakeClient) ConfirmTask(ctx context.Context, id string, approved bool) (any, error) {
	return map[string]any{"ok": true, "approved": approved}, f.err
}
func (f *fakeClient) RetryTask(ctx context.Context, id string) (any, error) {
	return map[string]any{"id": id}, f.err
}
func (f *fakeClient) ListTaskArtifacts(ctx context.Context, taskID string) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) PreviewArtifact(ctx context.Context, taskID string, artifactID uint64) (any, error) {
	return map[string]any{"previewable": true, "text": "hi"}, f.err
}
func (f *fakeClient) ListProjects(ctx context.Context) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) CreateProject(ctx context.Context, name, description string) (any, error) {
	return map[string]any{"name": name}, f.err
}
func (f *fakeClient) GetProject(ctx context.Context, id string) (any, error) {
	return map[string]any{"project": map[string]any{"id": id}}, f.err
}
func (f *fakeClient) UploadProjectFiles(ctx context.Context, projectID string, filePaths []string) (any, error) {
	return map[string]any{"files": filePaths}, f.err
}
func (f *fakeClient) ListSchedules(ctx context.Context) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) CreateSchedule(ctx context.Context, in CreateScheduleInput) (any, error) {
	return map[string]any{"goal": in.Goal}, f.err
}
func (f *fakeClient) RunSchedule(ctx context.Context, id string) (any, error) {
	return map[string]any{"id": id}, f.err
}
func (f *fakeClient) ToggleSchedule(ctx context.Context, id string) (any, error) {
	return map[string]any{"id": id}, f.err
}
func (f *fakeClient) ListConnectors(ctx context.Context) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) CreateConnector(ctx context.Context, name, kind string, config map[string]string) (any, error) {
	return map[string]any{"name": name, "kind": kind}, f.err
}
func (f *fakeClient) ToggleConnector(ctx context.Context, id string) (any, error) {
	return map[string]any{"id": id}, f.err
}
func (f *fakeClient) TestConnector(ctx context.Context, id string) (any, error) {
	return map[string]any{"ok": true}, f.err
}
func (f *fakeClient) ListAudit(ctx context.Context, limit int) (any, error) {
	return map[string]any{"items": []any{}}, f.err
}
func (f *fakeClient) GetEval(ctx context.Context, limit int) (any, error) {
	return map[string]any{"ok": true}, f.err
}

func TestRegisterToolsAndCallGetMeta(t *testing.T) {
	fake := &fakeClient{meta: map[string]any{"skills": []string{"weekly-report"}}}
	server := NewServer("test", "1.0.0", fake)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), t1, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(context.Background(), t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	tools, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 20 {
		t.Fatalf("expected full tool set, got %d", len(tools.Tools))
	}

	res, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "get_meta",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("want TextContent, got %T", res.Content[0])
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text.Text), &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["skills"]; !ok {
		t.Fatalf("missing skills in %v", parsed)
	}

	res2, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "create_task",
		Arguments: map[string]any{"goal": "写周报"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.IsError {
		t.Fatalf("create_task error: %+v", res2.Content)
	}
}

func TestUserFromRequest(t *testing.T) {
	cfg := config.Default()
	cfg.Server.JWTSecret = "test-secret"
	cfg.Server.AuthToken = "static-token"

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, err := tok.SignedString([]byte(cfg.Server.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	uid, uname, err := UserFromRequest(cfg, req)
	if err != nil || uid != "user-1" || uname != "alice" {
		t.Fatalf("jwt: uid=%s uname=%s err=%v", uid, uname, err)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	_, _, err = UserFromRequest(cfg, req2)
	if err == nil {
		t.Fatal("expected error without token")
	}

	req3 := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req3.Header.Set("Authorization", "Bearer static-token")
	_, _, err = UserFromRequest(cfg, req3)
	if err == nil {
		t.Fatal("expected rejection of static token")
	}
}

func TestHTTPAuthUnauthorized(t *testing.T) {
	cfg := config.Default()
	cfg.MCP.Enabled = true
	cfg.MCP.Addr = "127.0.0.1:0"
	// StartHTTP listens in background; exercise auth wrapper via ServeMux pattern.
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := UserFromRequest(cfg, r); err != nil {
			http.Error(w, `{"error":"login required"}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/mcp", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

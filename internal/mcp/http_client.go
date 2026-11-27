package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HTTPClient 通过 REST API 调用主服务（stdio MCP 使用）。
type HTTPClient struct {
	base   string
	token  string
	client *http.Client
}

func NewHTTPClient(baseURL, token string) *HTTPClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8180"
	}
	return &HTTPClient{
		base:  base,
		token: strings.TrimSpace(token),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *HTTPClient) GetMeta(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/v1/meta", nil)
}

func (c *HTTPClient) GetMe(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/v1/auth/me", nil)
}

func (c *HTTPClient) CreateTask(ctx context.Context, in CreateTaskInput) (any, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("goal", in.Goal)
	if in.Title != "" {
		_ = w.WriteField("title", in.Title)
	}
	if in.ConfirmPolicy != "" {
		_ = w.WriteField("confirm_policy", in.ConfirmPolicy)
	}
	if in.ProjectID != "" {
		_ = w.WriteField("project_id", in.ProjectID)
	}
	if len(in.Skills) > 0 {
		raw, _ := json.Marshal(in.Skills)
		_ = w.WriteField("skills", string(raw))
	}
	for _, p := range in.FilePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", p, err)
		}
		part, err := w.CreateFormFile("files", filepath.Base(p))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		_, err = io.Copy(part, f)
		_ = f.Close()
		if err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, "/api/v1/tasks", w.FormDataContentType(), body)
}

func (c *HTTPClient) ListTasks(ctx context.Context, limit int, projectID string) (any, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	return c.getJSON(ctx, "/api/v1/tasks", q)
}

func (c *HTTPClient) GetTask(ctx context.Context, id string) (any, error) {
	return c.getJSON(ctx, "/api/v1/tasks/"+url.PathEscape(id), nil)
}

func (c *HTTPClient) CancelTask(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/tasks/"+url.PathEscape(id)+"/cancel", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) ConfirmTask(ctx context.Context, id string, approved bool) (any, error) {
	payload, _ := json.Marshal(map[string]bool{"approved": approved})
	return c.do(ctx, http.MethodPost, "/api/v1/tasks/"+url.PathEscape(id)+"/confirm", "application/json", bytes.NewReader(payload))
}

func (c *HTTPClient) RetryTask(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/tasks/"+url.PathEscape(id)+"/retry", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) ListTaskArtifacts(ctx context.Context, taskID string) (any, error) {
	return c.getJSON(ctx, "/api/v1/tasks/"+url.PathEscape(taskID)+"/artifacts", nil)
}

func (c *HTTPClient) PreviewArtifact(ctx context.Context, taskID string, artifactID uint64) (any, error) {
	path := fmt.Sprintf("/api/v1/tasks/%s/artifacts/%d/preview", url.PathEscape(taskID), artifactID)
	return c.getJSON(ctx, path, nil)
}

func (c *HTTPClient) ListProjects(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/v1/projects", nil)
}

func (c *HTTPClient) CreateProject(ctx context.Context, name, description string) (any, error) {
	payload, _ := json.Marshal(map[string]string{"name": name, "description": description})
	return c.do(ctx, http.MethodPost, "/api/v1/projects", "application/json", bytes.NewReader(payload))
}

func (c *HTTPClient) GetProject(ctx context.Context, id string) (any, error) {
	return c.getJSON(ctx, "/api/v1/projects/"+url.PathEscape(id), nil)
}

func (c *HTTPClient) UploadProjectFiles(ctx context.Context, projectID string, filePaths []string) (any, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for _, p := range filePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", p, err)
		}
		part, err := w.CreateFormFile("files", filepath.Base(p))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		_, err = io.Copy(part, f)
		_ = f.Close()
		if err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	path := "/api/v1/projects/" + url.PathEscape(projectID) + "/files"
	return c.do(ctx, http.MethodPost, path, w.FormDataContentType(), body)
}

func (c *HTTPClient) ListSchedules(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/v1/schedules", nil)
}

func (c *HTTPClient) CreateSchedule(ctx context.Context, in CreateScheduleInput) (any, error) {
	payload, _ := json.Marshal(map[string]any{
		"project_id":     in.ProjectID,
		"title":          in.Title,
		"goal":           in.Goal,
		"skills":         in.Skills,
		"confirm_policy": in.ConfirmPolicy,
		"kind":           in.Kind,
		"weekday":        in.Weekday,
		"hour":           in.Hour,
		"minute":         in.Minute,
		"run_at":         in.RunAt,
		"timezone":       in.Timezone,
	})
	return c.do(ctx, http.MethodPost, "/api/v1/schedules", "application/json", bytes.NewReader(payload))
}

func (c *HTTPClient) RunSchedule(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/schedules/"+url.PathEscape(id)+"/run", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) ToggleSchedule(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/schedules/"+url.PathEscape(id)+"/toggle", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) ListConnectors(ctx context.Context) (any, error) {
	return c.getJSON(ctx, "/api/v1/connectors", nil)
}

func (c *HTTPClient) CreateConnector(ctx context.Context, name, kind string, config map[string]string) (any, error) {
	if config == nil {
		config = map[string]string{}
	}
	payload, _ := json.Marshal(map[string]any{"name": name, "kind": kind, "config": config})
	return c.do(ctx, http.MethodPost, "/api/v1/connectors", "application/json", bytes.NewReader(payload))
}

func (c *HTTPClient) ToggleConnector(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/connectors/"+url.PathEscape(id)+"/toggle", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) TestConnector(ctx context.Context, id string) (any, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/connectors/"+url.PathEscape(id)+"/test", "application/json", strings.NewReader("{}"))
}

func (c *HTTPClient) ListAudit(ctx context.Context, limit int) (any, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.getJSON(ctx, "/api/v1/audit", q)
}

func (c *HTTPClient) GetEval(ctx context.Context, limit int) (any, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.getJSON(ctx, "/api/v1/eval", q)
}

func (c *HTTPClient) getJSON(ctx context.Context, path string, q url.Values) (any, error) {
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	return c.do(ctx, http.MethodGet, path, "", nil)
}

func (c *HTTPClient) do(ctx context.Context, method, path, contentType string, body io.Reader) (any, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("api %s %s: %s", method, path, msg)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{"ok": true}, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return out, nil
}

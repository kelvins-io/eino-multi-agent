package harness

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

type projectedStep struct {
	Type             string
	Message          string
	Payload          string
	Todos            []store.TodoItem
	RefreshArtifacts bool
}

func projectMessage(msg *schema.Message) []projectedStep {
	if msg == nil {
		return nil
	}
	if len(msg.ToolCalls) > 0 {
		var steps []projectedStep
		for _, tc := range msg.ToolCalls {
			steps = append(steps, projectToolCall(tc.Function.Name, tc.Function.Arguments))
		}
		return steps
	}
	if msg.Role == schema.Tool {
		return nil
	}
	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return nil
	}
	return []projectedStep{{
		Type:    "think",
		Message: truncateRunes(text, 400),
	}}
}

func projectToolCall(name, args string) projectedStep {
	path := jsonString(args, "file_path", "path", "target")
	switch name {
	case "write_todos":
		todos := parseTodos(args)
		return projectedStep{
			Type:    "todos",
			Message: todoSummary(todos),
			Payload: args,
			Todos:   todos,
		}
	case "read_file", "ls", "glob", "grep":
		return projectedStep{Type: "read", Message: stepTarget("读取", name, path, args)}
	case "write_file", "edit_file":
		return projectedStep{
			Type:             "write",
			Message:          stepTarget("写入", name, path, args),
			RefreshArtifacts: true,
		}
	case "web_search", "fetch_url":
		q := jsonString(args, "query", "url")
		return projectedStep{Type: "search", Message: stepTarget("检索", name, q, args)}
	case "execute":
		return projectedStep{
			Type:             "shell",
			Message:          "执行 " + truncateRunes(jsonString(args, "command"), 160),
			RefreshArtifacts: true,
		}
	case "skill":
		return projectedStep{Type: "skill", Message: "加载技能 " + jsonString(args, "name")}
	case "confirm_action":
		return projectedStep{Type: "confirm", Message: "请求确认 " + jsonString(args, "action") + " " + jsonString(args, "target")}
	case "task":
		return projectedStep{Type: "think", Message: "调度子代理"}
	default:
		return projectedStep{Type: "think", Message: "调用 " + name}
	}
}

func parseTodos(args string) []store.TodoItem {
	var wrap struct {
		Todos []struct {
			Content    string `json:"content"`
			ActiveForm string `json:"activeForm"`
			Status     string `json:"status"`
		} `json:"todos"`
	}
	if err := json.Unmarshal([]byte(args), &wrap); err != nil {
		return nil
	}
	out := make([]store.TodoItem, 0, len(wrap.Todos))
	for _, t := range wrap.Todos {
		out = append(out, store.TodoItem{
			Content:    t.Content,
			ActiveForm: t.ActiveForm,
			Status:     t.Status,
		})
	}
	return out
}

func todoSummary(todos []store.TodoItem) string {
	if len(todos) == 0 {
		return "更新任务拆解"
	}
	done := 0
	current := ""
	for _, t := range todos {
		if t.Status == "completed" {
			done++
		}
		if current == "" && t.Status == "in_progress" {
			current = t.Content
		}
	}
	msg := "进度 " + strconv.Itoa(done) + "/" + strconv.Itoa(len(todos))
	if current != "" {
		msg += " · 正在" + current
	}
	return msg
}

func jsonString(raw string, keys ...string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func stepTarget(verb, name, target, args string) string {
	if target == "" {
		target = truncateRunes(args, 80)
	}
	if target == "" {
		return verb + " " + name
	}
	return verb + " " + filepath.ToSlash(target)
}

func Previewable(name, mime string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".md", ".txt", ".csv", ".json", ".yaml", ".yml", ".xml", ".log", ".html", ".htm",
		".py", ".go", ".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return true
	}
	return strings.HasPrefix(mime, "text/") || strings.HasPrefix(mime, "image/")
}

package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	ddg "github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/confirm"
)

type confirmInput struct {
	Action string `json:"action" jsonschema_description:"即将执行的操作，如 write/overwrite/delete/shell/http_post"`
	Target string `json:"target" jsonschema_description:"操作对象，通常是文件路径或 URL"`
	Reason string `json:"reason" jsonschema_description:"为什么需要执行该操作"`
}

func newConfirmTool(policy string) (tool.BaseTool, error) {
	policy = confirm.Normalize(policy)
	return utils.InferTool("confirm_action",
		"在执行可能覆盖文件、删除、运行危险命令或对外发送数据之前调用。若策略要求确认，将暂停等待用户批准。",
		func(ctx context.Context, in *confirmInput) (string, error) {
			if isResume, hasData, data := tool.GetResumeContext[bool](ctx); isResume && hasData {
				if data {
					return "用户已批准，请继续执行该操作。", nil
				}
				return "用户已拒绝，请停止该操作并给出替代方案。", nil
			}
			if !confirm.NeedsConfirm(policy, in.Action, in.Target) {
				return "当前确认策略允许自动执行，无需等待用户。", nil
			}
			msg := fmt.Sprintf("请求确认：%s %s。原因：%s", in.Action, in.Target, in.Reason)
			return "", tool.Interrupt(ctx, msg)
		})
}

type fetchInput struct {
	URL string `json:"url" jsonschema_description:"要读取的公开网页 URL"`
}

func newFetchTool() (tool.BaseTool, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	return utils.InferTool("fetch_url",
		"抓取公开网页的文本内容，用于调研。不要用于需要登录的页面。",
		func(ctx context.Context, in *fetchInput) (string, error) {
			if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
				return "", fmt.Errorf("only http/https urls are allowed")
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.URL, nil)
			if err != nil {
				return "", err
			}
			req.Header.Set("User-Agent", "eino-work-harness/0.1")
			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(io.LimitReader(resp.Body, 200_000))
			if err != nil {
				return "", err
			}
			text := string(body)
			if !utf8.ValidString(text) {
				text = strings.ToValidUTF8(text, "")
			}
			return fmt.Sprintf("status=%d\n%s", resp.StatusCode, trimRunes(text, 8000)), nil
		})
}

func extraTools(ctx context.Context, cfg *config.Config, policy string) ([]tool.BaseTool, error) {
	confirmTool, err := newConfirmTool(policy)
	if err != nil {
		return nil, err
	}
	fetchTool, err := newFetchTool()
	if err != nil {
		return nil, err
	}
	tools := []tool.BaseTool{confirmTool, fetchTool}
	if cfg.Search.Enabled {
		searchTool, err := ddg.NewTextSearchTool(ctx, &ddg.Config{
			ToolName:   "web_search",
			ToolDesc:   "搜索公开网页，返回标题、摘要和链接。",
			MaxResults: cfg.Search.MaxResults,
			Timeout:    20 * time.Second,
		})
		if err != nil {
			return nil, err
		}
		tools = append(tools, searchTool)
	}
	return tools, nil
}

func trimRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "\n...[truncated]"
}

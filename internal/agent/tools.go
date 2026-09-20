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
			action := confirm.Classify(in.Action, in.Target, false)
			if err := confirm.Gate(ctx, policy, action, in.Target); err != nil {
				return "", err
			}
			return "当前确认策略允许继续执行该操作。", nil
		})
}

type fetchInput struct {
	URL string `json:"url" jsonschema_description:"要读取的公开网页 URL"`
}

func newFetchTool() (tool.BaseTool, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	return utils.InferTool("fetch_url",
		"抓取公开网页并提取正文，只读采集，不要用于需要登录的页面。",
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
			if looksLikeHTML(resp.Header.Get("Content-Type"), text) {
				text = ReadableHTML(text)
			}
			return fmt.Sprintf("status=%d\n%s", resp.StatusCode, trimRunes(text, 8000)), nil
		})
}

type skillInput struct {
	Name string `json:"name" jsonschema_description:"要加载的技能名称，如 weekly-report、research、file-organize"`
}

func newSkillTool(skills []Skill) (tool.BaseTool, error) {
	index := make(map[string]Skill, len(skills))
	var names []string
	for _, s := range skills {
		index[s.Name] = s
		names = append(names, s.Name+": "+s.Description)
	}
	desc := "加载专业工作流程并按该技能执行。可用技能：\n" + strings.Join(names, "\n")
	return utils.InferTool("skill", desc, func(_ context.Context, in *skillInput) (string, error) {
		s, ok := index[strings.TrimSpace(in.Name)]
		if !ok {
			return "", fmt.Errorf("未知技能 %q，可用：%s", in.Name, strings.Join(skillNames(skills), ", "))
		}
		return fmt.Sprintf("# %s\n%s\n\n%s", s.Name, s.Description, s.Content), nil
	})
}

func skillNames(skills []Skill) []string {
	out := make([]string, 0, len(skills))
	for _, s := range skills {
		out = append(out, s.Name)
	}
	return out
}

func extraTools(ctx context.Context, cfg *config.Config, policy string, skills []Skill) ([]tool.BaseTool, error) {
	confirmTool, err := newConfirmTool(policy)
	if err != nil {
		return nil, err
	}
	fetchTool, err := newFetchTool()
	if err != nil {
		return nil, err
	}
	tools := []tool.BaseTool{confirmTool, fetchTool}
	if len(skills) > 0 {
		skillTool, err := newSkillTool(skills)
		if err != nil {
			return nil, err
		}
		tools = append(tools, skillTool)
	}
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

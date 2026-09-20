package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
)

type BuildRequest struct {
	Workspace *workspace.Sandbox
	Policy    string
	Skills    []Skill
}

type Factory struct {
	cfg   *config.Config
	model model.ToolCallingChatModel
}

func NewFactory(ctx context.Context, cfg *config.Config) (*Factory, error) {
	if cfg.Agent.Language == "zh" {
		_ = adk.SetLanguage(adk.LanguageChinese)
	}
	cm, err := NewChatModel(ctx, cfg.LLM)
	if err != nil {
		return nil, err
	}
	return &Factory{cfg: cfg, model: cm}, nil
}

func (f *Factory) Build(ctx context.Context, req BuildRequest) (adk.ResumableAgent, error) {
	backend, err := workspace.NewBackend(ctx, req.Workspace)
	if err != nil {
		return nil, err
	}
	tools, err := extraTools(ctx, f.cfg, req.Policy)
	if err != nil {
		return nil, err
	}

	research, err := deep.New(ctx, &deep.Config{
		Name:        "research",
		Description: "检索公开网页、阅读资料并整理研究结论",
		ChatModel:   f.model,
		Instruction: "你是调研子代理。使用 web_search 和 fetch_url 收集信息，把结论写进工作区 output/。使用绝对路径。",
		Backend:     backend,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools},
		},
		WithoutWriteTodos:      true,
		WithoutGeneralSubAgent: true,
		MaxIteration:           min(f.cfg.Agent.MaxIteration, 20),
		ModelRetryConfig:       f.retryConfig(),
		ModelFailoverConfig:    f.failoverConfig(),
	})
	if err != nil {
		return nil, fmt.Errorf("research agent: %w", err)
	}

	office, err := deep.New(ctx, &deep.Config{
		Name:                   "office",
		Description:            "用 Python 处理 CSV/Excel 并生成 Markdown、CSV、表格类交付物",
		ChatModel:              f.model,
		Instruction:            "你是办公产出子代理。优先用 python3 处理表格、生成图表数据和 Markdown。产物必须写到 output/。使用绝对路径。不要安装系统包。",
		Backend:                backend,
		StreamingShell:         backend,
		WithoutWriteTodos:      true,
		WithoutGeneralSubAgent: true,
		MaxIteration:           min(f.cfg.Agent.MaxIteration, 20),
		ModelRetryConfig:       f.retryConfig(),
		ModelFailoverConfig:    f.failoverConfig(),
	})
	if err != nil {
		return nil, fmt.Errorf("office agent: %w", err)
	}

	instruction := buildInstruction(req)
	return deep.New(ctx, &deep.Config{
		Name:           "work_harness",
		Description:    "长任务工作代理，拆解目标、调用工具并交付文件产物",
		ChatModel:      f.model,
		Instruction:    instruction,
		Backend:        backend,
		StreamingShell: backend,
		SubAgents:      []adk.Agent{research, office},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig:    compose.ToolsNodeConfig{Tools: tools},
			EmitInternalEvents: true,
		},
		MaxIteration:        f.cfg.Agent.MaxIteration,
		ModelRetryConfig:    f.retryConfig(),
		ModelFailoverConfig: f.failoverConfig(),
	})
}

func (f *Factory) retryConfig() *adk.ModelRetryConfig {
	return &adk.ModelRetryConfig{
		MaxRetries: 2,
		IsRetryAble: func(_ context.Context, err error) bool {
			return IsTransient(err)
		},
	}
}

func (f *Factory) failoverConfig() *adk.ModelFailoverConfig[*schema.Message] {
	models := f.cfg.LLM.FallbackModels
	if len(models) == 0 {
		return nil
	}
	return &adk.ModelFailoverConfig[*schema.Message]{
		MaxRetries: uint(len(models)),
		ShouldFailover: func(_ context.Context, _ *schema.Message, err error) bool {
			return IsQuotaOrRetryable(err)
		},
		GetFailoverModel: func(ctx context.Context, fc *adk.FailoverContext[*schema.Message]) (model.BaseModel[*schema.Message], []*schema.Message, error) {
			idx := int(fc.FailoverAttempt) - 1
			if idx < 0 || idx >= len(models) {
				return nil, nil, fmt.Errorf("no more fallback models")
			}
			next := f.cfg.LLM
			next.Model = models[idx]
			cm, err := NewChatModel(ctx, next)
			if err != nil {
				return nil, nil, err
			}
			return cm, nil, nil
		},
	}
}

func buildInstruction(req BuildRequest) string {
	var b strings.Builder
	b.WriteString("你是工作任务 Harness 中的主控代理，目标是完成用户交代的完整工作并交付可下载文件，而不是只聊天。\n")
	b.WriteString("工作区绝对路径：")
	b.WriteString(req.Workspace.Root)
	b.WriteString("\n目录约定：\n- input/ 用户上传的资料，只读优先\n- output/ 最终产物，必须写到这里\n- tmp/ 中间文件\n")
	b.WriteString("文件工具优先用相对路径：`.`、`input/`、`output/`。不要改写或猜测工作区的盘符与仓库路径。\n")
	b.WriteString("多步骤任务先用 write_todos 拆解，再逐步执行并更新进度。\n")
	b.WriteString("覆盖已有文件、删除、执行可能有副作用的 shell、对外发送数据前，必须先调用 confirm_action。\n")
	b.WriteString("调研交给 research 子代理，表格计算和文件生成交给 office 子代理。\n")
	b.WriteString("完成后在 output/ 留下成品，并用简体中文总结做了什么、产物路径和未完成项。\n")
	if len(req.Skills) > 0 {
		b.WriteString("\n已启用技能，按技能流程执行：\n")
		for _, s := range req.Skills {
			b.WriteString("\n## Skill: ")
			b.WriteString(s.Name)
			b.WriteString("\n")
			b.WriteString(s.Description)
			b.WriteString("\n")
			b.WriteString(s.Content)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

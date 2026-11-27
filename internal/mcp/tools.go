package mcp

import (
	"context"
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools 向 MCP server 注册全量业务工具。
func RegisterTools(server *mcpsdk.Server, client Client) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_meta", Description: "运行时元信息：模型、技能、确认策略"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.GetMeta(ctx))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_me", Description: "当前登录用户信息"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.GetMe(ctx))
		})

	type createTaskArgs struct {
		Goal          string   `json:"goal" jsonschema:"任务目标，必填"`
		Title         string   `json:"title,omitempty" jsonschema:"任务标题，可选"`
		ConfirmPolicy string   `json:"confirm_policy,omitempty" jsonschema:"确认策略：on_risk|always|never"`
		Skills        []string `json:"skills,omitempty" jsonschema:"技能名列表，如 weekly-report"`
		ProjectID     string   `json:"project_id,omitempty" jsonschema:"挂靠的项目 ID"`
		FilePaths     []string `json:"file_paths,omitempty" jsonschema:"本地文件路径，上传到任务 input/"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "create_task", Description: "创建并入队长任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in createTaskArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.CreateTask(ctx, CreateTaskInput{
				Title: in.Title, Goal: in.Goal, ConfirmPolicy: in.ConfirmPolicy,
				Skills: in.Skills, ProjectID: in.ProjectID, FilePaths: in.FilePaths,
			}))
		})

	type listTasksArgs struct {
		Limit     int    `json:"limit,omitempty" jsonschema:"返回条数，默认 50"`
		ProjectID string `json:"project_id,omitempty" jsonschema:"按项目过滤"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_tasks", Description: "列出当前用户的任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in listTasksArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListTasks(ctx, in.Limit, in.ProjectID))
		})

	type idArgs struct {
		ID string `json:"id" jsonschema:"资源 ID"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_task", Description: "任务详情，含事件与产物"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.GetTask(ctx, in.ID))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "cancel_task", Description: "取消任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.CancelTask(ctx, in.ID))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "retry_task", Description: "重试失败或已取消的任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.RetryTask(ctx, in.ID))
		})

	type confirmArgs struct {
		ID       string `json:"id" jsonschema:"任务 ID"`
		Approved bool   `json:"approved" jsonschema:"是否批准风险操作"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "confirm_task", Description: "批准或拒绝 waiting_confirm 任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in confirmArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ConfirmTask(ctx, in.ID, in.Approved))
		})

	type taskIDArgs struct {
		TaskID string `json:"task_id" jsonschema:"任务 ID"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_task_artifacts", Description: "列出任务产物"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in taskIDArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListTaskArtifacts(ctx, in.TaskID))
		})

	type previewArgs struct {
		TaskID     string `json:"task_id" jsonschema:"任务 ID"`
		ArtifactID uint64 `json:"artifact_id" jsonschema:"产物 ID"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "preview_artifact", Description: "预览文本产物内容（截断）"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in previewArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.PreviewArtifact(ctx, in.TaskID, in.ArtifactID))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_projects", Description: "列出项目"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListProjects(ctx))
		})

	type createProjectArgs struct {
		Name        string `json:"name" jsonschema:"项目名称"`
		Description string `json:"description,omitempty" jsonschema:"项目描述"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "create_project", Description: "创建项目空间"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in createProjectArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.CreateProject(ctx, in.Name, in.Description))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_project", Description: "项目详情、共享文件与任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.GetProject(ctx, in.ID))
		})

	type uploadProjectArgs struct {
		ProjectID string   `json:"project_id" jsonschema:"项目 ID"`
		FilePaths []string `json:"file_paths" jsonschema:"本地文件路径列表"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "upload_project_files", Description: "上传文件到项目共享目录"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in uploadProjectArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.UploadProjectFiles(ctx, in.ProjectID, in.FilePaths))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_schedules", Description: "列出定时任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListSchedules(ctx))
		})

	type createScheduleArgs struct {
		ProjectID     string   `json:"project_id,omitempty"`
		Title         string   `json:"title,omitempty"`
		Goal          string   `json:"goal" jsonschema:"定时任务目标"`
		Skills        []string `json:"skills,omitempty"`
		ConfirmPolicy string   `json:"confirm_policy,omitempty"`
		Kind          string   `json:"kind" jsonschema:"once|daily|weekly"`
		Weekday       int      `json:"weekday,omitempty" jsonschema:"weekly 时 0=周日 … 6=周六"`
		Hour          int      `json:"hour,omitempty"`
		Minute        int      `json:"minute,omitempty"`
		RunAt         string   `json:"run_at,omitempty" jsonschema:"once 时 RFC3339 时间"`
		Timezone      string   `json:"timezone,omitempty" jsonschema:"默认 Asia/Shanghai"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "create_schedule", Description: "创建定时任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in createScheduleArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.CreateSchedule(ctx, CreateScheduleInput{
				ProjectID: in.ProjectID, Title: in.Title, Goal: in.Goal, Skills: in.Skills,
				ConfirmPolicy: in.ConfirmPolicy, Kind: in.Kind, Weekday: in.Weekday,
				Hour: in.Hour, Minute: in.Minute, RunAt: in.RunAt, Timezone: in.Timezone,
			}))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "run_schedule", Description: "立刻跑一轮定时任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.RunSchedule(ctx, in.ID))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "toggle_schedule", Description: "启用/禁用定时任务"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ToggleSchedule(ctx, in.ID))
		})

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_connectors", Description: "列出连接器"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListConnectors(ctx))
		})

	type createConnectorArgs struct {
		Name   string            `json:"name" jsonschema:"连接器名称"`
		Kind   string            `json:"kind" jsonschema:"webhook|local_copy"`
		Config map[string]string `json:"config,omitempty" jsonschema:"连接器配置键值"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "create_connector", Description: "创建连接器"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in createConnectorArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.CreateConnector(ctx, in.Name, in.Kind, in.Config))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "toggle_connector", Description: "启用/禁用连接器"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ToggleConnector(ctx, in.ID))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "test_connector", Description: "测试连接器"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in idArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.TestConnector(ctx, in.ID))
		})

	type limitArgs struct {
		Limit int `json:"limit,omitempty" jsonschema:"返回条数"`
	}
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_audit", Description: "审计日志"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in limitArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.ListAudit(ctx, in.Limit))
		})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_eval", Description: "按内置用例评估历史任务完成率"},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in limitArgs) (*mcpsdk.CallToolResult, any, error) {
			return jsonResult(client.GetEval(ctx, in.Limit))
		})
}

func jsonResult(v any, err error) (*mcpsdk.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	raw, mErr := json.MarshalIndent(v, "", "  ")
	if mErr != nil {
		return nil, nil, mErr
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(raw)}},
	}, v, nil
}

// NewServer 构造已注册工具的 MCP Server。
func NewServer(name, version string, client Client) *mcpsdk.Server {
	if name == "" {
		name = "eino-work-harness"
	}
	if version == "" {
		version = "1.0.0"
	}
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: name, Version: version}, nil)
	RegisterTools(server, client)
	return server
}

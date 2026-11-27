package mcp

import "context"

// Client 是 MCP 工具背后的业务门面；stdio 用 HTTP 实现，主进程 HTTP MCP 用本地实现。
type Client interface {
	GetMeta(ctx context.Context) (any, error)
	GetMe(ctx context.Context) (any, error)

	CreateTask(ctx context.Context, in CreateTaskInput) (any, error)
	ListTasks(ctx context.Context, limit int, projectID string) (any, error)
	GetTask(ctx context.Context, id string) (any, error)
	CancelTask(ctx context.Context, id string) (any, error)
	ConfirmTask(ctx context.Context, id string, approved bool) (any, error)
	RetryTask(ctx context.Context, id string) (any, error)
	ListTaskArtifacts(ctx context.Context, taskID string) (any, error)
	PreviewArtifact(ctx context.Context, taskID string, artifactID uint64) (any, error)

	ListProjects(ctx context.Context) (any, error)
	CreateProject(ctx context.Context, name, description string) (any, error)
	GetProject(ctx context.Context, id string) (any, error)
	UploadProjectFiles(ctx context.Context, projectID string, filePaths []string) (any, error)

	ListSchedules(ctx context.Context) (any, error)
	CreateSchedule(ctx context.Context, in CreateScheduleInput) (any, error)
	RunSchedule(ctx context.Context, id string) (any, error)
	ToggleSchedule(ctx context.Context, id string) (any, error)

	ListConnectors(ctx context.Context) (any, error)
	CreateConnector(ctx context.Context, name, kind string, config map[string]string) (any, error)
	ToggleConnector(ctx context.Context, id string) (any, error)
	TestConnector(ctx context.Context, id string) (any, error)

	ListAudit(ctx context.Context, limit int) (any, error)
	GetEval(ctx context.Context, limit int) (any, error)
}

type CreateTaskInput struct {
	Title         string
	Goal          string
	ConfirmPolicy string
	Skills        []string
	ProjectID     string
	FilePaths     []string
}

type CreateScheduleInput struct {
	ProjectID     string
	Title         string
	Goal          string
	Skills        []string
	ConfirmPolicy string
	Kind          string
	Weekday       int
	Hour          int
	Minute        int
	RunAt         string
	Timezone      string
}

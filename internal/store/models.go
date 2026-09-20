package store

import "time"

const (
	StatusQueued         = "queued"
	StatusRunning        = "running"
	StatusWaitingConfirm = "waiting_confirm"
	StatusSucceeded      = "succeeded"
	StatusFailed         = "failed"
	StatusCancelled      = "cancelled"
)

type Task struct {
	ID            string     `gorm:"primaryKey;size:36" json:"id"`
	Title         string     `gorm:"size:255" json:"title"`
	Goal          string     `gorm:"type:text" json:"goal"`
	Status        string     `gorm:"size:32;index" json:"status"`
	ConfirmPolicy string     `gorm:"size:32" json:"confirm_policy"`
	Skills        []string   `gorm:"serializer:json;type:jsonb" json:"skills"`
	ProjectID     string     `gorm:"size:36;index" json:"project_id,omitempty"`
	ScheduleID    string     `gorm:"size:36;index" json:"schedule_id,omitempty"`
	Workspace     string     `gorm:"size:1024" json:"workspace"`
	Summary       string     `gorm:"type:text" json:"summary"`
	Todos         []TodoItem `gorm:"serializer:json;type:jsonb" json:"todos,omitempty"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	InterruptID   string     `gorm:"size:512" json:"interrupt_id,omitempty"`
	InterruptInfo string     `gorm:"type:text" json:"interrupt_info,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

func (Task) TableName() string { return "tasks" }

type TodoItem struct {
	Content    string `json:"content"`
	ActiveForm string `json:"active_form,omitempty"`
	Status     string `json:"status"`
}

type TaskEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"size:36;index" json:"task_id"`
	Type      string    `gorm:"size:64;index" json:"type"`
	Agent     string    `gorm:"size:128" json:"agent"`
	Message   string    `gorm:"type:text" json:"message"`
	Payload   string    `gorm:"type:text" json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (TaskEvent) TableName() string { return "task_events" }

type Artifact struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"size:36;index" json:"task_id"`
	RelPath   string    `gorm:"size:1024" json:"rel_path"`
	Name      string    `gorm:"size:255" json:"name"`
	Size      int64     `json:"size"`
	Mime      string    `gorm:"size:128" json:"mime"`
	CreatedAt time.Time `json:"created_at"`
}

func (Artifact) TableName() string { return "artifacts" }

type Checkpoint struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	Data      []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Checkpoint) TableName() string { return "checkpoints" }

type Project struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:255" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Workspace   string    `gorm:"size:1024" json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Project) TableName() string { return "projects" }

type Schedule struct {
	ID            string     `gorm:"primaryKey;size:36" json:"id"`
	ProjectID     string     `gorm:"size:36;index" json:"project_id,omitempty"`
	Title         string     `gorm:"size:255" json:"title"`
	Goal          string     `gorm:"type:text" json:"goal"`
	Skills        []string   `gorm:"serializer:json;type:jsonb" json:"skills"`
	ConfirmPolicy string     `gorm:"size:32" json:"confirm_policy"`
	Kind          string     `gorm:"size:16" json:"kind"`
	Weekday       int        `json:"weekday"`
	Hour          int        `json:"hour"`
	Minute        int        `json:"minute"`
	RunAt         *time.Time `json:"run_at,omitempty"`
	Timezone      string     `gorm:"size:64" json:"timezone"`
	Enabled       bool       `json:"enabled"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (Schedule) TableName() string { return "schedules" }

type Connector struct {
	ID        string            `gorm:"primaryKey;size:36" json:"id"`
	Name      string            `gorm:"size:255" json:"name"`
	Kind      string            `gorm:"size:32" json:"kind"`
	Config    map[string]string `gorm:"serializer:json;type:jsonb" json:"config"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (Connector) TableName() string { return "connectors" }

type AuditLog struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	Actor      string    `gorm:"size:128;index" json:"actor"`
	Action     string    `gorm:"size:64;index" json:"action"`
	TargetType string    `gorm:"size:32;index" json:"target_type"`
	TargetID   string    `gorm:"size:64;index" json:"target_id"`
	Detail     string    `gorm:"type:text" json:"detail,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

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
	Workspace     string     `gorm:"size:1024" json:"workspace"`
	Summary       string     `gorm:"type:text" json:"summary"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	InterruptID   string     `gorm:"size:512" json:"interrupt_id,omitempty"`
	InterruptInfo string     `gorm:"type:text" json:"interrupt_info,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

func (Task) TableName() string { return "tasks" }

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

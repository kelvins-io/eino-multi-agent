package a2a

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/a2a/models"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func textFromMessage(msg *models.Message) string {
	if msg == nil {
		return ""
	}
	var b strings.Builder
	for _, p := range msg.Parts {
		if p.Kind == models.PartKindText && p.Text != nil {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(*p.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

func uploadsFromMessage(msg *models.Message) []harness.Upload {
	if msg == nil {
		return nil
	}
	var out []harness.Upload
	for i, p := range msg.Parts {
		if p.Kind != models.PartKindFile || p.File == nil || p.File.Bytes == nil {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(*p.File.Bytes)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(*p.File.Bytes)
			if err != nil {
				continue
			}
		}
		name := strings.TrimSpace(p.File.Name)
		if name == "" {
			name = fmt.Sprintf("upload-%d.bin", i+1)
		}
		name = filepath.Base(name)
		data := raw
		out = append(out, harness.Upload{
			Name:   name,
			Reader: bytes.NewReader(data),
		})
	}
	return out
}

func skillsFromMetadata(md map[string]any) []string {
	if md == nil {
		return nil
	}
	raw, ok := md["skills"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return compactStrings(v)
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return compactStrings(out)
	case string:
		return compactStrings(strings.Split(v, ","))
	default:
		return nil
	}
}

func compactStrings(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func mapStatus(status string) models.TaskState {
	switch status {
	case store.StatusQueued:
		return models.TaskStateSubmitted
	case store.StatusRunning:
		return models.TaskStateWorking
	case store.StatusWaitingConfirm:
		return models.TaskStateInputRequired
	case store.StatusSucceeded:
		return models.TaskStateCompleted
	case store.StatusFailed:
		return models.TaskStateFailed
	case store.StatusCancelled:
		return models.TaskStateCanceled
	default:
		return models.TaskStateUnknown
	}
}

func agentMessage(text string) *models.Message {
	t := text
	return &models.Message{
		Role: models.RoleAgent,
		Parts: []models.Part{
			{Kind: models.PartKindText, Text: &t},
		},
	}
}

func taskContentFromHarness(task *store.Task, arts []store.Artifact) *models.TaskContent {
	state := mapStatus(task.Status)
	msg := task.Summary
	if msg == "" {
		msg = task.ErrorMessage
	}
	if msg == "" {
		switch task.Status {
		case store.StatusSucceeded:
			msg = "任务完成"
		case store.StatusFailed:
			msg = "任务失败"
		case store.StatusCancelled:
			msg = "任务已取消"
		case store.StatusWaitingConfirm:
			msg = task.InterruptInfo
			if msg == "" {
				msg = "等待确认"
			}
		default:
			msg = "任务进行中"
		}
	}
	content := &models.TaskContent{
		Status: models.TaskStatus{
			State:     state,
			Message:   agentMessage(msg),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Metadata: map[string]any{
			"harness_task_id": task.ID,
			"harness_status":  task.Status,
		},
	}
	for _, a := range arts {
		aid := fmt.Sprintf("%d", a.ID)
		name := a.Name
		desc := a.RelPath
		text := fmt.Sprintf("artifact path: %s (size=%d mime=%s)", a.RelPath, a.Size, a.Mime)
		content.Artifacts = append(content.Artifacts, &models.Artifact{
			ArtifactID:  aid,
			Name:        name,
			Description: desc,
			Parts: []models.Part{
				{Kind: models.PartKindText, Text: &text},
			},
			Metadata: map[string]any{
				"rel_path": a.RelPath,
				"size":     a.Size,
				"mime":     a.Mime,
			},
		})
	}
	return content
}

func isTerminal(status string) bool {
	return status == store.StatusSucceeded ||
		status == store.StatusFailed ||
		status == store.StatusCancelled ||
		status == store.StatusWaitingConfirm
}

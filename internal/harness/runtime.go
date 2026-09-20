package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/confirm"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
	"gorm.io/gorm"
)

type CreateInput struct {
	Title         string
	Goal          string
	ConfirmPolicy string
	Skills        []string
	Files         []Upload
}

type Upload struct {
	Name   string
	Reader interface{ Read([]byte) (int, error) }
}

type Runtime struct {
	cfg     *config.Config
	store   *store.Store
	cp      *store.CheckpointStore
	bus     *Bus
	factory *agent.Factory
	skills  []agent.Skill

	mu   sync.Mutex
	runs map[string]*liveRun
}

type liveRun struct {
	cancel context.CancelFunc
	adkFn  adk.AgentCancelFunc
}

func NewRuntime(cfg *config.Config, st *store.Store, factory *agent.Factory, skills []agent.Skill) *Runtime {
	return &Runtime{
		cfg:     cfg,
		store:   st,
		cp:      store.NewCheckpointStore(st),
		bus:     NewBus(),
		factory: factory,
		skills:  skills,
		runs:    map[string]*liveRun{},
	}
}

func (r *Runtime) Bus() *Bus { return r.bus }

func (r *Runtime) Skills() []agent.Skill { return r.skills }

func (r *Runtime) Create(ctx context.Context, in CreateInput) (*store.Task, error) {
	if strings.TrimSpace(in.Goal) == "" {
		return nil, fmt.Errorf("goal is required")
	}
	id := uuid.NewString()
	sb, err := workspace.New(r.cfg.Workspace.Root, id)
	if err != nil {
		return nil, err
	}
	for _, f := range in.Files {
		if _, err := sb.SaveUpload(f.Name, f.Reader); err != nil {
			return nil, err
		}
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = truncateRunes(strings.ReplaceAll(in.Goal, "\n", " "), 40)
	}
	task := &store.Task{
		ID:            id,
		Title:         title,
		Goal:          in.Goal,
		Status:        store.StatusQueued,
		ConfirmPolicy: confirm.Normalize(in.ConfirmPolicy),
		Skills:        in.Skills,
		Workspace:     sb.Root,
	}
	if err := r.store.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	r.emit(ctx, task.ID, "system", "", "任务已创建，准备执行", "")
	go r.start(task.ID, false, false)
	return task, nil
}

func (r *Runtime) Cancel(ctx context.Context, id string) error {
	task, err := r.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Status == store.StatusSucceeded || task.Status == store.StatusFailed || task.Status == store.StatusCancelled {
		return fmt.Errorf("task already finished")
	}
	r.mu.Lock()
	live := r.runs[id]
	r.mu.Unlock()
	if live != nil {
		if live.adkFn != nil {
			handle, _ := live.adkFn(adk.WithAgentCancelMode(adk.CancelImmediate))
			if handle != nil {
				_ = handle.Wait()
			}
		}
		if live.cancel != nil {
			live.cancel()
		}
	}
	now := time.Now()
	return r.store.UpdateStatus(ctx, id, store.StatusCancelled, func(t *store.Task) {
		t.FinishedAt = &now
		t.ErrorMessage = "cancelled by user"
	})
}

func (r *Runtime) Confirm(ctx context.Context, id string, approved bool) error {
	task, err := r.store.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Status != store.StatusWaitingConfirm {
		return fmt.Errorf("task is not waiting for confirmation")
	}
	r.emit(ctx, id, "confirm", "", map[bool]string{true: "用户已批准，继续执行", false: "用户已拒绝"}[approved], "")
	go r.start(id, true, approved)
	return nil
}

func CanRetry(status string) bool {
	return status == store.StatusFailed || status == store.StatusCancelled || status == store.StatusSucceeded
}

func (r *Runtime) Retry(ctx context.Context, id string) (*store.Task, error) {
	task, err := r.store.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanRetry(task.Status) {
		return nil, fmt.Errorf("当前状态 %s 不能重新执行", task.Status)
	}
	r.mu.Lock()
	if r.runs[id] != nil {
		r.mu.Unlock()
		return nil, fmt.Errorf("task is already running")
	}
	r.runs[id] = &liveRun{}
	r.mu.Unlock()
	if err := r.cp.Delete(ctx, id); err != nil {
		r.clearIdleRun(id)
		return nil, err
	}
	if err := r.store.UpdateStatus(ctx, id, store.StatusQueued, func(t *store.Task) {
		t.ErrorMessage = ""
		t.InterruptID = ""
		t.InterruptInfo = ""
		t.Summary = ""
		t.FinishedAt = nil
	}); err != nil {
		r.clearIdleRun(id)
		return nil, err
	}
	r.emit(ctx, id, "system", "", "任务重新执行", "")
	go r.start(id, false, false)
	return r.store.GetTask(ctx, id)
}

func (r *Runtime) clearIdleRun(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if live := r.runs[id]; live != nil && live.cancel == nil && live.adkFn == nil {
		delete(r.runs, id)
	}
}

func (r *Runtime) start(taskID string, resume bool, approved bool) {
	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Agent.RunTimeout)
	defer cancel()
	r.mu.Lock()
	r.runs[taskID] = &liveRun{cancel: cancel}
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.runs, taskID)
		r.mu.Unlock()
	}()

	task, err := r.store.GetTask(ctx, taskID)
	if err != nil {
		return
	}
	if err := agent.Ready(r.cfg.LLM); err != nil {
		r.fail(ctx, task, err)
		return
	}
	if r.factory == nil {
		factory, err := agent.NewFactory(ctx, r.cfg)
		if err != nil {
			r.fail(ctx, task, err)
			return
		}
		r.factory = factory
	}

	sb, err := workspace.Open(task.Workspace)
	if err != nil {
		r.fail(ctx, task, err)
		return
	}
	selected := agent.Select(r.skills, task.Skills)
	ag, err := r.factory.Build(ctx, agent.BuildRequest{
		Workspace: sb,
		Policy:    task.ConfirmPolicy,
		Skills:    selected,
	})
	if err != nil {
		r.fail(ctx, task, err)
		return
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           ag,
		EnableStreaming: true,
		CheckPointStore: r.cp,
	})
	cancelOpt, adkCancel := adk.WithCancel()
	r.mu.Lock()
	if live := r.runs[taskID]; live != nil {
		live.adkFn = adkCancel
	}
	r.mu.Unlock()

	now := time.Now()
	_ = r.store.UpdateStatus(ctx, taskID, store.StatusRunning, func(t *store.Task) {
		if t.StartedAt == nil {
			t.StartedAt = &now
		}
		t.InterruptID = ""
		t.InterruptInfo = ""
	})
	r.emit(ctx, taskID, "status", "", "任务开始执行", "")

	var iter *adk.AsyncIterator[*adk.AgentEvent]
	if resume {
		iter, err = runner.ResumeWithParams(ctx, taskID, &adk.ResumeParams{
			Targets: map[string]any{task.InterruptID: approved},
		}, cancelOpt)
		if err != nil {
			r.fail(ctx, task, err)
			return
		}
	} else {
		iter = runner.Query(ctx, buildUserPrompt(task, sb), cancelOpt, adk.WithCheckPointID(taskID))
	}

	interrupted := false
	var lastSummary string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			var cancelErr *adk.CancelError
			if errors.As(event.Err, &cancelErr) {
				_ = r.store.UpdateStatus(ctx, taskID, store.StatusCancelled, func(t *store.Task) {
					fin := time.Now()
					t.FinishedAt = &fin
					t.ErrorMessage = "cancelled"
				})
				r.emit(ctx, taskID, "status", "", "任务已取消", "")
				return
			}
			r.fail(ctx, task, event.Err)
			return
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			info := event.Action.Interrupted
			interruptID := ""
			msg := "等待用户确认"
			if len(info.InterruptContexts) > 0 {
				interruptID = info.InterruptContexts[0].ID
				if info.InterruptContexts[0].Info != nil {
					msg = fmt.Sprint(info.InterruptContexts[0].Info)
				}
			}
			_ = r.store.UpdateStatus(ctx, taskID, store.StatusWaitingConfirm, func(t *store.Task) {
				t.InterruptID = interruptID
				t.InterruptInfo = msg
			})
			r.emit(ctx, taskID, "interrupt", event.AgentName, msg, "")
			interrupted = true
			return
		}
		if text := projectEvent(ctx, r, taskID, event); text != "" {
			lastSummary = text
		}
	}
	if interrupted {
		return
	}
	if err := r.refreshArtifacts(ctx, taskID, sb); err != nil {
		r.fail(ctx, task, err)
		return
	}
	fin := time.Now()
	_ = r.store.UpdateStatus(ctx, taskID, store.StatusSucceeded, func(t *store.Task) {
		t.FinishedAt = &fin
		t.Summary = lastSummary
		t.ErrorMessage = ""
	})
	r.emit(ctx, taskID, "status", "", "任务完成", "")
}

func (r *Runtime) fail(ctx context.Context, task *store.Task, err error) {
	if task == nil {
		return
	}
	msg := agent.ExplainError(err)
	fin := time.Now()
	_ = r.store.UpdateStatus(ctx, task.ID, store.StatusFailed, func(t *store.Task) {
		t.FinishedAt = &fin
		t.ErrorMessage = msg
	})
	r.emit(ctx, task.ID, "error", "", msg, "")
}

func (r *Runtime) refreshArtifacts(ctx context.Context, taskID string, sb *workspace.Sandbox) error {
	files, err := sb.ScanOutput()
	if err != nil {
		return err
	}
	items := make([]store.Artifact, 0, len(files))
	for _, f := range files {
		items = append(items, store.Artifact{
			TaskID:  taskID,
			RelPath: f.RelPath,
			Name:    f.Name,
			Size:    f.Size,
			Mime:    f.Mime,
		})
	}
	return r.store.ReplaceArtifacts(ctx, taskID, items)
}

func (r *Runtime) emit(ctx context.Context, taskID, typ, agentName, message, payload string) {
	ev := &store.TaskEvent{
		TaskID:  taskID,
		Type:    typ,
		Agent:   agentName,
		Message: message,
		Payload: payload,
	}
	if err := r.store.AppendEvent(ctx, ev); err != nil {
		log.Printf("append event: %v", err)
		return
	}
	r.bus.Publish(Event{
		ID:      ev.ID,
		TaskID:  taskID,
		Type:    typ,
		Agent:   agentName,
		Message: message,
		Payload: payload,
	})
}

func projectEvent(ctx context.Context, r *Runtime, taskID string, event *adk.AgentEvent) string {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return ""
	}
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil || msg == nil {
		return ""
	}
	agentName := event.AgentName
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			args := tc.Function.Arguments
			r.emit(ctx, taskID, "tool_call", agentName, "调用 "+name, args)
			if name == "write_todos" {
				r.emit(ctx, taskID, "todos", agentName, "更新任务拆解", args)
			}
		}
	}
	if msg.Role == schema.Tool {
		r.emit(ctx, taskID, "tool_result", agentName, truncateRunes(msg.Content, 500), "")
		return ""
	}
	if strings.TrimSpace(msg.Content) != "" {
		r.emit(ctx, taskID, "assistant", agentName, msg.Content, "")
		return msg.Content
	}
	return ""
}

func buildUserPrompt(task *store.Task, sb *workspace.Sandbox) string {
	var b strings.Builder
	b.WriteString("请完成以下工作任务。\n\n目标：\n")
	b.WriteString(task.Goal)
	b.WriteString("\n\n确认策略：")
	b.WriteString(task.ConfirmPolicy)
	b.WriteString("\n工作区：")
	b.WriteString(sb.Root)
	inputs, _ := sb.ListInputs()
	if len(inputs) > 0 {
		b.WriteString("\n用户已上传文件：\n")
		for _, f := range inputs {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n请将最终可交付文件写入 output/ 目录。")
	return b.String()
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func Marshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

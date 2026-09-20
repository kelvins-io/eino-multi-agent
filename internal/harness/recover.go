package harness

import (
	"context"

	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"go.uber.org/zap"
)

const (
	RecoverSkip   = "skip"
	RecoverFresh  = "fresh"
	RecoverResume = "resume"
)

func PlanRecover(status string, hasCheckpoint bool) string {
	switch status {
	case store.StatusQueued:
		return RecoverFresh
	case store.StatusRunning:
		if hasCheckpoint {
			return RecoverResume
		}
		return RecoverFresh
	default:
		return RecoverSkip
	}
}

func (r *Runtime) Recover(ctx context.Context) {
	log := logx.Named("recover")
	items, err := r.store.ListInFlight(ctx)
	if err != nil {
		log.Error("list in-flight tasks", zap.Error(err))
		return
	}
	if len(items) == 0 {
		log.Info("no in-flight tasks to recover")
		return
	}
	log.Info("recovering in-flight tasks", zap.Int("count", len(items)))
	for i := range items {
		task := items[i]
		action := PlanRecover(task.Status, r.cp.Exists(ctx, task.ID))
		if action == RecoverSkip {
			continue
		}
		msg := "进程重启后继续执行"
		resume := false
		if action == RecoverResume {
			msg = "进程重启后从检查点续跑"
			resume = true
		}
		_ = r.store.CreateAudit(ctx, &store.AuditLog{
			UserID:     task.UserID,
			Actor:      "system",
			Action:     "task.recover",
			TargetType: "task",
			TargetID:   task.ID,
			Detail:     action,
		})
		log.Info("recover task",
			zap.String("task_id", task.ID),
			zap.String("user_id", task.UserID),
			zap.String("action", action),
			zap.String("status", task.Status),
		)
		r.emit(ctx, task.ID, "system", "", msg, "")
		go r.start(task.ID, resume, false)
	}
}

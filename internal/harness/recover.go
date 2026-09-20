package harness

import (
	"context"
	"log"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
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
	items, err := r.store.ListInFlight(ctx)
	if err != nil {
		log.Printf("recover list in-flight: %v", err)
		return
	}
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
		r.emit(ctx, task.ID, "system", "", msg, "")
		go r.start(task.ID, resume, false)
	}
}

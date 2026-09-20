package scheduler

import (
	"context"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/logx"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"go.uber.org/zap"
)

type Scheduler struct {
	store   *store.Store
	runtime *harness.Runtime
	tick    time.Duration
}

func New(st *store.Store, rt *harness.Runtime, tick time.Duration) *Scheduler {
	if tick <= 0 {
		tick = 20 * time.Second
	}
	return &Scheduler{store: st, runtime: rt, tick: tick}
}

func (s *Scheduler) Start(ctx context.Context) {
	logx.Named("scheduler").Info("started", zap.Duration("tick", s.tick))
	go func() {
		s.Tick(ctx)
		t := time.NewTicker(s.tick)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				logx.Named("scheduler").Info("stopped")
				return
			case <-t.C:
				s.Tick(ctx)
			}
		}
	}()
}

func (s *Scheduler) Tick(ctx context.Context) {
	now := time.Now()
	items, err := s.store.DueSchedules(ctx, now)
	if err != nil {
		logx.Named("scheduler").Error("list due schedules", zap.Error(err))
		return
	}
	for i := range items {
		item := items[i]
		if err := s.fire(ctx, &item, now); err != nil {
			logx.Named("scheduler").Error("fire schedule",
				zap.String("schedule_id", item.ID),
				zap.String("user_id", item.UserID),
				zap.Error(err),
			)
		}
	}
}

func (s *Scheduler) FireNow(ctx context.Context, id string) (*store.Task, error) {
	item, err := s.store.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.run(ctx, item, time.Now(), false)
}

func (s *Scheduler) fire(ctx context.Context, item *store.Schedule, now time.Time) error {
	_, err := s.run(ctx, item, now, true)
	return err
}

func (s *Scheduler) run(ctx context.Context, item *store.Schedule, now time.Time, advance bool) (*store.Task, error) {
	logx.Named("scheduler").Info("run schedule",
		zap.String("schedule_id", item.ID),
		zap.String("user_id", item.UserID),
		zap.String("title", item.Title),
		zap.Bool("advance", advance),
	)
	task, err := s.runtime.Create(ctx, harness.CreateInput{
		Title:         item.Title,
		Goal:          item.Goal,
		ConfirmPolicy: item.ConfirmPolicy,
		Skills:        item.Skills,
		ProjectID:     item.ProjectID,
		ScheduleID:    item.ID,
		UserID:        item.UserID,
	})
	if err != nil {
		return nil, err
	}
	loc := LoadLocation(item.Timezone)
	next, err := NextRun(now, loc, item.Kind, item.Weekday, item.Hour, item.Minute, item.RunAt)
	if err != nil {
		return task, err
	}
	item.LastRunAt = &now
	if item.Kind == KindOnce {
		item.Enabled = false
		item.NextRunAt = nil
	} else if advance {
		if !next.After(now) {
			next, _ = NextRun(now.Add(time.Second), loc, item.Kind, item.Weekday, item.Hour, item.Minute, item.RunAt)
		}
		item.NextRunAt = &next
	}
	if err := s.store.SaveSchedule(ctx, item); err != nil {
		return task, err
	}
	logx.Named("scheduler").Info("schedule spawned task",
		zap.String("schedule_id", item.ID),
		zap.String("task_id", task.ID),
	)
	return task, nil
}

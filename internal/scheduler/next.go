package scheduler

import (
	"fmt"
	"strings"
	"time"
)

const (
	KindOnce   = "once"
	KindDaily  = "daily"
	KindWeekly = "weekly"
)

func LoadLocation(name string) *time.Location {
	if strings.TrimSpace(name) == "" {
		name = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

func NextRun(now time.Time, loc *time.Location, kind string, weekday, hour, minute int, runAt *time.Time) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case KindOnce:
		if runAt == nil {
			return time.Time{}, fmt.Errorf("once schedule requires run_at")
		}
		return runAt.In(loc), nil
	case KindDaily:
		candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate, nil
	case KindWeekly:
		if weekday < 0 || weekday > 6 {
			return time.Time{}, fmt.Errorf("weekday must be 0-6")
		}
		candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		delta := (weekday - int(candidate.Weekday()) + 7) % 7
		candidate = candidate.AddDate(0, 0, delta)
		if !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		return candidate, nil
	default:
		return time.Time{}, fmt.Errorf("unknown schedule kind %s", kind)
	}
}

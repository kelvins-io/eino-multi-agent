package harness

import (
	"testing"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestPlanRecover(t *testing.T) {
	cases := []struct {
		status string
		hasCP  bool
		want   string
	}{
		{store.StatusQueued, false, RecoverFresh},
		{store.StatusQueued, true, RecoverFresh},
		{store.StatusRunning, true, RecoverResume},
		{store.StatusRunning, false, RecoverFresh},
		{store.StatusWaitingConfirm, true, RecoverSkip},
		{store.StatusSucceeded, false, RecoverSkip},
		{store.StatusFailed, false, RecoverSkip},
		{store.StatusCancelled, false, RecoverSkip},
	}
	for _, tc := range cases {
		got := PlanRecover(tc.status, tc.hasCP)
		if got != tc.want {
			t.Fatalf("status=%s cp=%v got %s want %s", tc.status, tc.hasCP, got, tc.want)
		}
	}
}

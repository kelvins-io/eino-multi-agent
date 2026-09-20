package harness

import (
	"testing"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestCanRetry(t *testing.T) {
	if !CanRetry(store.StatusFailed) || !CanRetry(store.StatusCancelled) || !CanRetry(store.StatusSucceeded) {
		t.Fatal("finished tasks should be retryable")
	}
	if CanRetry(store.StatusRunning) || CanRetry(store.StatusQueued) || CanRetry(store.StatusWaitingConfirm) {
		t.Fatal("in-flight tasks should not be retried")
	}
}

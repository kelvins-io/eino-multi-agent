package agent

import "testing"

func TestExplainErrorQuota(t *testing.T) {
	errMsg := `[NodeRunError] Error code: 429 - {"code":"SetLimitExceeded","message":"reached the set inference limit"}`
	got := ExplainError(errString(errMsg))
	if got == errMsg {
		t.Fatalf("expected humanized quota error, got raw: %s", got)
	}
	if !containsAny(got, "安全体验模式", "额度") {
		t.Fatalf("unexpected explain: %s", got)
	}
}

func TestExplainErrorTimeout(t *testing.T) {
	errMsg := "[NodeRunError] context deadline exceeded (Client.Timeout or context cancellation while reading body)"
	got := ExplainError(errString(errMsg))
	if !containsAny(got, "超时", "EINO_LLM_TIMEOUT") {
		t.Fatalf("unexpected explain: %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

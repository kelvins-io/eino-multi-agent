package a2a

import (
	"testing"

	"github.com/cloudwego/eino-ext/a2a/models"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestTextFromMessage(t *testing.T) {
	s1, s2 := "hello", "world"
	msg := &models.Message{
		Parts: []models.Part{
			{Kind: models.PartKindText, Text: &s1},
			{Kind: models.PartKindData, Data: map[string]any{"x": 1}},
			{Kind: models.PartKindText, Text: &s2},
		},
	}
	if got := textFromMessage(msg); got != "hello\nworld" {
		t.Fatalf("got %q", got)
	}
}

func TestMapStatus(t *testing.T) {
	cases := map[string]models.TaskState{
		store.StatusQueued:         models.TaskStateSubmitted,
		store.StatusRunning:        models.TaskStateWorking,
		store.StatusWaitingConfirm: models.TaskStateInputRequired,
		store.StatusSucceeded:      models.TaskStateCompleted,
		store.StatusFailed:         models.TaskStateFailed,
		store.StatusCancelled:      models.TaskStateCanceled,
	}
	for in, want := range cases {
		if got := mapStatus(in); got != want {
			t.Fatalf("%s -> %s, want %s", in, got, want)
		}
	}
}

func TestSkillsFromMetadata(t *testing.T) {
	got := skillsFromMetadata(map[string]any{"skills": []any{"weekly-report", " research "}})
	if len(got) != 2 || got[0] != "weekly-report" || got[1] != "research" {
		t.Fatalf("got %#v", got)
	}
}

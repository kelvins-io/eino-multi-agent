package eval

import (
	"testing"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestScoreWeeklyReport(t *testing.T) {
	c := Catalog()[0]
	now := time.Now()
	start := now.Add(-2 * time.Minute)
	task := store.Task{
		ID:         "t1",
		Title:      "周报",
		Status:     store.StatusSucceeded,
		Skills:     []string{"weekly-report"},
		StartedAt:  &start,
		FinishedAt: &now,
	}
	arts := []store.Artifact{{Name: "weekly-report.md", RelPath: "output/weekly-report.md"}}
	events := []store.TaskEvent{{Type: "interrupt"}, {Type: "confirm"}}
	got := Score(c, task, events, arts)
	if !got.Passed || got.Confirms != 1 || got.DurationMS <= 0 {
		t.Fatalf("%+v", got)
	}
}

func TestScoreMissingArtifact(t *testing.T) {
	c := Catalog()[0]
	task := store.Task{ID: "t2", Status: store.StatusSucceeded, Skills: []string{"weekly-report"}}
	got := Score(c, task, nil, nil)
	if got.Passed {
		t.Fatal("expected fail")
	}
}

func TestScoreNotSucceeded(t *testing.T) {
	c := Catalog()[1]
	task := store.Task{ID: "t3", Status: store.StatusFailed, Skills: []string{"research"}}
	got := Score(c, task, nil, []store.Artifact{{Name: "research-report.md"}})
	if got.Passed {
		t.Fatal("failed task should not pass")
	}
}

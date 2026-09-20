package scheduler

import (
	"testing"
	"time"
)

func TestNextRunDaily(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 20, 19, 0, 0, 0, loc)
	got, err := NextRun(now, loc, KindDaily, 0, 18, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 21, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestNextRunWeeklyFriday(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	// Sunday Sep 20 2026 12:00 -> next Friday 18:00 is Sep 25
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, loc)
	got, err := NextRun(now, loc, KindWeekly, int(time.Friday), 18, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 25, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s (%s) want %s", got, got.Weekday(), want)
	}
}

func TestNextRunWeeklyAlreadyPassedThisWeek(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 25, 19, 0, 0, 0, loc) // Friday 19:00
	got, err := NextRun(now, loc, KindWeekly, int(time.Friday), 18, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 2, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

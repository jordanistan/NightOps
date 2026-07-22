package domain

import (
	"testing"
	"time"
)

func TestMissionLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	mission, err := NewMission("m-1", "First Light", "site-1", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []MissionStatus{StatusPlanned, StatusLaunched, StatusActive, StatusCompleted, StatusArchived} {
		if err := mission.Transition(status, now); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	if mission.CompletedAt == nil {
		t.Fatal("completed mission has no completion timestamp")
	}
}

func TestMissionRejectsIllegalTransition(t *testing.T) {
	now := time.Now()
	mission, _ := NewMission("m-1", "First Light", "site-1", now)
	if err := mission.Transition(StatusCompleted, now); err == nil {
		t.Fatal("expected illegal transition error")
	}
}

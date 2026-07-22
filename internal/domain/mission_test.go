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

func TestNewIDIsUniqueShape(t *testing.T) {
	first, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || len(first) != 32 || len(second) != 32 {
		t.Fatalf("unexpected IDs: %q %q", first, second)
	}
}

func TestMissionRejectsIllegalTransition(t *testing.T) {
	now := time.Now()
	mission, _ := NewMission("m-1", "First Light", "site-1", now)
	if err := mission.Transition(StatusCompleted, now); err == nil {
		t.Fatal("expected illegal transition error")
	}
}

func TestMissionScheduleRequiresOrderedWindow(t *testing.T) {
	now := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	mission, _ := NewMission("m-1", "First Light", "site-1", now)
	if err := mission.SetSchedule(now.Add(time.Hour), now); err == nil {
		t.Fatal("expected reversed mission window error")
	}
	start, end := now.Add(time.Hour), now.Add(3*time.Hour)
	if err := mission.SetSchedule(start, end); err != nil {
		t.Fatal(err)
	}
	if mission.PlannedStart == nil || !mission.PlannedStart.Equal(start) || mission.PlannedEnd == nil || !mission.PlannedEnd.Equal(end) {
		t.Fatalf("mission schedule was not stored: %+v", mission)
	}
}

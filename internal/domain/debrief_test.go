package domain

import (
	"testing"
	"time"
)

func TestNewDebriefRequiresSummary(t *testing.T) {
	if _, err := NewDebrief("d-1", "m-1", "  ", time.Now()); err == nil {
		t.Fatal("empty debrief summary was accepted")
	}
}

func TestNewDebriefTrimsSummary(t *testing.T) {
	debrief, err := NewDebrief("d-1", "m-1", "  clear skies  ", time.Now())
	if err != nil || debrief.Summary != "clear skies" {
		t.Fatalf("debrief=%+v err=%v", debrief, err)
	}
}

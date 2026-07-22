package domain

import (
	"errors"
	"strings"
	"time"
)

// Debrief is the user-authored record that closes an observing operation.
type Debrief struct {
	ID        string
	MissionID string
	Summary   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDebrief creates a validated debrief for a completed mission.
func NewDebrief(id, missionID, summary string, now time.Time) (Debrief, error) {
	if id == "" || missionID == "" {
		return Debrief{}, errors.New("debrief id and mission id are required")
	}
	if strings.TrimSpace(summary) == "" {
		return Debrief{}, errors.New("debrief summary is required")
	}
	return Debrief{ID: id, MissionID: missionID, Summary: strings.TrimSpace(summary), CreatedAt: now, UpdatedAt: now}, nil
}

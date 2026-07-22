// Package domain contains NightOps business concepts and invariants.
package domain

import (
	"errors"
	"time"
)

// MissionStatus describes the lifecycle stage of an observing mission.
type MissionStatus string

const (
	StatusDraft     MissionStatus = "draft"
	StatusPlanned   MissionStatus = "planned"
	StatusLaunched  MissionStatus = "launched"
	StatusActive    MissionStatus = "active"
	StatusPaused    MissionStatus = "paused"
	StatusCompleted MissionStatus = "completed"
	StatusCancelled MissionStatus = "cancelled"
	StatusArchived  MissionStatus = "archived"
)

// LaunchSite is a named observing location with explicit coordinate provenance.
type LaunchSite struct {
	ID        string
	Name      string
	Latitude  *float64
	Longitude *float64
	Timezone  string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Mission is the durable aggregate around an observing operation.
type Mission struct {
	ID                 string
	Name               string
	Status             MissionStatus
	LaunchSiteID       string
	EquipmentProfileID string
	PlannedStart       *time.Time
	PlannedEnd         *time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewMission creates a mission in the draft state.
func NewMission(id, name, launchSiteID string, now time.Time) (Mission, error) {
	if id == "" || name == "" || launchSiteID == "" {
		return Mission{}, errors.New("mission id, name, and launch site are required")
	}
	return Mission{ID: id, Name: name, Status: StatusDraft, LaunchSiteID: launchSiteID, CreatedAt: now, UpdatedAt: now}, nil
}

// SetSchedule assigns a complete observing window to a mission.
func (m *Mission) SetSchedule(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return errors.New("mission schedule requires start and end")
	}
	if !end.After(start) {
		return errors.New("mission schedule must end after it starts")
	}
	start, end = start.UTC(), end.UTC()
	m.PlannedStart, m.PlannedEnd = &start, &end
	return nil
}

// Transition moves a mission through a legal lifecycle transition.
func (m *Mission) Transition(next MissionStatus, now time.Time) error {
	legal := map[MissionStatus][]MissionStatus{
		StatusDraft: {StatusPlanned, StatusCancelled}, StatusPlanned: {StatusLaunched, StatusCancelled},
		StatusLaunched: {StatusActive, StatusCancelled}, StatusActive: {StatusPaused, StatusCompleted, StatusCancelled},
		StatusPaused: {StatusActive, StatusCancelled}, StatusCompleted: {StatusArchived}, StatusCancelled: {StatusArchived},
	}
	for _, candidate := range legal[m.Status] {
		if candidate == next {
			m.Status, m.UpdatedAt = next, now
			if next == StatusLaunched || next == StatusActive {
				m.StartedAt = &now
			}
			if next == StatusCompleted {
				m.CompletedAt = &now
			}
			return nil
		}
	}
	return errors.New("illegal mission transition from " + string(m.Status) + " to " + string(next))
}

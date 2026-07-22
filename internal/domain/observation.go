package domain

import (
	"errors"
	"time"
)

// Observation is a durable record made during an active operation.
type Observation struct {
	ID         string
	MissionID  string
	TargetName string
	Notes      string
	ObservedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewObservation creates an observation with a required target name.
func NewObservation(id, missionID, targetName, notes string, now time.Time) (Observation, error) {
	if id == "" || missionID == "" || targetName == "" {
		return Observation{}, errors.New("observation ID, mission ID, and target name are required")
	}
	return Observation{ID: id, MissionID: missionID, TargetName: targetName, Notes: notes, ObservedAt: &now, CreatedAt: now, UpdatedAt: now}, nil
}

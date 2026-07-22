package domain

import (
	"fmt"
	"strings"
	"time"
)

// MissionTarget is a catalog target selected for one observing mission.
// Coordinates and provenance are copied at selection time so a later catalog
// update cannot silently change a planned mission.
type MissionTarget struct {
	ID             string
	MissionID      string
	Name           string
	Kind           string
	RightAscension float64
	Declination    float64
	Source         string
	Position       int
	CreatedAt      time.Time
}

// TargetKnowledge is cached, source-attributed reference material for a
// celestial target. It is deliberately separate from the catalog snapshot so
// a provider update cannot change a historical mission plan.
type TargetKnowledge struct {
	TargetID   string
	TargetName string
	Title      string
	URL        string
	Summary    string
	ImageURL   string
	Source     string
	Status     string
	FetchedAt  time.Time
}

// Validate checks the stable identity and provenance of cached target knowledge.
func (k TargetKnowledge) Validate() error {
	if strings.TrimSpace(k.TargetID) == "" || strings.TrimSpace(k.TargetName) == "" {
		return fmt.Errorf("target knowledge requires target id and name")
	}
	if strings.TrimSpace(k.Status) == "" {
		return fmt.Errorf("target knowledge requires a status")
	}
	if !k.FetchedAt.IsZero() && strings.TrimSpace(k.Source) == "" {
		return fmt.Errorf("fetched target knowledge requires a source")
	}
	return nil
}

// Validate checks the stable identity and selection order of a mission target.
func (t MissionTarget) Validate() error {
	if strings.TrimSpace(t.ID) == "" || strings.TrimSpace(t.MissionID) == "" || strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("mission target requires id, mission id, and name")
	}
	if t.Position < 0 {
		return fmt.Errorf("mission target position cannot be negative")
	}
	if t.CreatedAt.IsZero() {
		return fmt.Errorf("mission target creation time is required")
	}
	return nil
}

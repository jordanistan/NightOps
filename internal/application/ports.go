// Package application defines use-case boundaries between NightOps domains and adapters.
package application

import (
	"context"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/projection"
)

// Clock makes time-dependent use cases deterministic in tests.
type Clock interface{ Now() time.Time }

// MissionRepository persists mission aggregates.
type MissionRepository interface {
	SaveMission(context.Context, domain.Mission) error
	FindMission(context.Context, string) (domain.Mission, error)
}

// MissionTargetRepository persists the selected target sequence for a mission.
type MissionTargetRepository interface {
	ReplaceMissionTargets(context.Context, string, []domain.MissionTarget) error
	ListMissionTargets(context.Context, string) ([]domain.MissionTarget, error)
}

// TargetKnowledgeRepository persists source-attributed target reference data.
type TargetKnowledgeRepository interface {
	SaveTargetKnowledge(context.Context, domain.TargetKnowledge) error
	FindTargetKnowledge(context.Context, string) (domain.TargetKnowledge, error)
}

// TargetKnowledgeProvider retrieves live reference data for a selected target.
type TargetKnowledgeProvider interface {
	Fetch(context.Context, domain.MissionTarget) (domain.TargetKnowledge, error)
}

// MissionProjection contains the live or cached facts rendered into a vault.
// RichMissionExporter receives the complete mission projection when the
// adapter supports detailed knowledge notes.
type RichMissionExporter interface {
	ExportMissionProjection(context.Context, domain.Mission, domain.LaunchSite, projection.Mission) error
}

// LaunchSiteRepository persists and queries observing locations.
type LaunchSiteRepository interface {
	SaveLaunchSite(context.Context, domain.LaunchSite) error
	FindLaunchSite(context.Context, string) (domain.LaunchSite, error)
}

// EquipmentRepository persists reusable observing setups.
type EquipmentRepository interface {
	SaveEquipmentProfile(context.Context, domain.EquipmentProfile) error
	FindEquipmentProfile(context.Context, string) (domain.EquipmentProfile, error)
	ListEquipmentProfiles(context.Context) ([]domain.EquipmentProfile, error)
	SaveEquipmentItem(context.Context, domain.EquipmentItem) error
	ListEquipmentItems(context.Context, string) ([]domain.EquipmentItem, error)
}

// MissionExporter projects a mission into an external knowledge system.
type MissionExporter interface {
	Export(context.Context, domain.Mission, domain.LaunchSite) error
	ExportObservation(context.Context, domain.Mission, domain.LaunchSite, domain.Observation) error
	ExportDebrief(context.Context, domain.Mission, domain.LaunchSite, domain.Debrief) error
	ExportMissionTarget(context.Context, domain.Mission, domain.LaunchSite, domain.MissionTarget) error
	ExportMissionEquipment(context.Context, domain.Mission, domain.LaunchSite, domain.EquipmentProfile, []domain.EquipmentItem) error
}

// ObservationRepository persists records made during an operation.
type ObservationRepository interface {
	SaveObservation(context.Context, domain.Observation) error
	ListObservations(context.Context, string) ([]domain.Observation, error)
}

// DebriefRepository persists the user-authored closeout of an operation.
type DebriefRepository interface {
	SaveDebrief(context.Context, domain.Debrief) error
	FindDebrief(context.Context, string) (domain.Debrief, error)
}

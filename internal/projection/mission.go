// Package projection defines rich, adapter-facing mission knowledge views.
package projection

import (
	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/weather"
)

// Mission contains live or cached facts rendered into a knowledge vault.
type Mission struct {
	Weather         *weather.Snapshot
	Targets         []domain.MissionTarget
	TargetKnowledge map[string]domain.TargetKnowledge
	Equipment       *domain.EquipmentProfile
	EquipmentItems  []domain.EquipmentItem
	Observations    []domain.Observation
}

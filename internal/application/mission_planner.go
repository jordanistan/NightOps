package application

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/projection"
	"github.com/jordanistan/nightops/internal/weather"
)

// PlanInput contains the user-confirmed origin for a new mission.
type PlanInput struct {
	Name               string
	Kind               string
	Label              string
	ZIP                string
	Latitude           *float64
	Longitude          *float64
	Timezone           string
	EquipmentProfileID string
	PlannedStart       *time.Time
	PlannedEnd         *time.Time
}

// MissionPlanner creates durable mission and launch-site aggregates.
type MissionPlanner struct {
	Sites             LaunchSiteRepository
	Missions          MissionRepository
	MissionTargets    MissionTargetRepository
	TargetKnowledge   TargetKnowledgeRepository
	KnowledgeProvider TargetKnowledgeProvider
	Weather           weather.Cache
	Observations      ObservationRepository
	Debriefs          DebriefRepository
	Equipment         EquipmentRepository
	Exporter          MissionExporter
	Clock             Clock
}

// SetMissionTargets stores the observer's ordered target plan and projects
// each selected object into the mission's linked knowledge graph.
func (p MissionPlanner) SetMissionTargets(ctx context.Context, missionID string, targets []domain.MissionTarget) error {
	if p.MissionTargets == nil {
		return fmt.Errorf("mission target storage is unavailable")
	}
	mission, err := p.Missions.FindMission(ctx, missionID)
	if err != nil {
		return err
	}
	if err := p.MissionTargets.ReplaceMissionTargets(ctx, missionID, targets); err != nil {
		return err
	}
	if p.Exporter == nil {
		return nil
	}
	site, err := p.Sites.FindLaunchSite(ctx, mission.LaunchSiteID)
	if err != nil {
		return err
	}
	knowledge := make(map[string]domain.TargetKnowledge, len(targets))
	for _, target := range targets {
		knowledge[target.ID] = p.resolveTargetKnowledge(ctx, target)
	}
	if rich, ok := p.Exporter.(RichMissionExporter); ok {
		return rich.ExportMissionProjection(ctx, mission, site, p.projection(ctx, mission, site, targets, knowledge))
	}
	for _, target := range targets {
		if err := p.Exporter.ExportMissionTarget(ctx, mission, site, target); err != nil {
			return err
		}
	}
	return nil
}

// StartMission transitions a planned mission into the launched state.
func (p MissionPlanner) StartMission(ctx context.Context, missionID string) (domain.Mission, error) {
	return p.transitionMission(ctx, missionID, domain.StatusLaunched)
}

// ActivateMission transitions a launched mission into active observing.
func (p MissionPlanner) ActivateMission(ctx context.Context, missionID string) (domain.Mission, error) {
	return p.transitionMission(ctx, missionID, domain.StatusActive)
}

// CompleteMission transitions an active mission into its debrief state.
func (p MissionPlanner) CompleteMission(ctx context.Context, missionID string) (domain.Mission, error) {
	return p.transitionMission(ctx, missionID, domain.StatusCompleted)
}

// RecordDebrief persists the user-authored closeout for a completed mission.
func (p MissionPlanner) RecordDebrief(ctx context.Context, missionID, summary string) (domain.Debrief, error) {
	if p.Debriefs == nil {
		return domain.Debrief{}, fmt.Errorf("debrief storage is unavailable")
	}
	mission, err := p.Missions.FindMission(ctx, missionID)
	if err != nil {
		return domain.Debrief{}, err
	}
	if mission.Status != domain.StatusCompleted && mission.Status != domain.StatusArchived {
		return domain.Debrief{}, fmt.Errorf("mission is not completed")
	}
	now := p.Clock.Now().UTC()
	id, err := domain.NewID()
	if err != nil {
		return domain.Debrief{}, err
	}
	debrief, err := domain.NewDebrief(id, missionID, summary, now)
	if err != nil {
		return domain.Debrief{}, err
	}
	if err := p.Debriefs.SaveDebrief(ctx, debrief); err != nil {
		return domain.Debrief{}, err
	}
	if err := p.exportMission(ctx, mission); err != nil {
		return domain.Debrief{}, err
	}
	if p.Exporter != nil {
		site, findErr := p.Sites.FindLaunchSite(ctx, mission.LaunchSiteID)
		if findErr != nil {
			return domain.Debrief{}, findErr
		}
		if exportErr := p.Exporter.ExportDebrief(ctx, mission, site, debrief); exportErr != nil {
			return domain.Debrief{}, exportErr
		}
	}
	return debrief, nil
}

// RecordObservation persists an observation for an active or launched mission.
func (p MissionPlanner) RecordObservation(ctx context.Context, missionID, targetName, notes string) (domain.Observation, error) {
	if p.Observations == nil {
		return domain.Observation{}, fmt.Errorf("observation storage is unavailable")
	}
	mission, err := p.Missions.FindMission(ctx, missionID)
	if err != nil {
		return domain.Observation{}, err
	}
	if mission.Status != domain.StatusLaunched && mission.Status != domain.StatusActive {
		return domain.Observation{}, fmt.Errorf("mission is not in an observing state")
	}
	now := p.Clock.Now().UTC()
	id, err := domain.NewID()
	if err != nil {
		return domain.Observation{}, err
	}
	observation, err := domain.NewObservation(id, missionID, targetName, notes, now)
	if err != nil {
		return domain.Observation{}, err
	}
	if err := p.Observations.SaveObservation(ctx, observation); err != nil {
		return domain.Observation{}, err
	}
	if err := p.exportMission(ctx, mission); err != nil {
		return domain.Observation{}, err
	}
	if p.Exporter != nil {
		site, findErr := p.Sites.FindLaunchSite(ctx, mission.LaunchSiteID)
		if findErr != nil {
			return domain.Observation{}, findErr
		}
		if err := p.Exporter.ExportObservation(ctx, mission, site, observation); err != nil {
			return domain.Observation{}, err
		}
	}
	return observation, nil
}

func (p MissionPlanner) transitionMission(ctx context.Context, missionID string, status domain.MissionStatus) (domain.Mission, error) {
	mission, err := p.Missions.FindMission(ctx, missionID)
	if err != nil {
		return domain.Mission{}, err
	}
	if err := mission.Transition(status, p.Clock.Now().UTC()); err != nil {
		return domain.Mission{}, err
	}
	if err := p.Missions.SaveMission(ctx, mission); err != nil {
		return domain.Mission{}, err
	}
	if err := p.exportMission(ctx, mission); err != nil {
		return domain.Mission{}, err
	}
	return mission, nil
}

// CreateMission persists a planned mission and optionally exports it to a knowledge vault.
func (p MissionPlanner) CreateMission(ctx context.Context, input PlanInput) (domain.Mission, error) {
	if input.Label == "" || input.Kind == "" {
		return domain.Mission{}, fmt.Errorf("mission origin type and label are required")
	}
	now := p.Clock.Now().UTC()
	if input.Name == "" {
		location := time.UTC
		if input.Timezone != "" {
			if loaded, loadErr := time.LoadLocation(input.Timezone); loadErr == nil {
				location = loaded
			}
		}
		input.Name = "Mission " + now.In(location).Format("2006-01-02 15-04-05")
	}
	siteID, err := domain.NewID()
	if err != nil {
		return domain.Mission{}, fmt.Errorf("create launch site ID: %w", err)
	}
	missionID, err := domain.NewID()
	if err != nil {
		return domain.Mission{}, fmt.Errorf("create mission ID: %w", err)
	}
	site := domain.LaunchSite{ID: siteID, Name: input.Label, Latitude: input.Latitude, Longitude: input.Longitude, Timezone: input.Timezone, Source: input.Kind, CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission(missionID, input.Name, siteID, now)
	if err != nil {
		return domain.Mission{}, err
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		return domain.Mission{}, err
	}
	if (input.PlannedStart == nil) != (input.PlannedEnd == nil) {
		return domain.Mission{}, fmt.Errorf("mission schedule requires both start and end")
	}
	if input.PlannedStart != nil {
		if err := mission.SetSchedule(*input.PlannedStart, *input.PlannedEnd); err != nil {
			return domain.Mission{}, err
		}
	}
	var equipmentProfile domain.EquipmentProfile
	var equipmentItems []domain.EquipmentItem
	if input.EquipmentProfileID != "" {
		if p.Equipment == nil {
			return domain.Mission{}, fmt.Errorf("equipment repository is unavailable")
		}
		equipmentProfile, err = p.Equipment.FindEquipmentProfile(ctx, input.EquipmentProfileID)
		if err != nil {
			return domain.Mission{}, fmt.Errorf("find equipment profile: %w", err)
		}
		equipmentItems, err = p.Equipment.ListEquipmentItems(ctx, input.EquipmentProfileID)
		if err != nil {
			return domain.Mission{}, fmt.Errorf("list equipment items: %w", err)
		}
		mission.EquipmentProfileID = input.EquipmentProfileID
	}
	if err := p.Sites.SaveLaunchSite(ctx, site); err != nil {
		return domain.Mission{}, fmt.Errorf("save launch site: %w", err)
	}
	if err := p.Missions.SaveMission(ctx, mission); err != nil {
		return domain.Mission{}, fmt.Errorf("save mission: %w", err)
	}
	if err := p.exportMissionWithSite(ctx, mission, site, nil); err != nil {
		return domain.Mission{}, fmt.Errorf("export mission: %w", err)
	}
	if p.Exporter != nil && input.EquipmentProfileID != "" && !implementsRichExporter(p.Exporter) {
		if input.EquipmentProfileID != "" {
			if err := p.Exporter.ExportMissionEquipment(ctx, mission, site, equipmentProfile, equipmentItems); err != nil {
				return domain.Mission{}, fmt.Errorf("export mission equipment: %w", err)
			}
		}
	}
	return mission, nil
}

func (p MissionPlanner) exportMission(ctx context.Context, mission domain.Mission) error {
	if p.Exporter == nil {
		return nil
	}
	site, err := p.Sites.FindLaunchSite(ctx, mission.LaunchSiteID)
	if err != nil {
		return err
	}
	return p.exportMissionWithSite(ctx, mission, site, nil)
}

func (p MissionPlanner) exportMissionWithSite(ctx context.Context, mission domain.Mission, site domain.LaunchSite, explicit map[string]domain.TargetKnowledge) error {
	if p.Exporter == nil {
		return nil
	}
	targets := []domain.MissionTarget{}
	if p.MissionTargets != nil {
		if loaded, err := p.MissionTargets.ListMissionTargets(ctx, mission.ID); err == nil {
			targets = loaded
		}
	}
	knowledge := explicit
	if knowledge == nil {
		knowledge = make(map[string]domain.TargetKnowledge, len(targets))
		for _, target := range targets {
			knowledge[target.ID] = p.cachedTargetKnowledge(ctx, target)
		}
	}
	if rich, ok := p.Exporter.(RichMissionExporter); ok {
		return rich.ExportMissionProjection(ctx, mission, site, p.projection(ctx, mission, site, targets, knowledge))
	}
	return p.Exporter.Export(ctx, mission, site)
}

func (p MissionPlanner) projection(ctx context.Context, mission domain.Mission, site domain.LaunchSite, targets []domain.MissionTarget, knowledge map[string]domain.TargetKnowledge) projection.Mission {
	result := projection.Mission{Targets: targets, TargetKnowledge: knowledge}
	if p.Weather != nil && site.Latitude != nil && site.Longitude != nil {
		key := fmt.Sprintf("%.6f,%.6f", *site.Latitude, *site.Longitude)
		if snapshot, err := p.Weather.FindWeatherSnapshot(ctx, key); err == nil {
			result.Weather = &snapshot
		}
	}
	if p.Equipment != nil && mission.EquipmentProfileID != "" {
		if profile, err := p.Equipment.FindEquipmentProfile(ctx, mission.EquipmentProfileID); err == nil {
			result.Equipment = &profile
			result.EquipmentItems, _ = p.Equipment.ListEquipmentItems(ctx, mission.EquipmentProfileID)
		}
	}
	if p.Observations != nil {
		result.Observations, _ = p.Observations.ListObservations(ctx, mission.ID)
	}
	return result
}

func (p MissionPlanner) cachedTargetKnowledge(ctx context.Context, target domain.MissionTarget) domain.TargetKnowledge {
	if p.TargetKnowledge == nil {
		return unavailableKnowledge(target)
	}
	knowledge, err := p.TargetKnowledge.FindTargetKnowledge(ctx, target.ID)
	if err != nil {
		return unavailableKnowledge(target)
	}
	if knowledge.Status == "live" {
		knowledge.Status = "cached"
	}
	return knowledge
}

func (p MissionPlanner) resolveTargetKnowledge(ctx context.Context, target domain.MissionTarget) domain.TargetKnowledge {
	if cached := p.cachedTargetKnowledge(ctx, target); cached.Status != "unavailable" && cached.Summary != "" {
		return cached
	}
	if p.KnowledgeProvider != nil {
		if knowledge, err := p.KnowledgeProvider.Fetch(ctx, target); err == nil {
			if p.TargetKnowledge != nil {
				_ = p.TargetKnowledge.SaveTargetKnowledge(ctx, knowledge)
			}
			return knowledge
		}
	}
	return p.cachedTargetKnowledge(ctx, target)
}

func unavailableKnowledge(target domain.MissionTarget) domain.TargetKnowledge {
	return domain.TargetKnowledge{TargetID: target.ID, TargetName: target.Name, Title: target.Name, Status: "unavailable"}
}

func implementsRichExporter(exporter MissionExporter) bool {
	_, ok := exporter.(RichMissionExporter)
	return ok
}

// SystemClock returns the current UTC-aware time.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now() }

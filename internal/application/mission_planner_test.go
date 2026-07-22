package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/export/obsidian"
	"github.com/jordanistan/nightops/internal/storage/sqlite"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func TestMissionPlannerPersistsAndExports(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "nightops.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	vault := t.TempDir()
	now := time.Date(2026, 7, 21, 23, 0, 0, 0, time.UTC)
	if err := store.SaveEquipmentProfile(ctx, domain.EquipmentProfile{ID: "rig-1", Name: "Visual Rig", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEquipmentItem(ctx, domain.EquipmentItem{ID: "item-1", ProfileID: "rig-1", Name: "Dwarf Mini", Category: "Telescope", Required: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	planner := MissionPlanner{Sites: store, Missions: store, MissionTargets: store, TargetKnowledge: store, KnowledgeProvider: fakeKnowledgeProvider{}, Observations: store, Debriefs: store, Equipment: store, Exporter: obsidian.New(vault, "NightOps"), Clock: fixedClock{now: now}}
	plannedStart, plannedEnd := now.Add(2*time.Hour), now.Add(5*time.Hour)
	mission, err := planner.CreateMission(ctx, PlanInput{Kind: "ZIP Code", Label: "ZIP 78701", ZIP: "78701", Timezone: "America/Chicago", EquipmentProfileID: "rig-1", PlannedStart: &plannedStart, PlannedEnd: &plannedEnd})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Status != "planned" {
		t.Fatalf("mission was not planned: %s", mission.Status)
	}
	if mission.EquipmentProfileID != "rig-1" {
		t.Fatalf("mission equipment was not associated: %+v", mission)
	}
	if mission.PlannedStart == nil || !mission.PlannedStart.Equal(plannedStart) || mission.PlannedEnd == nil || !mission.PlannedEnd.Equal(plannedEnd) {
		t.Fatalf("mission schedule was not associated: %+v", mission)
	}
	if !strings.Contains(mission.Name, "Mission 2026-07-21") {
		t.Fatalf("mission name did not include the local mission date: %q", mission.Name)
	}
	if _, err := store.FindMission(ctx, mission.ID); err != nil {
		t.Fatal(err)
	}
	targets := []domain.MissionTarget{
		{ID: "m31", MissionID: mission.ID, Name: "Andromeda Galaxy", Kind: "galaxy", RightAscension: 10.6847, Declination: 41.269, Source: "catalog", Position: 0, CreatedAt: now},
		{ID: "m42", MissionID: mission.ID, Name: "Orion Nebula", Kind: "nebula", RightAscension: 83.822, Declination: -5.391, Source: "catalog", Position: 1, CreatedAt: now},
	}
	if err := planner.SetMissionTargets(ctx, mission.ID, targets); err != nil {
		t.Fatal(err)
	}
	loadedTargets, err := store.ListMissionTargets(ctx, mission.ID)
	if err != nil || len(loadedTargets) != 2 || loadedTargets[1].Name != "Orion Nebula" {
		t.Fatalf("mission target sequence did not round-trip: %+v err=%v", loadedTargets, err)
	}
	for _, target := range []string{"Andromeda-Galaxy.md", "Orion-Nebula.md"} {
		if _, err := os.Stat(filepath.Join(vault, "NightOps", "Targets", target)); err != nil {
			t.Fatalf("target note %s was not exported: %v", target, err)
		}
	}
	knowledgeNote, err := os.ReadFile(filepath.Join(vault, "NightOps", "Targets", "Andromeda-Galaxy.md"))
	missionNoteName := strings.ReplaceAll(mission.Name, " ", "-")
	if err != nil || !contains(string(knowledgeNote), "Cached reference for Andromeda Galaxy") || !contains(string(knowledgeNote), "[[Missions/"+missionNoteName+"]]") {
		t.Fatalf("target reference knowledge was not projected: %s err=%v", knowledgeNote, err)
	}
	site, err := store.FindLaunchSite(ctx, mission.LaunchSiteID)
	if err != nil || site.Timezone != "America/Chicago" {
		t.Fatalf("launch-site timezone was not persisted: %+v err=%v", site, err)
	}
	note, err := os.ReadFile(filepath.Join(vault, "NightOps", "Missions", missionNoteName+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(note) == "" || !contains(string(note), "unknown") || !contains(string(note), "equipment_profile_id: rig-1") || !contains(string(note), "planned_start: 2026-07-22T01:00:00Z") {
		t.Fatalf("export did not preserve unknown coordinates: %s", note)
	}
	equipmentNote, err := os.ReadFile(filepath.Join(vault, "NightOps", "Equipment", "Visual-Rig.md"))
	if err != nil || !contains(string(equipmentNote), "Dwarf Mini") || !contains(string(equipmentNote), "[[Missions/"+missionNoteName+"]]") {
		t.Fatalf("equipment projection was not exported: %s err=%v", equipmentNote, err)
	}
	missionEquipmentNote, err := os.ReadFile(filepath.Join(vault, "NightOps", "Missions", missionNoteName, "Equipment", "Visual-Rig.md"))
	if err != nil || !contains(string(missionEquipmentNote), "mission_id: "+mission.ID) || !contains(string(missionEquipmentNote), "required") {
		t.Fatalf("mission-scoped equipment projection was not exported: %s err=%v", missionEquipmentNote, err)
	}
	if _, err := planner.StartMission(ctx, mission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := planner.ActivateMission(ctx, mission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := planner.RecordObservation(ctx, mission.ID, "M 42", "Clear view"); err != nil {
		t.Fatal(err)
	}
	observations, err := store.ListObservations(ctx, mission.ID)
	if err != nil || len(observations) != 1 || observations[0].TargetName != "M 42" {
		t.Fatalf("observation did not round-trip: observations=%+v err=%v", observations, err)
	}
	if _, err := planner.CompleteMission(ctx, mission.ID); err != nil {
		t.Fatal(err)
	}
	note, err = os.ReadFile(filepath.Join(vault, "NightOps", "Missions", missionNoteName+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(note), "M 42") || !contains(string(note), "completed") {
		t.Fatalf("lifecycle export lost observation or final status: %s", note)
	}
	debrief, err := planner.RecordDebrief(ctx, mission.ID, "Clear skies and excellent transparency.")
	if err != nil || debrief.Summary != "Clear skies and excellent transparency." {
		t.Fatalf("debrief was not recorded: %+v err=%v", debrief, err)
	}
	loadedDebrief, err := store.FindDebrief(ctx, mission.ID)
	if err != nil || loadedDebrief.Summary != debrief.Summary {
		t.Fatalf("debrief did not round-trip: %+v err=%v", loadedDebrief, err)
	}
	note, err = os.ReadFile(filepath.Join(vault, "NightOps", "Missions", missionNoteName+".md"))
	if err != nil || !contains(string(note), "## Debrief") || !contains(string(note), "Clear skies and excellent transparency.") {
		t.Fatalf("debrief export missing: %s err=%v", note, err)
	}
}

type fakeKnowledgeProvider struct{}

func (fakeKnowledgeProvider) Fetch(_ context.Context, target domain.MissionTarget) (domain.TargetKnowledge, error) {
	return domain.TargetKnowledge{TargetID: target.ID, TargetName: target.Name, Title: target.Name, URL: "https://wiki.test/" + target.ID, Summary: "Cached reference for " + target.Name, Source: "fixture-wiki", Status: "live", FetchedAt: time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)}, nil
}

func contains(value, target string) bool {
	for i := 0; i+len(target) <= len(value); i++ {
		if value[i:i+len(target)] == target {
			return true
		}
	}
	return false
}

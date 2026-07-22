package obsidian

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/projection"
	"github.com/jordanistan/nightops/internal/weather"
)

func TestExportCreatesLinkedMissionLocationAndTargetNotes(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	latitude, longitude := 30.7373, -98.369
	site := domain.LaunchSite{ID: "site-1", Name: "Inks Lake State Park", Latitude: &latitude, Longitude: &longitude, Timezone: "America/Chicago", Source: "SkyBase Atlas", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-1", "First Light", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		t.Fatal(err)
	}
	start, end := now.Add(time.Hour), now.Add(4*time.Hour)
	if err := mission.SetSchedule(start, end); err != nil {
		t.Fatal(err)
	}
	if err := exporter.Export(context.Background(), mission, site); err != nil {
		t.Fatal(err)
	}
	observation := domain.Observation{ID: "observation-1", MissionID: mission.ID, TargetName: "M 42", Notes: "Clear view", CreatedAt: now, UpdatedAt: now}
	if err := exporter.ExportObservation(context.Background(), mission, site, observation); err != nil {
		t.Fatal(err)
	}
	missionNote := readNote(t, vault, "Missions", "First-Light.md")
	locationNote := readNote(t, vault, "Locations", "Inks-Lake-State-Park.md")
	targetNote := readNote(t, vault, "Targets", "M-42.md")
	for _, expected := range []string{"[[Locations/Inks-Lake-State-Park]]", "[[Targets/M-42]]"} {
		if !strings.Contains(missionNote, expected) {
			t.Errorf("mission note missing %q: %s", expected, missionNote)
		}
	}
	if !strings.Contains(missionNote, "planned_start: 2026-07-22T21:00:00Z") || !strings.Contains(missionNote, "## Mission Window") {
		t.Fatalf("mission window was not exported: %s", missionNote)
	}
	if !strings.Contains(locationNote, "id: site-1") || !strings.Contains(targetNote, "[[Missions/First-Light]]") {
		t.Fatalf("linked notes missing metadata/backlink: location=%s target=%s", locationNote, targetNote)
	}
}

func TestLocationAndTargetRewritesPreserveNotesAndAvoidDuplicateLinks(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Now().UTC()
	site := domain.LaunchSite{ID: "site-1", Name: "Dark Site", Timezone: "UTC", Source: "fixture", CreatedAt: now, UpdatedAt: now}
	mission, _ := domain.NewMission("mission-1", "Mission", site.ID, now)
	if err := exporter.Location(site); err != nil {
		t.Fatal(err)
	}
	locationPath := filepath.Join(vault, "NightOps", "Locations", "Dark-Site.md")
	if err := os.WriteFile(locationPath, []byte(readNote(t, vault, "Locations", "Dark-Site.md")+"\n## Notes\n\nKeep this field log.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exporter.Location(site); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readNote(t, vault, "Locations", "Dark-Site.md"), "Keep this field log.") {
		t.Fatal("location rewrite discarded user notes")
	}
	if err := exporter.Target(mission, "M 31"); err != nil {
		t.Fatal(err)
	}
	if err := exporter.Target(mission, "M 31"); err != nil {
		t.Fatal(err)
	}
	target := readNote(t, vault, "Targets", "M-31.md")
	if strings.Count(target, "[[Missions/Mission]]") != 1 {
		t.Fatalf("target backlink duplicated: %s", target)
	}
}

func TestExportDebriefIsIdempotent(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-1", Name: "Dark Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-1", "Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	debrief, err := domain.NewDebrief("debrief-1", mission.ID, "Clear skies", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := exporter.Export(context.Background(), mission, site); err != nil {
		t.Fatal(err)
	}
	if err := exporter.ExportDebrief(context.Background(), mission, site, debrief); err != nil {
		t.Fatal(err)
	}
	if err := exporter.ExportDebrief(context.Background(), mission, site, debrief); err != nil {
		t.Fatal(err)
	}
	note := readNote(t, vault, "Missions", "Mission.md")
	if strings.Count(note, "## Debrief") != 1 || strings.Count(note, "Clear skies") != 1 {
		t.Fatalf("debrief export was duplicated: %s", note)
	}
}

func TestExportMissionTargetCreatesMissionScopedNote(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-1", Name: "Dark Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-1", "Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	target := domain.MissionTarget{ID: "m31", MissionID: mission.ID, Name: "Andromeda Galaxy", Kind: "galaxy", RightAscension: 10.6847, Declination: 41.269, Source: "catalog", Position: 1, CreatedAt: now}
	if err := exporter.ExportMissionTarget(context.Background(), mission, site, target); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(vault, "NightOps", "Missions", "Mission", "Targets", "Andromeda-Galaxy.md")
	note, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(note), "position: 2") || !strings.Contains(string(note), "[[Targets/Andromeda-Galaxy]]") {
		t.Fatalf("mission-scoped target note missing: %s err=%v", note, err)
	}
}

func TestExportMissionEquipmentIsDeterministicAndLinked(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-1", Name: "Dark Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-1", "Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	profile := domain.EquipmentProfile{ID: "rig-1", Name: "Dwarf Mini", Description: "Portable rig", CreatedAt: now, UpdatedAt: now}
	items := []domain.EquipmentItem{{ID: "item-1", ProfileID: profile.ID, Name: "Tripod", Category: "Support", Required: true, CreatedAt: now, UpdatedAt: now}}
	ctx := context.Background()
	if err := exporter.ExportMissionEquipment(ctx, mission, site, profile, items); err != nil {
		t.Fatal(err)
	}
	globalPath := filepath.Join(vault, "NightOps", "Equipment", "Dwarf-Mini.md")
	if err := os.WriteFile(globalPath, []byte(noteWithSection(t, globalPath, "## Notes\n\nKeep the batteries charged.\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exporter.ExportMissionEquipment(ctx, mission, site, profile, items); err != nil {
		t.Fatal(err)
	}
	note := readNote(t, vault, "Equipment", "Dwarf-Mini.md")
	if !strings.Contains(note, "Tripod") || strings.Count(note, "Tripod") != 1 || !strings.Contains(note, "[[Missions/Mission]]") || !strings.Contains(note, "Keep the batteries charged.") {
		t.Fatalf("equipment note was not deterministic: %s", note)
	}
	missionNote := readNote(t, vault, filepath.Join("Missions", "Mission", "Equipment"), "Dwarf-Mini.md")
	if !strings.Contains(missionNote, "mission_id: mission-1") || !strings.Contains(missionNote, "profile_id: rig-1") {
		t.Fatalf("mission equipment note missing scoped metadata: %s", missionNote)
	}
}

func TestRichMissionProjectionWritesLiveConditionsTargetsAndHistory(t *testing.T) {
	vault := t.TempDir()
	exporter := New(vault, "NightOps")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	latitude, longitude := 30.2672, -97.7431
	site := domain.LaunchSite{ID: "site-1", Name: "Home Base", Latitude: &latitude, Longitude: &longitude, Timezone: "UTC", Source: "Home Base", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-1", "Live Session", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	target := domain.MissionTarget{ID: "m31", MissionID: mission.ID, Name: "Andromeda Galaxy", Kind: "galaxy", Source: "catalog", Position: 0, CreatedAt: now}
	temperature, cloud, precip := 22.5, 18.0, 4.0
	snapshot := weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "fixture", ObservedAt: now, ExpiresAt: now.Add(time.Hour), TemperatureC: &temperature, CloudCoverPercent: &cloud, Forecast: []weather.ForecastPoint{{At: now.Add(time.Hour), TemperatureC: &temperature, CloudCoverPercent: &cloud, PrecipitationProbability: &precip}}}
	knowledge := domain.TargetKnowledge{TargetID: target.ID, TargetName: target.Name, Title: target.Name, URL: "https://wiki.test/m31", Summary: "A nearby galaxy.", ImageURL: "https://images.test/m31.jpg", Source: "wikipedia", Status: "live", FetchedAt: now}
	if err := exporter.ExportMissionProjection(context.Background(), mission, site, projection.Mission{Weather: &snapshot, Targets: []domain.MissionTarget{target}, TargetKnowledge: map[string]domain.TargetKnowledge{target.ID: knowledge}}); err != nil {
		t.Fatal(err)
	}
	note := readNote(t, vault, "Missions", "Live-Session.md")
	targetNote := readNote(t, vault, "Targets", "Andromeda-Galaxy.md")
	for _, expected := range []string{"## Live Conditions", "22.5°C", "## Equipment Checklist", "Telescope or optical tube", "## Selected Targets", "| # | Target | Type | Capture guidance | Recommended starting settings | Reference |", "Use a wider field", "Broadband / UV-IR filter", "[[Targets/Andromeda-Galaxy]]", "## Hourly Forecast"} {
		if !strings.Contains(note, expected) {
			t.Fatalf("rich mission note missing %q: %s", expected, note)
		}
	}
	if strings.Index(note, "## Hourly Forecast") < strings.Index(note, "## Selected Targets") {
		t.Fatalf("hourly forecast should follow the target table: %s", note)
	}
	for _, expected := range []string{"A nearby galaxy.", "https://images.test/m31.jpg", "[[Missions/Live-Session]]", "Home Base"} {
		if !strings.Contains(targetNote, expected) {
			t.Fatalf("target knowledge/history missing %q: %s", expected, targetNote)
		}
	}
	if !strings.Contains(targetNote, "## Capture Settings") || !strings.Contains(targetNote, "Broadband / UV-IR filter") {
		t.Fatalf("target note missing reusable capture settings: %s", targetNote)
	}
}

func readNote(t *testing.T, vault, section, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(vault, "NightOps", section, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func noteWithSection(t *testing.T, path, section string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data) + "\n" + section
}

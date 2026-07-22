package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/atlas"
	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/routing"
	"github.com/jordanistan/nightops/internal/weather"
)

func TestOpenAppliesSchema(t *testing.T) {
	store, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBackupCreatesReadableOwnerOnlyDatabase(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	databasePath := filepath.Join(directory, "nightops.db")
	backupPath := filepath.Join(directory, "backups", "nightops.db")
	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Backup(ctx, backupPath); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions = %o", info.Mode().Perm())
	}
	backup, err := Open(ctx, backupPath)
	if err != nil {
		t.Fatalf("backup was not readable: %v", err)
	}
	defer backup.Close()
	if err := backup.Ping(ctx); err != nil {
		t.Fatalf("backup did not contain a usable database: %v", err)
	}
}

func TestBackupRejectsActiveDatabaseAndEmptyPath(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "nightops.db")
	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Backup(ctx, ""); err == nil {
		t.Fatal("empty backup path was accepted")
	}
	if err := store.Backup(ctx, databasePath); err == nil {
		t.Fatal("active database path was accepted")
	}
}

func TestRepositoriesRoundTripUnknownCoordinates(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC().Truncate(time.Second)
	site := domain.LaunchSite{ID: "site-1", Name: "ZIP Origin", Timezone: "America/Chicago", Source: "ZIP Code", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	loadedSite, err := store.FindLaunchSite(ctx, site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedSite.Latitude != nil || loadedSite.Longitude != nil {
		t.Fatalf("unknown coordinates were not preserved as NULL: %+v", loadedSite)
	}
	mission, err := domain.NewMission("mission-1", "ZIP Operation", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMission(ctx, mission); err != nil {
		t.Fatal(err)
	}
	loadedMission, err := store.FindMission(ctx, mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedMission.Status != domain.StatusPlanned || loadedMission.LaunchSiteID != site.ID {
		t.Fatalf("mission did not round-trip: %+v", loadedMission)
	}
}

func TestDebriefRoundTripAndUpdate(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 22, 3, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-debrief", Name: "Debrief Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	mission, err := domain.NewMission("mission-debrief", "Debrief Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusLaunched, now); err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusActive, now); err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusCompleted, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMission(ctx, mission); err != nil {
		t.Fatal(err)
	}
	first, _ := domain.NewDebrief("debrief-1", mission.ID, "First note", now)
	if err := store.SaveDebrief(ctx, first); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.FindDebrief(ctx, mission.ID)
	if err != nil || loaded.ID != first.ID || loaded.Summary != first.Summary {
		t.Fatalf("debrief did not round-trip: %+v err=%v", loaded, err)
	}
	second, _ := domain.NewDebrief("debrief-2", mission.ID, "Updated note", now.Add(time.Minute))
	if err := store.SaveDebrief(ctx, second); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.FindDebrief(ctx, mission.ID)
	if err != nil || loaded.ID != second.ID || loaded.Summary != second.Summary {
		t.Fatalf("debrief did not update by mission: %+v err=%v", loaded, err)
	}
}

func TestMissionTargetSequenceRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 22, 3, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-targets", Name: "Target Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	mission, err := domain.NewMission("mission-targets", "Target Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMission(ctx, mission); err != nil {
		t.Fatal(err)
	}
	targets := []domain.MissionTarget{
		{ID: "first", MissionID: mission.ID, Name: "First Target", Kind: "galaxy", RightAscension: 1, Declination: 2, Source: "test", Position: 0, CreatedAt: now},
		{ID: "second", MissionID: mission.ID, Name: "Second Target", Kind: "nebula", RightAscension: 3, Declination: 4, Source: "test", Position: 1, CreatedAt: now},
	}
	if err := store.ReplaceMissionTargets(ctx, mission.ID, targets); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.ListMissionTargets(ctx, mission.ID)
	if err != nil || len(loaded) != 2 || loaded[0].Name != "First Target" || loaded[1].Position != 1 {
		t.Fatalf("target sequence did not round-trip in order: %+v err=%v", loaded, err)
	}
	if err := store.ReplaceMissionTargets(ctx, mission.ID, targets[:1]); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.ListMissionTargets(ctx, mission.ID)
	if err != nil || len(loaded) != 1 || loaded[0].ID != "first" {
		t.Fatalf("target sequence was not replaced: %+v err=%v", loaded, err)
	}
}

func TestListMissionsReturnsNewestPlannedFirst(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-archive", Name: "Archive Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	first, _ := domain.NewMission("mission-old", "Old Mission", site.ID, now)
	_ = first.Transition(domain.StatusPlanned, now)
	second, _ := domain.NewMission("mission-new", "New Mission", site.ID, now.Add(time.Hour))
	_ = second.Transition(domain.StatusPlanned, now.Add(time.Hour))
	if err := store.SaveMission(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMission(ctx, second); err != nil {
		t.Fatal(err)
	}
	missions, err := store.ListMissions(ctx)
	if err != nil || len(missions) != 2 || missions[0].ID != "mission-new" {
		t.Fatalf("mission archive order = %+v err=%v", missions, err)
	}
}

func TestWeatherCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	temperature := 21.5
	snapshot := weather.Snapshot{LocationKey: "site-a", Source: "local-fixture", ObservedAt: now, ExpiresAt: now.Add(time.Hour), TemperatureC: &temperature, Payload: "{}"}
	snapshot.Forecast = []weather.ForecastPoint{{At: now.Add(time.Hour), TemperatureC: &temperature}}
	if err := store.SaveWeatherSnapshot(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.FindWeatherSnapshot(ctx, "site-a")
	if err != nil || loaded.TemperatureC == nil || *loaded.TemperatureC != temperature || len(loaded.Forecast) != 1 || !loaded.Fresh(now.Add(30*time.Minute)) {
		t.Fatalf("weather snapshot did not round-trip: %+v err=%v", loaded, err)
	}
}

func TestRouteCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	plan := routing.Plan{FromLabel: "Home", ToLabel: "Dark Site", DistanceKm: 42.5, DistanceMi: 26.4, TravelTime: "1 hours", Provider: "osrm"}
	if err := store.SaveRoute(ctx, "home->site", plan, now, now.Add(6*time.Hour)); err != nil {
		t.Fatal(err)
	}
	loaded, fetched, expires, err := store.FindRoute(ctx, "home->site")
	if err != nil || loaded != plan || !fetched.Equal(now) || !expires.Equal(now.Add(6*time.Hour)) {
		t.Fatalf("route cache did not round-trip: %+v %v %v err=%v", loaded, fetched, expires, err)
	}
}

func TestEquipmentProfileAndMissionAssociationRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC().Truncate(time.Second)
	profile := domain.EquipmentProfile{ID: "rig-1", Name: "Visual Rig", Description: "80mm refractor", CreatedAt: now, UpdatedAt: now}
	if err := store.SaveEquipmentProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	loadedProfile, err := store.FindEquipmentProfile(ctx, profile.ID)
	if err != nil || loadedProfile != profile {
		t.Fatalf("equipment profile did not round-trip: %+v err=%v", loadedProfile, err)
	}
	nowSite := now
	site := domain.LaunchSite{ID: "site-1", Name: "Home", Timezone: "UTC", Source: "test", CreatedAt: nowSite, UpdatedAt: nowSite}
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	mission, err := domain.NewMission("mission-1", "Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	mission.EquipmentProfileID = profile.ID
	if err := store.SaveMission(ctx, mission); err != nil {
		t.Fatal(err)
	}
	loadedMission, err := store.FindMission(ctx, mission.ID)
	if err != nil || loadedMission.EquipmentProfileID != profile.ID {
		t.Fatalf("mission equipment association did not round-trip: %+v err=%v", loadedMission, err)
	}
	item := domain.EquipmentItem{ID: "item-1", ProfileID: profile.ID, Name: "Eyepiece case", Category: "Optics", Required: true, CreatedAt: now, UpdatedAt: now}
	if err := store.SaveEquipmentItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListEquipmentItems(ctx, profile.ID)
	if err != nil || len(items) != 1 || items[0] != item {
		t.Fatalf("equipment inventory did not round-trip: %+v err=%v", items, err)
	}
}

func TestAtlasCatalogReplacementRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first := atlas.Catalog{Version: "atlas-1", Locations: []atlas.Location{{ID: "site-a", Name: "First Site", Latitude: 30, Longitude: -97, Timezone: "UTC", Source: "survey"}}}
	if err := store.SaveAtlasCatalog(ctx, first); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadActiveAtlasCatalog(ctx)
	if err != nil || loaded.Version != first.Version || len(loaded.Locations) != 1 || loaded.Locations[0].Name != "First Site" {
		t.Fatalf("first atlas catalog did not round-trip: %+v err=%v", loaded, err)
	}
	second := atlas.Catalog{Version: "atlas-2", Locations: []atlas.Location{{ID: "site-b", Name: "Second Site", Latitude: 31, Longitude: -98, Timezone: "UTC", Source: "community"}}}
	if err := store.SaveAtlasCatalog(ctx, second); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.LoadActiveAtlasCatalog(ctx)
	if err != nil || loaded.Version != second.Version || loaded.Locations[0].ID != "site-b" {
		t.Fatalf("active atlas catalog was not replaced: %+v err=%v", loaded, err)
	}
}

func TestTargetKnowledgeRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	knowledge := domain.TargetKnowledge{TargetID: "m31", TargetName: "Andromeda Galaxy", Title: "Andromeda Galaxy", URL: "https://wiki.test/m31", Summary: "A nearby galaxy.", ImageURL: "https://images.test/m31.jpg", Source: "wikipedia", Status: "live", FetchedAt: now}
	if err := store.SaveTargetKnowledge(ctx, knowledge); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.FindTargetKnowledge(ctx, knowledge.TargetID)
	if err != nil || loaded != knowledge {
		t.Fatalf("target knowledge did not round-trip: %+v err=%v", loaded, err)
	}
}

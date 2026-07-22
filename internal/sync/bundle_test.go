package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/storage/sqlite"
	"github.com/jordanistan/nightops/internal/sync"
)

func TestBundleRoundTripAndNewerMerge(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "site-sync", Name: "Sync Site", Timezone: "UTC", Source: "local", CreatedAt: now, UpdatedAt: now}
	mission, err := domain.NewMission("mission-sync", "Sync Mission", site.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mission.Transition(domain.StatusPlanned, now); err != nil {
		t.Fatal(err)
	}
	bundle, err := sync.NewBundle(now, []domain.LaunchSite{site}, []domain.Mission{mission}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "nightops.sync.json")
	if err := sync.Save(path, bundle); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("bundle permissions=%o", info.Mode().Perm())
	}
	loaded, err := sync.Load(path)
	if err != nil || loaded.Missions[0].ID != mission.ID {
		t.Fatalf("bundle round trip=%+v err=%v", loaded, err)
	}
	store, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	report, err := loaded.Merge(ctx, store)
	if err != nil || report.Added != 2 || report.Updated != 0 {
		t.Fatalf("initial merge report=%+v err=%v", report, err)
	}
	newer := mission
	newer.Name = "Updated Mission"
	newer.UpdatedAt = now.Add(time.Hour)
	updatedBundle, err := sync.NewBundle(now.Add(time.Hour), nil, []domain.Mission{newer}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err = updatedBundle.Merge(ctx, store)
	if err != nil || report.Updated != 1 {
		t.Fatalf("newer merge report=%+v err=%v", report, err)
	}
	local, err := store.FindMission(ctx, mission.ID)
	if err != nil || local.Name != "Updated Mission" {
		t.Fatalf("newer record was not applied: %+v err=%v", local, err)
	}
	older, _ := sync.NewBundle(now, nil, []domain.Mission{mission}, nil, nil, nil)
	report, err = older.Merge(ctx, store)
	if err != nil || report.Skipped != 1 {
		t.Fatalf("older merge was not skipped: %+v err=%v", report, err)
	}
}

func TestBundleRejectsUnsupportedSchemaAndDuplicateIDs(t *testing.T) {
	now := time.Now().UTC()
	site := domain.LaunchSite{ID: "duplicate", Name: "Site", Timezone: "UTC", Source: "test", CreatedAt: now, UpdatedAt: now}
	bundle := sync.Bundle{SchemaVersion: sync.CurrentVersion, GeneratedAt: now, LaunchSites: []domain.LaunchSite{site, site}}
	if err := bundle.Validate(); err == nil {
		t.Fatal("duplicate IDs were accepted")
	}
	bundle = sync.Bundle{SchemaVersion: sync.CurrentVersion + 1, GeneratedAt: now}
	if err := bundle.Validate(); err == nil {
		t.Fatal("unsupported schema was accepted")
	}
}

func TestBundleReportsEqualTimestampConflictWithoutOverwritingLocalData(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	site := domain.LaunchSite{ID: "conflict-site", Name: "Local Name", Timezone: "UTC", Source: "local", CreatedAt: now, UpdatedAt: now}
	store, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.SaveLaunchSite(ctx, site); err != nil {
		t.Fatal(err)
	}
	incoming := site
	incoming.Name = "Peer Name"
	incoming.Source = "peer"
	bundle, err := sync.NewBundleWithDevice("peer-device", now, []domain.LaunchSite{incoming}, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := bundle.Merge(ctx, store)
	if err != nil || report.Conflicts != 1 || len(report.ConflictIDs) != 1 || report.Skipped != 1 {
		t.Fatalf("conflict report=%+v err=%v", report, err)
	}
	loaded, err := store.FindLaunchSite(ctx, site.ID)
	if err != nil || loaded.Name != "Local Name" {
		t.Fatalf("local record was overwritten: %+v err=%v", loaded, err)
	}
}

func TestLoadOrCreateDeviceIDIsStableAndOwnerReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync", "device-id")
	first, err := sync.LoadOrCreateDeviceID(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sync.LoadOrCreateDeviceID(path)
	if err != nil || first == "" || first != second {
		t.Fatalf("device ID was not stable: first=%q second=%q err=%v", first, second, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("device ID permissions=%v err=%v", info.Mode().Perm(), err)
	}
}

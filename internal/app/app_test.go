package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/astronomy"
	"github.com/jordanistan/nightops/internal/config"
	"github.com/jordanistan/nightops/internal/console"
	"github.com/jordanistan/nightops/internal/routing"
	syncbundle "github.com/jordanistan/nightops/internal/sync"
	"github.com/jordanistan/nightops/internal/weather"
)

func TestPrepareObsidianWorkspaceCreatesDefaultNotesVault(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Documents", "Obsidian")
	cfg := config.Defaults()
	cfg.Obsidian.VaultDir = root
	cfg.Obsidian.NotesDir = "NightOps"

	preparedRoot, notes, err := prepareObsidianWorkspace(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if preparedRoot != root || notes != filepath.Join(root, "NightOps") {
		t.Fatalf("unexpected Obsidian paths: root=%q notes=%q", preparedRoot, notes)
	}
	for _, path := range []string{root, notes, filepath.Join(notes, ".obsidian")} {
		info, statErr := os.Stat(path)
		if statErr != nil || !info.IsDir() {
			t.Fatalf("expected Obsidian directory %q: %v", path, statErr)
		}
	}
	if status := obsidianStatus(cfg); status != "READY" {
		t.Fatalf("prepared workspace status=%q", status)
	}
}

func TestObsidianURIUsesVaultNameInsteadOfOpeningFileBrowser(t *testing.T) {
	got, err := obsidianURI("/tmp/Obsidian/NightOps")
	if err != nil {
		t.Fatal(err)
	}
	if got != "obsidian://open?vault=NightOps&file=Index" {
		t.Fatalf("unexpected Obsidian URI: %q", got)
	}
}

func TestFormatSyncMergeReportSurfacesConflictIDs(t *testing.T) {
	report := syncbundle.MergeReport{Added: 1, Updated: 2, Skipped: 3, Conflicts: 2, ConflictIDs: []string{"mission:m-1", "launch_site:s-1"}}
	message := formatSyncMergeReport(report)
	for _, expected := range []string{"1 added", "2 updated", "2 conflicts retained locally", "mission:m-1", "launch_site:s-1"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("merge report %q does not contain %q", message, expected)
		}
	}
}

func TestFormatSyncMergeReportLimitsLongConflictList(t *testing.T) {
	ids := make([]string, 10)
	for index := range ids {
		ids[index] = fmt.Sprintf("mission:m-%d", index)
	}
	message := formatSyncMergeReport(syncbundle.MergeReport{Conflicts: len(ids), ConflictIDs: ids})
	if !strings.Contains(message, "(+2 more)") || strings.Contains(message, "mission:m-9") {
		t.Fatalf("long merge report was not bounded: %q", message)
	}
}

func TestInitializeWeatherUsesFreshCacheOffline(t *testing.T) {
	now := time.Now().UTC()
	cache := fakeWeatherCache{snapshot: weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "open-meteo", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}}
	status := initializeWeather(context.Background(), config.Config{Features: config.FeatureConfig{Weather: true}}, &cache, failingWeatherProvider{}, 30.2672, -97.7431)
	if status != "READY" {
		t.Fatalf("fresh cache status = %q", status)
	}
	if cache.saved {
		t.Fatal("fresh cache should not be overwritten")
	}
}

func TestInitializeWeatherStoresProviderResult(t *testing.T) {
	cache := fakeWeatherCache{}
	now := time.Now().UTC()
	provider := fakeWeatherProvider{snapshot: weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "fixture", ObservedAt: now, ExpiresAt: now.Add(time.Hour)}}
	status := initializeWeather(context.Background(), config.Config{Features: config.FeatureConfig{Weather: true}}, &cache, provider, 30.2672, -97.7431)
	if status != "READY" || !cache.saved {
		t.Fatalf("provider result was not cached: status=%q saved=%v", status, cache.saved)
	}
}

func TestInitializeWeatherReportsStaleCacheWhenProviderFails(t *testing.T) {
	now := time.Now().UTC()
	cache := fakeWeatherCache{snapshot: weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "open-meteo", ObservedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)}}
	status := initializeWeather(context.Background(), config.Config{Features: config.FeatureConfig{Weather: true}}, &cache, failingWeatherProvider{}, 30.2672, -97.7431)
	if status != "STANDBY" {
		t.Fatalf("stale cache status = %q", status)
	}
}

func TestInitializeWeatherDoesNotInventDataWithoutCoordinates(t *testing.T) {
	status := initializeWeather(context.Background(), config.Config{Features: config.FeatureConfig{Weather: true}}, &fakeWeatherCache{}, fakeWeatherProvider{}, 0, 0)
	if status != "NOT CONFIGURED" {
		t.Fatalf("unknown coordinates status = %q", status)
	}
}

func TestWeatherSummaryLabelsFreshAndStaleData(t *testing.T) {
	temperature, cloud := 28.4, 17.0
	now := time.Now().UTC()
	cache := fakeWeatherCache{snapshot: weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "fixture", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), TemperatureC: &temperature, CloudCoverPercent: &cloud}}
	cfg := config.Config{Features: config.FeatureConfig{Weather: true}}
	origin := consoleOrigin(30.2672, -97.7431)
	fresh := weatherSummary(context.Background(), cfg, &cache, failingWeatherProvider{}, origin)
	if fresh != "28.4°C · clouds 17% · fixture" {
		t.Fatalf("fresh weather summary = %q", fresh)
	}
	cache.snapshot.ExpiresAt = now.Add(-time.Minute)
	stale := weatherSummary(context.Background(), cfg, &cache, failingWeatherProvider{}, origin)
	if stale != "28.4°C · clouds 17% · fixture · stale" {
		t.Fatalf("stale weather summary = %q", stale)
	}
}

func TestWeatherForecastSummaryUsesCachedHourlyPoints(t *testing.T) {
	now := time.Now().UTC()
	temperature := 21.5
	cloud := 12.0
	probability := 5.0
	cache := fakeWeatherCache{snapshot: weather.Snapshot{LocationKey: "30.267200,-97.743100", Source: "fixture", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), Forecast: []weather.ForecastPoint{{At: now.Add(time.Hour), TemperatureC: &temperature, CloudCoverPercent: &cloud, PrecipitationProbability: &probability}}}}
	result := weatherForecastSummary(context.Background(), config.Config{Features: config.FeatureConfig{Weather: true}, Origin: config.OriginConfig{Timezone: "UTC"}}, &cache, failingWeatherProvider{}, consoleOrigin(30.2672, -97.7431))
	want := formatLocalTime(now.Add(time.Hour), "UTC") + " 21.5°C clouds 12% precip 5%"
	if result != want {
		t.Fatalf("cached forecast summary = %q want %q", result, want)
	}
}

func TestRouteSummaryUsesProviderThenFreshCache(t *testing.T) {
	cache := &fakeRouteCache{}
	cfg := config.Config{Origin: config.OriginConfig{HomeBaseName: "Home Base", Latitude: 30.2672, Longitude: -97.7431}, Routing: config.RoutingConfig{CacheMinutes: 60}}
	origin := consoleOrigin(30.7373, -98.369)
	provider := fakeRouteProvider{plan: routing.Plan{FromLabel: "origin", ToLabel: "destination", DistanceKm: 58.2, DistanceMi: 36.2, TravelTime: "1 hours", Provider: "osrm"}}
	first := routeSummary(context.Background(), cfg, cache, provider, origin)
	if !cache.saved || !strings.Contains(first, "SOURCE         osrm") || !strings.Contains(first, "1 hours") {
		t.Fatalf("provider route was not used: saved=%v summary=%q", cache.saved, first)
	}
	second := routeSummary(context.Background(), cfg, cache, failingRouteProvider{}, origin)
	if !strings.Contains(second, "osrm (cached)") {
		t.Fatalf("fresh cached route was not used: %q", second)
	}
}

func TestRouteSummaryUsesStaleCacheWhenProviderFails(t *testing.T) {
	now := time.Now().UTC()
	cache := &fakeRouteCache{plan: routing.Plan{FromLabel: "origin", ToLabel: "destination", DistanceKm: 58.2, DistanceMi: 36.2, TravelTime: "1 hours", Provider: "osrm"}, fetchedAt: now.Add(-2 * time.Hour), expiresAt: now.Add(-time.Hour), saved: true}
	cfg := config.Config{Origin: config.OriginConfig{HomeBaseName: "Home Base", Latitude: 30.2672, Longitude: -97.7431}, Routing: config.RoutingConfig{CacheMinutes: 60}}
	result := routeSummary(context.Background(), cfg, cache, failingRouteProvider{}, consoleOrigin(30.7373, -98.369))
	if !strings.Contains(result, "osrm (cached)") || !strings.Contains(result, "1 hours") {
		t.Fatalf("stale route cache was not retained: %q", result)
	}
}

func TestTargetMinimumAltitudeDefaultsAndOverrides(t *testing.T) {
	if got := targetMinimumAltitude(config.Config{}); got != 30 {
		t.Fatalf("default target altitude = %.0f, want 30", got)
	}
	cfg := config.Config{Astronomy: config.AstronomyConfig{MinimumTargetAltitude: 45}}
	if got := targetMinimumAltitude(cfg); got != 45 {
		t.Fatalf("configured target altitude = %.0f, want 45", got)
	}
}

func consoleOrigin(latitude, longitude float64) console.Origin {
	return console.Origin{Kind: "Atlas", Label: "Test Site", Latitude: &latitude, Longitude: &longitude}
}

type fakeWeatherCache struct {
	snapshot weather.Snapshot
	saved    bool
}

func (f fakeWeatherCache) FindWeatherSnapshot(context.Context, string) (weather.Snapshot, error) {
	if f.snapshot.LocationKey == "" {
		return weather.Snapshot{}, errors.New("cache miss")
	}
	return f.snapshot, nil
}

func (f *fakeWeatherCache) SaveWeatherSnapshot(_ context.Context, snapshot weather.Snapshot) error {
	f.snapshot = snapshot
	f.saved = true
	return nil
}

type fakeWeatherProvider struct{ snapshot weather.Snapshot }

func (f fakeWeatherProvider) Fetch(context.Context, float64, float64) (weather.Snapshot, error) {
	return f.snapshot, nil
}

type failingWeatherProvider struct{}

func (failingWeatherProvider) Fetch(context.Context, float64, float64) (weather.Snapshot, error) {
	return weather.Snapshot{}, errors.New("offline")
}

type fakeRouteCache struct {
	plan      routing.Plan
	fetchedAt time.Time
	expiresAt time.Time
	saved     bool
}

func (f *fakeRouteCache) SaveRoute(_ context.Context, _ string, plan routing.Plan, fetchedAt, expiresAt time.Time) error {
	f.plan, f.fetchedAt, f.expiresAt, f.saved = plan, fetchedAt, expiresAt, true
	return nil
}

func (f *fakeRouteCache) FindRoute(_ context.Context, _ string) (routing.Plan, time.Time, time.Time, error) {
	if !f.saved {
		return routing.Plan{}, time.Time{}, time.Time{}, errors.New("route cache miss")
	}
	return f.plan, f.fetchedAt, f.expiresAt, nil
}

type fakeRouteProvider struct{ plan routing.Plan }

func (f fakeRouteProvider) Route(context.Context, astronomy.Coordinates, astronomy.Coordinates) (routing.Plan, error) {
	return f.plan, nil
}

type failingRouteProvider struct{}

func (failingRouteProvider) Route(context.Context, astronomy.Coordinates, astronomy.Coordinates) (routing.Plan, error) {
	return routing.Plan{}, errors.New("offline")
}

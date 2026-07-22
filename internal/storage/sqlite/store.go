// Package sqlite provides the local durable store.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/atlas"
	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/routing"
	"github.com/jordanistan/nightops/internal/weather"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store owns a SQLite connection and its schema lifecycle.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens a SQLite database and applies all embedded migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, path: path}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate(ctx context.Context) error {
	entries, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)
	for _, entry := range entries {
		version, err := migrationVersion(entry)
		if err != nil {
			return err
		}
		if version > 1 {
			var applied int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err != nil {
				return fmt.Errorf("check migration %d: %w", version, err)
			}
			if applied > 0 {
				continue
			}
		}
		data, err := migrationFS.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry, err)
		}
		if _, err := s.db.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}
	return nil
}

func migrationVersion(path string) (int, error) {
	name := filepath.Base(path)
	value := strings.SplitN(name, "_", 2)[0]
	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid migration filename %q: %w", name, err)
	}
	return version, nil
}

// SaveLaunchSite inserts or updates a launch site.
func (s *Store) SaveLaunchSite(ctx context.Context, site domain.LaunchSite) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO launch_sites (id, name, latitude, longitude, timezone, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, latitude=excluded.latitude, longitude=excluded.longitude, timezone=excluded.timezone, source=excluded.source, updated_at=excluded.updated_at`, site.ID, site.Name, nullableFloat(site.Latitude), nullableFloat(site.Longitude), site.Timezone, site.Source, site.CreatedAt.UTC().Format(time.RFC3339), site.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// FindLaunchSite loads a launch site by stable ID.
func (s *Store) FindLaunchSite(ctx context.Context, id string) (domain.LaunchSite, error) {
	var site domain.LaunchSite
	var latitude, longitude sql.NullFloat64
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, latitude, longitude, timezone, source, created_at, updated_at FROM launch_sites WHERE id = ?`, id).Scan(&site.ID, &site.Name, &latitude, &longitude, &site.Timezone, &site.Source, &created, &updated)
	if err != nil {
		return domain.LaunchSite{}, err
	}
	site.Latitude, site.Longitude = floatPointer(latitude), floatPointer(longitude)
	site.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return domain.LaunchSite{}, err
	}
	site.UpdatedAt, err = time.Parse(time.RFC3339, updated)
	return site, err
}

// ReplaceMissionTargets replaces the planned target sequence atomically.
func (s *Store) ReplaceMissionTargets(ctx context.Context, missionID string, targets []domain.MissionTarget) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM mission_targets WHERE mission_id = ?`, missionID); err != nil {
		return err
	}
	for _, target := range targets {
		if target.MissionID == "" {
			target.MissionID = missionID
		}
		if target.MissionID != missionID {
			return fmt.Errorf("mission target belongs to %q, expected %q", target.MissionID, missionID)
		}
		if err := target.Validate(); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mission_targets (mission_id, target_id, target_name, target_kind, right_ascension, declination, source, position, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, target.MissionID, target.ID, target.Name, target.Kind, target.RightAscension, target.Declination, target.Source, target.Position, target.CreatedAt.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListMissionTargets returns the planned target sequence in observer order.
func (s *Store) ListMissionTargets(ctx context.Context, missionID string) ([]domain.MissionTarget, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT mission_id, target_id, target_name, target_kind, right_ascension, declination, source, position, created_at FROM mission_targets WHERE mission_id = ? ORDER BY position, target_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []domain.MissionTarget
	for rows.Next() {
		var target domain.MissionTarget
		var created string
		if err := rows.Scan(&target.MissionID, &target.ID, &target.Name, &target.Kind, &target.RightAscension, &target.Declination, &target.Source, &target.Position, &created); err != nil {
			return nil, err
		}
		if target.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

// SaveTargetKnowledge stores the latest source-attributed reference material
// for a catalog target. It is keyed by stable catalog ID for offline reuse.
func (s *Store) SaveTargetKnowledge(ctx context.Context, knowledge domain.TargetKnowledge) error {
	if err := knowledge.Validate(); err != nil {
		return err
	}
	var fetched any
	if !knowledge.FetchedAt.IsZero() {
		fetched = knowledge.FetchedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO target_knowledge (target_id, target_name, title, url, summary, image_url, source, status, fetched_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(target_id) DO UPDATE SET target_name=excluded.target_name, title=excluded.title, url=excluded.url, summary=excluded.summary, image_url=excluded.image_url, source=excluded.source, status=excluded.status, fetched_at=excluded.fetched_at`, knowledge.TargetID, knowledge.TargetName, knowledge.Title, knowledge.URL, knowledge.Summary, knowledge.ImageURL, knowledge.Source, knowledge.Status, fetched)
	return err
}

// FindTargetKnowledge loads cached reference material for a catalog target.
func (s *Store) FindTargetKnowledge(ctx context.Context, targetID string) (domain.TargetKnowledge, error) {
	var knowledge domain.TargetKnowledge
	var fetched sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT target_id, target_name, title, url, summary, image_url, source, status, fetched_at FROM target_knowledge WHERE target_id = ?`, targetID).Scan(&knowledge.TargetID, &knowledge.TargetName, &knowledge.Title, &knowledge.URL, &knowledge.Summary, &knowledge.ImageURL, &knowledge.Source, &knowledge.Status, &fetched)
	if err != nil {
		return domain.TargetKnowledge{}, err
	}
	if fetched.Valid && fetched.String != "" {
		knowledge.FetchedAt, err = time.Parse(time.RFC3339, fetched.String)
		if err != nil {
			return domain.TargetKnowledge{}, err
		}
	}
	return knowledge, nil
}

// SaveEquipmentProfile inserts or updates a local equipment profile.
func (s *Store) SaveEquipmentProfile(ctx context.Context, profile domain.EquipmentProfile) error {
	if profile.ID == "" || profile.Name == "" {
		return fmt.Errorf("equipment profile id and name are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO equipment_profiles (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, updated_at=excluded.updated_at`, profile.ID, profile.Name, profile.Description, profile.CreatedAt.UTC().Format(time.RFC3339), profile.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// FindEquipmentProfile loads one local equipment profile.
func (s *Store) FindEquipmentProfile(ctx context.Context, id string) (domain.EquipmentProfile, error) {
	var profile domain.EquipmentProfile
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, created_at, updated_at FROM equipment_profiles WHERE id = ?`, id).Scan(&profile.ID, &profile.Name, &profile.Description, &created, &updated)
	if err != nil {
		return domain.EquipmentProfile{}, err
	}
	profile.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return domain.EquipmentProfile{}, err
	}
	profile.UpdatedAt, err = time.Parse(time.RFC3339, updated)
	return profile, err
}

// ListEquipmentProfiles returns local profiles in stable display order.
func (s *Store) ListEquipmentProfiles(ctx context.Context) ([]domain.EquipmentProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at FROM equipment_profiles ORDER BY name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := make([]domain.EquipmentProfile, 0)
	for rows.Next() {
		var profile domain.EquipmentProfile
		var created, updated string
		if err := rows.Scan(&profile.ID, &profile.Name, &profile.Description, &created, &updated); err != nil {
			return nil, err
		}
		profile.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return nil, err
		}
		profile.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

// SaveEquipmentItem inserts or updates an inventory item.
func (s *Store) SaveEquipmentItem(ctx context.Context, item domain.EquipmentItem) error {
	if item.ID == "" || item.ProfileID == "" || item.Name == "" || item.Category == "" {
		return fmt.Errorf("equipment item id, profile, name, and category are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO equipment_items (id, profile_id, name, category, required, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET profile_id=excluded.profile_id, name=excluded.name, category=excluded.category, required=excluded.required, updated_at=excluded.updated_at`, item.ID, item.ProfileID, item.Name, item.Category, item.Required, item.CreatedAt.UTC().Format(time.RFC3339), item.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// ListEquipmentItems returns inventory items for a profile in stable order.
func (s *Store) ListEquipmentItems(ctx context.Context, profileID string) ([]domain.EquipmentItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, profile_id, name, category, required, created_at, updated_at FROM equipment_items WHERE profile_id = ? ORDER BY category, name, id`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.EquipmentItem, 0)
	for rows.Next() {
		var item domain.EquipmentItem
		var required int
		var created, updated string
		if err := rows.Scan(&item.ID, &item.ProfileID, &item.Name, &item.Category, &required, &created, &updated); err != nil {
			return nil, err
		}
		item.Required = required == 1
		var err error
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return nil, err
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveAtlasCatalog replaces the active local Atlas catalog atomically.
func (s *Store) SaveAtlasCatalog(ctx context.Context, catalog atlas.Catalog) error {
	if strings.TrimSpace(catalog.Version) == "" || len(catalog.Locations) == 0 {
		return fmt.Errorf("atlas catalog version and locations are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `UPDATE atlas_catalogs SET active = 0`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO atlas_catalogs (version, imported_at, active) VALUES (?, ?, 1) ON CONFLICT(version) DO UPDATE SET imported_at=excluded.imported_at, active=1`, catalog.Version, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM atlas_locations WHERE catalog_version = ?`, catalog.Version); err != nil {
		return err
	}
	for _, location := range catalog.Locations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO atlas_locations (id, catalog_version, name, latitude, longitude, timezone, bortle_class, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, location.ID, catalog.Version, location.Name, location.Latitude, location.Longitude, location.Timezone, nullableFloat(location.BortleClass), location.Source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// LoadActiveAtlasCatalog loads the most recently imported active catalog.
func (s *Store) LoadActiveAtlasCatalog(ctx context.Context) (atlas.Catalog, error) {
	var version string
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM atlas_catalogs WHERE active = 1 ORDER BY imported_at DESC LIMIT 1`).Scan(&version); err != nil {
		return atlas.Catalog{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, latitude, longitude, timezone, bortle_class, source FROM atlas_locations WHERE catalog_version = ? ORDER BY name, id`, version)
	if err != nil {
		return atlas.Catalog{}, err
	}
	defer rows.Close()
	catalog := atlas.Catalog{Version: version, Locations: make([]atlas.Location, 0)}
	for rows.Next() {
		var location atlas.Location
		var bortle sql.NullFloat64
		if err := rows.Scan(&location.ID, &location.Name, &location.Latitude, &location.Longitude, &location.Timezone, &bortle, &location.Source); err != nil {
			return atlas.Catalog{}, err
		}
		location.BortleClass = floatPointer(bortle)
		catalog.Locations = append(catalog.Locations, location)
	}
	if err := rows.Err(); err != nil {
		return atlas.Catalog{}, err
	}
	if len(catalog.Locations) == 0 {
		return atlas.Catalog{}, fmt.Errorf("active atlas catalog %q has no locations", version)
	}
	return catalog, nil
}

// SaveMission inserts or updates a mission aggregate.
func (s *Store) SaveMission(ctx context.Context, mission domain.Mission) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO missions (id, name, status, launch_site_id, equipment_profile_id, planned_start, planned_end, started_at, completed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, status=excluded.status, launch_site_id=excluded.launch_site_id, equipment_profile_id=excluded.equipment_profile_id, planned_start=excluded.planned_start, planned_end=excluded.planned_end, started_at=excluded.started_at, completed_at=excluded.completed_at, updated_at=excluded.updated_at`, mission.ID, mission.Name, mission.Status, mission.LaunchSiteID, mission.EquipmentProfileID, nullableTime(mission.PlannedStart), nullableTime(mission.PlannedEnd), nullableTime(mission.StartedAt), nullableTime(mission.CompletedAt), mission.CreatedAt.UTC().Format(time.RFC3339), mission.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// FindMission loads a mission aggregate by stable ID.
func (s *Store) FindMission(ctx context.Context, id string) (domain.Mission, error) {
	var mission domain.Mission
	var plannedStart, plannedEnd, started, completed sql.NullString
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, status, launch_site_id, equipment_profile_id, planned_start, planned_end, started_at, completed_at, created_at, updated_at FROM missions WHERE id = ?`, id).Scan(&mission.ID, &mission.Name, &mission.Status, &mission.LaunchSiteID, &mission.EquipmentProfileID, &plannedStart, &plannedEnd, &started, &completed, &created, &updated)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.PlannedStart, err = parseNullableTime(plannedStart)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.PlannedEnd, err = parseNullableTime(plannedEnd)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.StartedAt, err = parseNullableTime(started)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.CompletedAt, err = parseNullableTime(completed)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return domain.Mission{}, err
	}
	mission.UpdatedAt, err = time.Parse(time.RFC3339, updated)
	return mission, err
}

// ListMissions returns durable missions in most-recently-planned order.
func (s *Store) ListMissions(ctx context.Context) ([]domain.Mission, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, status, launch_site_id, equipment_profile_id, planned_start, planned_end, started_at, completed_at, created_at, updated_at FROM missions ORDER BY COALESCE(planned_start, created_at) DESC, created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	missions := make([]domain.Mission, 0)
	for rows.Next() {
		var mission domain.Mission
		var plannedStart, plannedEnd, started, completed sql.NullString
		var created, updated string
		if err := rows.Scan(&mission.ID, &mission.Name, &mission.Status, &mission.LaunchSiteID, &mission.EquipmentProfileID, &plannedStart, &plannedEnd, &started, &completed, &created, &updated); err != nil {
			return nil, err
		}
		if mission.PlannedStart, err = parseNullableTime(plannedStart); err != nil {
			return nil, err
		}
		if mission.PlannedEnd, err = parseNullableTime(plannedEnd); err != nil {
			return nil, err
		}
		if mission.StartedAt, err = parseNullableTime(started); err != nil {
			return nil, err
		}
		if mission.CompletedAt, err = parseNullableTime(completed); err != nil {
			return nil, err
		}
		if mission.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
			return nil, err
		}
		if mission.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
			return nil, err
		}
		missions = append(missions, mission)
	}
	return missions, rows.Err()
}

// SaveObservation inserts or updates an observation.
func (s *Store) SaveObservation(ctx context.Context, observation domain.Observation) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO observations (id, mission_id, target_name, notes, observed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET target_name=excluded.target_name, notes=excluded.notes, observed_at=excluded.observed_at, updated_at=excluded.updated_at`, observation.ID, observation.MissionID, observation.TargetName, observation.Notes, nullableTime(observation.ObservedAt), observation.CreatedAt.UTC().Format(time.RFC3339), observation.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// ListObservations returns observations in recording order.
func (s *Store) ListObservations(ctx context.Context, missionID string) ([]domain.Observation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, mission_id, target_name, notes, observed_at, created_at, updated_at FROM observations WHERE mission_id = ? ORDER BY created_at, id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	observations := make([]domain.Observation, 0)
	for rows.Next() {
		var observation domain.Observation
		var observedAt sql.NullString
		var created, updated string
		if err := rows.Scan(&observation.ID, &observation.MissionID, &observation.TargetName, &observation.Notes, &observedAt, &created, &updated); err != nil {
			return nil, err
		}
		if observation.ObservedAt, err = parseNullableTime(observedAt); err != nil {
			return nil, err
		}
		if observation.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
			return nil, err
		}
		if observation.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}

// SaveDebrief inserts or updates the one debrief belonging to a mission.
func (s *Store) SaveDebrief(ctx context.Context, debrief domain.Debrief) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO debriefs (id, mission_id, summary, created_at, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(mission_id) DO UPDATE SET id=excluded.id, summary=excluded.summary, updated_at=excluded.updated_at`, debrief.ID, debrief.MissionID, debrief.Summary, debrief.CreatedAt.UTC().Format(time.RFC3339), debrief.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// FindDebrief loads the debrief for a mission.
func (s *Store) FindDebrief(ctx context.Context, missionID string) (domain.Debrief, error) {
	var debrief domain.Debrief
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, mission_id, summary, created_at, updated_at FROM debriefs WHERE mission_id = ?`, missionID).Scan(&debrief.ID, &debrief.MissionID, &debrief.Summary, &created, &updated)
	if err != nil {
		return domain.Debrief{}, err
	}
	if debrief.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return domain.Debrief{}, err
	}
	if debrief.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return domain.Debrief{}, err
	}
	return debrief, nil
}

// SaveWeatherSnapshot persists a validated weather result for offline use.
func (s *Store) SaveWeatherSnapshot(ctx context.Context, snapshot weather.Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	forecast, err := json.Marshal(snapshot.Forecast)
	if err != nil {
		return fmt.Errorf("encode weather forecast: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO weather_cache (location_key, source, observed_at, expires_at, temperature_c, cloud_cover_percent, payload, forecast_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(location_key) DO UPDATE SET source=excluded.source, observed_at=excluded.observed_at, expires_at=excluded.expires_at, temperature_c=excluded.temperature_c, cloud_cover_percent=excluded.cloud_cover_percent, payload=excluded.payload, forecast_json=excluded.forecast_json`, snapshot.LocationKey, snapshot.Source, snapshot.ObservedAt.UTC().Format(time.RFC3339), snapshot.ExpiresAt.UTC().Format(time.RFC3339), nullableFloat(snapshot.TemperatureC), nullableFloat(snapshot.CloudCoverPercent), snapshot.Payload, string(forecast))
	return err
}

// FindWeatherSnapshot loads a cached weather result by location key.
func (s *Store) FindWeatherSnapshot(ctx context.Context, locationKey string) (weather.Snapshot, error) {
	var snapshot weather.Snapshot
	var observed, expires string
	var temperature, cloud sql.NullFloat64
	var forecastJSON string
	err := s.db.QueryRowContext(ctx, `SELECT location_key, source, observed_at, expires_at, temperature_c, cloud_cover_percent, payload, forecast_json FROM weather_cache WHERE location_key = ?`, locationKey).Scan(&snapshot.LocationKey, &snapshot.Source, &observed, &expires, &temperature, &cloud, &snapshot.Payload, &forecastJSON)
	if err != nil {
		return weather.Snapshot{}, err
	}
	if snapshot.ObservedAt, err = time.Parse(time.RFC3339, observed); err != nil {
		return weather.Snapshot{}, err
	}
	if snapshot.ExpiresAt, err = time.Parse(time.RFC3339, expires); err != nil {
		return weather.Snapshot{}, err
	}
	snapshot.TemperatureC, snapshot.CloudCoverPercent = floatPointer(temperature), floatPointer(cloud)
	if forecastJSON != "" {
		if err := json.Unmarshal([]byte(forecastJSON), &snapshot.Forecast); err != nil {
			return weather.Snapshot{}, fmt.Errorf("decode cached weather forecast: %w", err)
		}
	}
	return snapshot, nil
}

// SaveRoute persists a validated provider route for offline reuse.
func (s *Store) SaveRoute(ctx context.Context, key string, plan routing.Plan, fetchedAt, expiresAt time.Time) error {
	if key == "" {
		return fmt.Errorf("route cache key is required")
	}
	if err := plan.Validate(); err != nil {
		return err
	}
	if expiresAt.Before(fetchedAt) {
		return fmt.Errorf("route cache expires before it was fetched")
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("encode route plan: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO route_cache (route_key, route_json, fetched_at, expires_at) VALUES (?, ?, ?, ?) ON CONFLICT(route_key) DO UPDATE SET route_json=excluded.route_json, fetched_at=excluded.fetched_at, expires_at=excluded.expires_at`, key, string(data), fetchedAt.UTC().Format(time.RFC3339), expiresAt.UTC().Format(time.RFC3339))
	return err
}

// FindRoute loads a cached provider route and its freshness timestamps.
func (s *Store) FindRoute(ctx context.Context, key string) (routing.Plan, time.Time, time.Time, error) {
	var data, fetched, expires string
	if err := s.db.QueryRowContext(ctx, `SELECT route_json, fetched_at, expires_at FROM route_cache WHERE route_key = ?`, key).Scan(&data, &fetched, &expires); err != nil {
		return routing.Plan{}, time.Time{}, time.Time{}, err
	}
	var plan routing.Plan
	if err := json.Unmarshal([]byte(data), &plan); err != nil {
		return routing.Plan{}, time.Time{}, time.Time{}, fmt.Errorf("decode cached route: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return routing.Plan{}, time.Time{}, time.Time{}, err
	}
	fetchedAt, err := time.Parse(time.RFC3339, fetched)
	if err != nil {
		return routing.Plan{}, time.Time{}, time.Time{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return routing.Plan{}, time.Time{}, time.Time{}, err
	}
	return plan, fetchedAt, expiresAt, nil
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
func floatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value.String)
	return &parsed, err
}

// Ping verifies that the local database is available.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Backup creates a consistent SQLite backup at path using an atomic replacement.
// The destination is replaced only after SQLite has successfully materialized the
// complete backup. Backup files are created with owner-only permissions.
func (s *Store) Backup(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("backup path is required")
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	database, databaseErr := filepath.Abs(s.path)
	if databaseErr == nil && target == database {
		return fmt.Errorf("backup path cannot be the active database")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	temporary := target + ".tmp"
	if err := os.Remove(temporary); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove incomplete backup: %w", err)
	}
	defer os.Remove(temporary)
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, temporary); err != nil {
		return fmt.Errorf("write sqlite backup: %w", err)
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		return fmt.Errorf("secure sqlite backup: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		return fmt.Errorf("replace sqlite backup: %w", err)
	}
	return nil
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }

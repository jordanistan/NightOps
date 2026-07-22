// Package sync defines a versioned, offline interchange bundle for future clients.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
)

// CurrentVersion is the supported portable bundle schema.
const CurrentVersion = 2

// Bundle is a portable snapshot of user-owned NightOps records.
type Bundle struct {
	SchemaVersion     int                       `json:"schema_version"`
	GeneratedAt       time.Time                 `json:"generated_at"`
	SourceDeviceID    string                    `json:"source_device_id,omitempty"`
	LaunchSites       []domain.LaunchSite       `json:"launch_sites"`
	Missions          []domain.Mission          `json:"missions"`
	Observations      []domain.Observation      `json:"observations"`
	EquipmentProfiles []domain.EquipmentProfile `json:"equipment_profiles"`
	EquipmentItems    []domain.EquipmentItem    `json:"equipment_items"`
}

// MergeStore is the storage boundary required to merge a validated bundle.
type MergeStore interface {
	FindLaunchSite(context.Context, string) (domain.LaunchSite, error)
	SaveLaunchSite(context.Context, domain.LaunchSite) error
	FindMission(context.Context, string) (domain.Mission, error)
	SaveMission(context.Context, domain.Mission) error
	ListObservations(context.Context, string) ([]domain.Observation, error)
	SaveObservation(context.Context, domain.Observation) error
	FindEquipmentProfile(context.Context, string) (domain.EquipmentProfile, error)
	SaveEquipmentProfile(context.Context, domain.EquipmentProfile) error
	ListEquipmentItems(context.Context, string) ([]domain.EquipmentItem, error)
	SaveEquipmentItem(context.Context, domain.EquipmentItem) error
}

// MergeReport records how many records were added, updated, or already current.
type MergeReport struct {
	Added       int      `json:"added"`
	Updated     int      `json:"updated"`
	Skipped     int      `json:"skipped"`
	Conflicts   int      `json:"conflicts"`
	ConflictIDs []string `json:"conflict_ids,omitempty"`
}

// NewBundle constructs a validated bundle from local records.
func NewBundle(now time.Time, sites []domain.LaunchSite, missions []domain.Mission, observations []domain.Observation, profiles []domain.EquipmentProfile, items []domain.EquipmentItem) (Bundle, error) {
	return NewBundleWithDevice("", now, sites, missions, observations, profiles, items)
}

// NewBundleWithDevice constructs a bundle with an explicit source identity.
func NewBundleWithDevice(sourceDeviceID string, now time.Time, sites []domain.LaunchSite, missions []domain.Mission, observations []domain.Observation, profiles []domain.EquipmentProfile, items []domain.EquipmentItem) (Bundle, error) {
	bundle := Bundle{SchemaVersion: CurrentVersion, GeneratedAt: now.UTC(), SourceDeviceID: strings.TrimSpace(sourceDeviceID), LaunchSites: sites, Missions: missions, Observations: observations, EquipmentProfiles: profiles, EquipmentItems: items}
	if err := bundle.Validate(); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// Validate checks schema, identity, timestamps, references, and mission statuses.
func (b Bundle) Validate() error {
	if b.SchemaVersion != 1 && b.SchemaVersion != CurrentVersion {
		return fmt.Errorf("unsupported sync bundle schema version %d", b.SchemaVersion)
	}
	if b.GeneratedAt.IsZero() {
		return fmt.Errorf("sync bundle generated_at is required")
	}
	if len(b.SourceDeviceID) > 128 || strings.TrimSpace(b.SourceDeviceID) != b.SourceDeviceID {
		return fmt.Errorf("sync bundle source_device_id is invalid")
	}
	if err := validateIDs("launch site", b.LaunchSites, func(site domain.LaunchSite) string { return site.ID }, func(site domain.LaunchSite) time.Time { return site.UpdatedAt }); err != nil {
		return err
	}
	if err := validateIDs("mission", b.Missions, func(mission domain.Mission) string { return mission.ID }, func(mission domain.Mission) time.Time { return mission.UpdatedAt }); err != nil {
		return err
	}
	for _, mission := range b.Missions {
		if !validMissionStatus(mission.Status) {
			return fmt.Errorf("mission %q has unsupported status %q", mission.ID, mission.Status)
		}
	}
	if err := validateIDs("observation", b.Observations, func(observation domain.Observation) string { return observation.ID }, func(observation domain.Observation) time.Time { return observation.UpdatedAt }); err != nil {
		return err
	}
	if err := validateIDs("equipment profile", b.EquipmentProfiles, func(profile domain.EquipmentProfile) string { return profile.ID }, func(profile domain.EquipmentProfile) time.Time { return profile.UpdatedAt }); err != nil {
		return err
	}
	if err := validateIDs("equipment item", b.EquipmentItems, func(item domain.EquipmentItem) string { return item.ID }, func(item domain.EquipmentItem) time.Time { return item.UpdatedAt }); err != nil {
		return err
	}
	return nil
}

// Save writes a bundle as owner-readable JSON with an atomic replacement.
func Save(path string, bundle Bundle) error {
	if err := bundle.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("sync bundle path is required")
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve sync bundle path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create sync bundle directory: %w", err)
	}
	temporary := target + ".tmp"
	if err := os.Remove(temporary); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove incomplete sync bundle: %w", err)
	}
	defer os.Remove(temporary)
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create sync bundle: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(bundle)
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("encode sync bundle: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close sync bundle: %w", closeErr)
	}
	if err := os.Rename(temporary, target); err != nil {
		return fmt.Errorf("replace sync bundle: %w", err)
	}
	return nil
}

// Load reads and validates a local sync bundle.
func Load(path string) (Bundle, error) {
	file, err := os.Open(path)
	if err != nil {
		return Bundle{}, err
	}
	defer file.Close()
	var bundle Bundle
	decoder := json.NewDecoder(io.LimitReader(file, 32<<20))
	if err := decoder.Decode(&bundle); err != nil {
		return Bundle{}, fmt.Errorf("decode sync bundle: %w", err)
	}
	if err := bundle.Validate(); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// Merge applies only records newer than the local record with the same stable ID.
func (b Bundle) Merge(ctx context.Context, store MergeStore) (MergeReport, error) {
	if err := b.Validate(); err != nil {
		return MergeReport{}, err
	}
	var report MergeReport
	for _, site := range b.LaunchSites {
		existing, err := store.FindLaunchSite(ctx, site.ID)
		if err != nil {
			if err := store.SaveLaunchSite(ctx, site); err != nil {
				return report, err
			}
			report.Added++
		} else if site.UpdatedAt.After(existing.UpdatedAt) {
			if err := store.SaveLaunchSite(ctx, site); err != nil {
				return report, err
			}
			report.Updated++
		} else if site.UpdatedAt.Equal(existing.UpdatedAt) && !reflect.DeepEqual(site, existing) {
			reportConflict(&report, "launch_site", site.ID)
			report.Skipped++
		} else {
			report.Skipped++
		}
	}
	for _, profile := range b.EquipmentProfiles {
		existing, err := store.FindEquipmentProfile(ctx, profile.ID)
		if err != nil {
			if err := store.SaveEquipmentProfile(ctx, profile); err != nil {
				return report, err
			}
			report.Added++
		} else if profile.UpdatedAt.After(existing.UpdatedAt) {
			if err := store.SaveEquipmentProfile(ctx, profile); err != nil {
				return report, err
			}
			report.Updated++
		} else if profile.UpdatedAt.Equal(existing.UpdatedAt) && !reflect.DeepEqual(profile, existing) {
			reportConflict(&report, "equipment_profile", profile.ID)
			report.Skipped++
		} else {
			report.Skipped++
		}
	}
	for _, item := range b.EquipmentItems {
		items, err := store.ListEquipmentItems(ctx, item.ProfileID)
		if err != nil {
			return report, err
		}
		var found *domain.EquipmentItem
		for index := range items {
			if items[index].ID == item.ID {
				found = &items[index]
				break
			}
		}
		if found == nil {
			if err := store.SaveEquipmentItem(ctx, item); err != nil {
				return report, err
			}
			report.Added++
		} else if item.UpdatedAt.After(found.UpdatedAt) {
			if err := store.SaveEquipmentItem(ctx, item); err != nil {
				return report, err
			}
			report.Updated++
		} else if item.UpdatedAt.Equal(found.UpdatedAt) && !reflect.DeepEqual(item, *found) {
			reportConflict(&report, "equipment_item", item.ID)
			report.Skipped++
		} else {
			report.Skipped++
		}
	}
	for _, mission := range b.Missions {
		existing, err := store.FindMission(ctx, mission.ID)
		if err != nil {
			if err := store.SaveMission(ctx, mission); err != nil {
				return report, err
			}
			report.Added++
		} else if mission.UpdatedAt.After(existing.UpdatedAt) {
			if err := store.SaveMission(ctx, mission); err != nil {
				return report, err
			}
			report.Updated++
		} else if mission.UpdatedAt.Equal(existing.UpdatedAt) && !reflect.DeepEqual(mission, existing) {
			reportConflict(&report, "mission", mission.ID)
			report.Skipped++
		} else {
			report.Skipped++
		}
	}
	for _, observation := range b.Observations {
		observations, err := store.ListObservations(ctx, observation.MissionID)
		if err != nil {
			return report, err
		}
		var found *domain.Observation
		for index := range observations {
			if observations[index].ID == observation.ID {
				found = &observations[index]
				break
			}
		}
		if found == nil {
			if err := store.SaveObservation(ctx, observation); err != nil {
				return report, err
			}
			report.Added++
		} else if observation.UpdatedAt.After(found.UpdatedAt) {
			if err := store.SaveObservation(ctx, observation); err != nil {
				return report, err
			}
			report.Updated++
		} else if observation.UpdatedAt.Equal(found.UpdatedAt) && !reflect.DeepEqual(observation, *found) {
			reportConflict(&report, "observation", observation.ID)
			report.Skipped++
		} else {
			report.Skipped++
		}
	}
	return report, nil
}

func reportConflict(report *MergeReport, kind, id string) {
	report.Conflicts++
	report.ConflictIDs = append(report.ConflictIDs, kind+":"+id)
}

func validMissionStatus(status domain.MissionStatus) bool {
	switch status {
	case domain.StatusDraft, domain.StatusPlanned, domain.StatusLaunched, domain.StatusActive, domain.StatusPaused, domain.StatusCompleted, domain.StatusCancelled, domain.StatusArchived:
		return true
	default:
		return false
	}
}

func validateIDs[T any](kind string, records []T, id func(T) string, updated func(T) time.Time) error {
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		value := id(record)
		if value == "" {
			return fmt.Errorf("%s ID is required", kind)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("duplicate %s ID %q", kind, value)
		}
		seen[value] = struct{}{}
		if updated(record).IsZero() {
			return fmt.Errorf("%s %q updated_at is required", kind, value)
		}
	}
	return nil
}

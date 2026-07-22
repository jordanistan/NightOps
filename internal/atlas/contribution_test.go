package atlas

import (
	"path/filepath"
	"testing"
	"time"
)

func TestContributionRoundTripsAndPreservesReviewStatus(t *testing.T) {
	catalog := Catalog{Version: "community-2026-07", Locations: []Location{{ID: "site-a", Name: "Dark Site", Latitude: 30.1, Longitude: -97.2, Timezone: "America/Chicago", Source: "community survey"}}}
	generated := time.Date(2026, 7, 22, 2, 0, 0, 0, time.FixedZone("CDT", -5*60*60))
	contribution, err := NewContributionPackage(catalog, generated)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "atlas-contribution.json")
	if err := SaveContribution(path, contribution); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadContribution(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Catalog.Version != catalog.Version || loaded.ReviewStatus != "unreviewed" || len(loaded.Catalog.Locations) != 1 || !loaded.GeneratedAt.Equal(generated.UTC()) {
		t.Fatalf("loaded contribution = %+v", loaded)
	}
}

func TestContributionRejectsInvalidEnvelope(t *testing.T) {
	catalog := Catalog{Version: "community-1", Locations: []Location{{ID: "site-a", Name: "Site", Latitude: 0, Longitude: 0, Timezone: "UTC", Source: "survey"}}}
	contribution, err := NewContributionPackage(catalog, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	contribution.SchemaVersion = 2
	if err := contribution.Validate(); err == nil {
		t.Fatal("unsupported contribution schema was accepted")
	}
	contribution.SchemaVersion = contributionSchemaVersion
	contribution.Catalog.Locations[0].Source = ""
	if err := contribution.Validate(); err == nil {
		t.Fatal("missing row provenance was accepted")
	}
}

package atlas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndQueryCatalog(t *testing.T) {
	catalog, err := ParseCSV(strings.NewReader("id,name,latitude,longitude,timezone,bortle_class,source\nsite-a,Dark Site,30.1,-97.2,America/Chicago,4,community\nsite-b,Far Site,31.5,-98.5,America/Chicago,,survey\n"), "atlas-2026.01")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Locations) != 2 || catalog.Version != "atlas-2026.01" {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	location, ok := catalog.Find("site-a")
	if !ok || location.BortleClass == nil || *location.BortleClass != 4 {
		t.Fatalf("catalog lookup failed: %+v", location)
	}
	if len(catalog.Nearby(30, -97, 1)) != 1 {
		t.Fatal("nearby query returned incorrect results")
	}
}

func TestParseRejectsInvalidCatalog(t *testing.T) {
	_, err := ParseCSV(strings.NewReader("id,name,latitude,longitude,timezone,bortle_class,source\ninvalid,Site,99,-97,UTC,,source\n"), "atlas-1")
	if err == nil {
		t.Fatal("expected invalid latitude error")
	}
}

func TestParseRejectsEmptyCatalog(t *testing.T) {
	_, err := ParseCSV(strings.NewReader("id,name,latitude,longitude,timezone,bortle_class,source\n"), "atlas-empty")
	if err == nil {
		t.Fatal("expected empty catalog error")
	}
}

func TestLoadEmbeddedAustin(t *testing.T) {
	catalog, err := LoadEmbeddedAustin()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Version != "austin-v1" || len(catalog.Locations) < 2 {
		t.Fatalf("unexpected embedded catalog: %+v", catalog)
	}
	for _, location := range catalog.Locations {
		if location.Source == "" || location.BortleClass == nil {
			t.Fatalf("embedded location is missing provenance or Bortle data: %+v", location)
		}
	}
}

func TestWriteCSVPreservesProvenanceAndRoundTrips(t *testing.T) {
	bortle := 4.5
	original := Catalog{Version: "community-v2", Locations: []Location{{ID: "site-a", Name: "Dark Site", Latitude: 30.1, Longitude: -97.2, Timezone: "America/Chicago", BortleClass: &bortle, Source: "community survey"}}}
	var output strings.Builder
	if err := WriteCSV(&output, original); err != nil {
		t.Fatal(err)
	}
	loaded, err := ParseCSV(strings.NewReader(output.String()), original.Version)
	if err != nil || len(loaded.Locations) != 1 || loaded.Locations[0].Source != "community survey" || loaded.Locations[0].BortleClass == nil {
		t.Fatalf("export did not round-trip provenance: %+v err=%v csv=%s", loaded, err, output.String())
	}
}

func TestSaveCSVUsesAtomicOwnerOnlyFile(t *testing.T) {
	catalog := Catalog{Version: "community-v2", Locations: []Location{{ID: "site-a", Name: "Dark Site", Latitude: 30.1, Longitude: -97.2, Timezone: "UTC", Source: "survey"}}}
	path := filepath.Join(t.TempDir(), "exports", "atlas.csv")
	if err := SaveCSV(path, catalog); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export permissions=%o", info.Mode().Perm())
	}
}

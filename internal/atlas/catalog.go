// Package atlas provides versioned, offline launch-site catalogs.
package atlas

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Location is a catalog entry with source provenance.
type Location struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Timezone    string   `json:"timezone"`
	BortleClass *float64 `json:"bortle_class,omitempty"`
	Source      string   `json:"source"`
}

// Catalog is an immutable in-memory view of a validated location dataset.
type Catalog struct {
	Version   string     `json:"version"`
	Locations []Location `json:"locations"`
}

// Validate checks the invariants required for a shareable catalog.
func (c Catalog) Validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("catalog version is required")
	}
	if len(c.Locations) == 0 {
		return fmt.Errorf("catalog must contain at least one location")
	}
	seen := make(map[string]struct{}, len(c.Locations))
	for index, location := range c.Locations {
		if location.ID == "" || location.Name == "" || location.Timezone == "" || location.Source == "" || location.Latitude < -90 || location.Latitude > 90 || location.Longitude < -180 || location.Longitude > 180 {
			return fmt.Errorf("atlas row %d contains invalid required values", index+2)
		}
		if _, ok := seen[location.ID]; ok {
			return fmt.Errorf("atlas row %d duplicates location ID %q", index+2, location.ID)
		}
		seen[location.ID] = struct{}{}
		if location.BortleClass != nil && (*location.BortleClass < 1 || *location.BortleClass > 9) {
			return fmt.Errorf("atlas row %d Bortle class is invalid", index+2)
		}
	}
	return nil
}

// LoadCSV loads a catalog whose first row is the required header.
func LoadCSV(path, version string) (Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return Catalog{}, err
	}
	defer file.Close()
	return ParseCSV(file, version)
}

// ParseCSV validates and loads CSV data from a reader.
func ParseCSV(reader io.Reader, version string) (Catalog, error) {
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return Catalog{}, fmt.Errorf("read atlas CSV: %w", err)
	}
	if len(rows) < 2 || strings.Join(rows[0], ",") != "id,name,latitude,longitude,timezone,bortle_class,source" {
		return Catalog{}, fmt.Errorf("atlas CSV header is invalid")
	}
	catalog := Catalog{Version: version, Locations: make([]Location, 0, len(rows)-1)}
	for index, row := range rows[1:] {
		if len(row) != 7 {
			return Catalog{}, fmt.Errorf("atlas row %d has %d columns, expected 7", index+2, len(row))
		}
		latitude, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return Catalog{}, fmt.Errorf("atlas row %d latitude: %w", index+2, err)
		}
		longitude, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return Catalog{}, fmt.Errorf("atlas row %d longitude: %w", index+2, err)
		}
		location := Location{ID: row[0], Name: row[1], Latitude: latitude, Longitude: longitude, Timezone: row[4], Source: row[6]}
		if row[5] != "" {
			bortle, parseErr := strconv.ParseFloat(row[5], 64)
			if parseErr != nil {
				return Catalog{}, fmt.Errorf("atlas row %d Bortle class is invalid", index+2)
			}
			location.BortleClass = &bortle
		}
		catalog.Locations = append(catalog.Locations, location)
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

// WriteCSV writes a validated catalog with its source provenance intact.
func WriteCSV(writer io.Writer, catalog Catalog) error {
	if err := catalog.Validate(); err != nil {
		return err
	}
	output := csv.NewWriter(writer)
	if err := output.Write([]string{"id", "name", "latitude", "longitude", "timezone", "bortle_class", "source"}); err != nil {
		return fmt.Errorf("write atlas header: %w", err)
	}
	for _, location := range catalog.Locations {
		bortle := ""
		if location.BortleClass != nil {
			bortle = strconv.FormatFloat(*location.BortleClass, 'f', -1, 64)
		}
		if err := output.Write([]string{location.ID, location.Name, strconv.FormatFloat(location.Latitude, 'f', -1, 64), strconv.FormatFloat(location.Longitude, 'f', -1, 64), location.Timezone, bortle, location.Source}); err != nil {
			return fmt.Errorf("write atlas location %q: %w", location.ID, err)
		}
	}
	output.Flush()
	if err := output.Error(); err != nil {
		return fmt.Errorf("flush atlas CSV: %w", err)
	}
	return nil
}

// SaveCSV writes a validated catalog to a local path using an atomic replacement.
func SaveCSV(path string, catalog Catalog) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("atlas export path is required")
	}
	if err := catalog.Validate(); err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve atlas export path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create atlas export directory: %w", err)
	}
	temporary := target + ".tmp"
	if err := os.Remove(temporary); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove incomplete atlas export: %w", err)
	}
	defer os.Remove(temporary)
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create atlas export: %w", err)
	}
	writeErr := WriteCSV(file, catalog)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return fmt.Errorf("close atlas export: %w", closeErr)
	}
	if err := os.Rename(temporary, target); err != nil {
		return fmt.Errorf("replace atlas export: %w", err)
	}
	return nil
}

//go:embed data/austin-v1.csv
var embeddedCatalog embed.FS

// LoadEmbeddedAustin loads the provenance-backed Austin starter catalog.
func LoadEmbeddedAustin() (Catalog, error) {
	file, err := embeddedCatalog.Open("data/austin-v1.csv")
	if err != nil {
		return Catalog{}, err
	}
	defer file.Close()
	return ParseCSV(file, "austin-v1")
}

// Find returns a location by stable catalog ID.
func (c Catalog) Find(id string) (Location, bool) {
	for _, location := range c.Locations {
		if location.ID == id {
			return location, true
		}
	}
	return Location{}, false
}

// Nearby returns locations within an approximate latitude/longitude radius.
func (c Catalog) Nearby(latitude, longitude, radiusDegrees float64) []Location {
	nearby := make([]Location, 0)
	for _, location := range c.Locations {
		if abs(location.Latitude-latitude) <= radiusDegrees && abs(location.Longitude-longitude) <= radiusDegrees {
			nearby = append(nearby, location)
		}
	}
	return nearby
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

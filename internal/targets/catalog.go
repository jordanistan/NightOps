// Package targets contains the versioned offline celestial target catalog.
package targets

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jordanistan/nightops/internal/astronomy"
)

//go:embed data/targets-v1.csv
var embeddedFS embed.FS

// Target is a cataloged fixed celestial object.
type Target struct {
	ID             string
	Name           string
	Kind           string
	RightAscension float64
	Declination    float64
	Source         string
}

// Validate checks identity, coordinates, and provenance.
func (t Target) Validate() error {
	if t.ID == "" || t.Name == "" || t.Kind == "" || t.Source == "" {
		return fmt.Errorf("target requires id, name, kind, and source")
	}
	return astronomy.TargetCoordinates{RightAscension: t.RightAscension, Declination: t.Declination}.Validate()
}

// Coordinates returns the target's equatorial coordinate pair.
func (t Target) Coordinates() astronomy.TargetCoordinates {
	return astronomy.TargetCoordinates{RightAscension: t.RightAscension, Declination: t.Declination}
}

// Catalog is a versioned collection of fixed targets.
type Catalog struct {
	Version string
	Targets []Target
}

// LoadEmbedded returns the built-in source-attributed target catalog.
func LoadEmbedded() (Catalog, error) {
	data, err := embeddedFS.Open("data/targets-v1.csv")
	if err != nil {
		return Catalog{}, err
	}
	defer data.Close()
	return ParseCSV(data, "targets-v1")
}

// ParseCSV parses a target catalog with a stable header.
func ParseCSV(reader io.Reader, version string) (Catalog, error) {
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return Catalog{}, fmt.Errorf("read target catalog: %w", err)
	}
	if len(rows) < 2 || strings.Join(rows[0], ",") != "id,name,kind,right_ascension_deg,declination_deg,source" {
		return Catalog{}, fmt.Errorf("target catalog has invalid header")
	}
	catalog := Catalog{Version: version, Targets: make([]Target, 0, len(rows)-1)}
	for index, row := range rows[1:] {
		if len(row) != 6 {
			return Catalog{}, fmt.Errorf("target row %d has %d fields", index+2, len(row))
		}
		ra, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return Catalog{}, fmt.Errorf("target row %d right ascension: %w", index+2, err)
		}
		dec, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			return Catalog{}, fmt.Errorf("target row %d declination: %w", index+2, err)
		}
		target := Target{ID: row[0], Name: row[1], Kind: row[2], RightAscension: ra, Declination: dec, Source: row[5]}
		if err := target.Validate(); err != nil {
			return Catalog{}, fmt.Errorf("target row %d: %w", index+2, err)
		}
		catalog.Targets = append(catalog.Targets, target)
	}
	return catalog, nil
}

// Find returns a target by stable catalog ID.
func (c Catalog) Find(id string) (Target, bool) {
	for _, target := range c.Targets {
		if target.ID == id {
			return target, true
		}
	}
	return Target{}, false
}

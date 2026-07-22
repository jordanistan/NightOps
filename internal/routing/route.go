// Package routing provides offline route facts without pretending to be a
// turn-by-turn navigation service.
package routing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jordanistan/nightops/internal/astronomy"
)

// Plan is the honest route information available without a routing provider.
type Plan struct {
	FromLabel  string
	ToLabel    string
	DistanceKm float64
	DistanceMi float64
	TravelTime string
	Provider   string
}

// Provider supplies road-network route facts from an optional routing service.
type Provider interface {
	Route(context.Context, astronomy.Coordinates, astronomy.Coordinates) (Plan, error)
}

// Cache persists provider route results for offline reuse.
type Cache interface {
	SaveRoute(context.Context, string, Plan, time.Time, time.Time) error
	FindRoute(context.Context, string) (Plan, time.Time, time.Time, error)
}

// Key returns a stable cache key for a coordinate pair.
func Key(from, to astronomy.Coordinates) string {
	return formatCoordinate(from) + "->" + formatCoordinate(to)
}

// Build calculates a straight-line distance between two coordinate-backed
// places. Driving distance and time remain unavailable by design.
func Build(fromLabel string, from astronomy.Coordinates, toLabel string, to astronomy.Coordinates) (Plan, error) {
	if fromLabel == "" || toLabel == "" {
		return Plan{}, errors.New("route endpoints require labels")
	}
	if err := from.Validate(); err != nil {
		return Plan{}, err
	}
	if err := to.Validate(); err != nil {
		return Plan{}, err
	}
	distanceKm := haversine(from, to)
	return Plan{FromLabel: fromLabel, ToLabel: toLabel, DistanceKm: distanceKm, DistanceMi: distanceKm * 0.621371, TravelTime: "unavailable without a routing provider", Provider: "offline geodesic"}, nil
}

// Validate checks route facts before they are displayed or persisted.
func (p Plan) Validate() error {
	if p.FromLabel == "" || p.ToLabel == "" || p.Provider == "" {
		return errors.New("route plan requires endpoints and provider")
	}
	if p.DistanceKm < 0 || p.DistanceMi < 0 {
		return errors.New("route distance cannot be negative")
	}
	if p.TravelTime == "" {
		return errors.New("route plan requires travel time state")
	}
	return nil
}

func formatCoordinate(coordinates astronomy.Coordinates) string {
	return fmt.Sprintf("%.6f,%.6f", coordinates.Latitude, coordinates.Longitude)
}

func haversine(from, to astronomy.Coordinates) float64 {
	const earthRadiusKm = 6371.0088
	latitudeDelta := (to.Latitude - from.Latitude) * math.Pi / 180
	longitudeDelta := (to.Longitude - from.Longitude) * math.Pi / 180
	fromLatitude := from.Latitude * math.Pi / 180
	toLatitude := to.Latitude * math.Pi / 180
	a := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) + math.Cos(fromLatitude)*math.Cos(toLatitude)*math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

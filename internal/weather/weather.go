// Package weather defines offline-safe weather snapshots and provider boundaries.
package weather

import (
	"context"
	"errors"
	"time"
)

// Snapshot is a provider result retained for offline display.
type Snapshot struct {
	LocationKey       string
	Source            string
	ObservedAt        time.Time
	ExpiresAt         time.Time
	TemperatureC      *float64
	CloudCoverPercent *float64
	Forecast          []ForecastPoint
	Payload           string
}

// ForecastPoint is one provider-reported hourly forecast value.
type ForecastPoint struct {
	At                       time.Time
	TemperatureC             *float64
	CloudCoverPercent        *float64
	PrecipitationProbability *float64
	// Dark is calculated locally from astronomy and is not provider data.
	Dark bool
}

// Validate checks provenance and freshness metadata before persistence.
func (s Snapshot) Validate() error {
	if s.LocationKey == "" || s.Source == "" || s.ObservedAt.IsZero() || s.ExpiresAt.IsZero() {
		return errors.New("weather snapshot requires location, source, and timestamps")
	}
	if s.ExpiresAt.Before(s.ObservedAt) {
		return errors.New("weather snapshot expires before it was observed")
	}
	if s.CloudCoverPercent != nil && (*s.CloudCoverPercent < 0 || *s.CloudCoverPercent > 100) {
		return errors.New("cloud cover must be between 0 and 100 percent")
	}
	previous := time.Time{}
	for _, point := range s.Forecast {
		if point.At.IsZero() {
			return errors.New("weather forecast point requires a timestamp")
		}
		if !previous.IsZero() && !point.At.After(previous) {
			return errors.New("weather forecast points must be in ascending order")
		}
		if point.CloudCoverPercent != nil && (*point.CloudCoverPercent < 0 || *point.CloudCoverPercent > 100) {
			return errors.New("forecast cloud cover must be between 0 and 100 percent")
		}
		if point.PrecipitationProbability != nil && (*point.PrecipitationProbability < 0 || *point.PrecipitationProbability > 100) {
			return errors.New("precipitation probability must be between 0 and 100 percent")
		}
		previous = point.At
	}
	return nil
}

// Fresh reports whether the snapshot is valid at a given instant.
func (s Snapshot) Fresh(at time.Time) bool { return s.Validate() == nil && at.Before(s.ExpiresAt) }

// Provider retrieves a snapshot from a real weather source.
type Provider interface {
	Fetch(context.Context, float64, float64) (Snapshot, error)
}

// ForecastProvider retrieves hourly forecast points in addition to current conditions.
type ForecastProvider interface {
	FetchForecast(context.Context, float64, float64) (Snapshot, error)
}

// Cache persists weather snapshots for offline use.
type Cache interface {
	SaveWeatherSnapshot(context.Context, Snapshot) error
	FindWeatherSnapshot(context.Context, string) (Snapshot, error)
}

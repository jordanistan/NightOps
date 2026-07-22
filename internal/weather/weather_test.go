package weather

import (
	"testing"
	"time"
)

func TestSnapshotFreshnessAndValidation(t *testing.T) {
	observed := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	snapshot := Snapshot{LocationKey: "30.2,-97.7", Source: "fixture-provider", ObservedAt: observed, ExpiresAt: observed.Add(time.Hour)}
	if !snapshot.Fresh(observed.Add(30*time.Minute)) || snapshot.Fresh(observed.Add(2*time.Hour)) {
		t.Fatal("weather freshness is incorrect")
	}
	invalid := snapshot
	invalid.ExpiresAt = observed.Add(-time.Minute)
	if invalid.Validate() == nil {
		t.Fatal("expected invalid expiry error")
	}
}

func TestForecastValidationRequiresAscendingTimes(t *testing.T) {
	now := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	snapshot := Snapshot{LocationKey: "site-a", Source: "fixture", ObservedAt: now, ExpiresAt: now.Add(time.Hour), Forecast: []ForecastPoint{{At: now.Add(time.Hour)}, {At: now}}}
	if err := snapshot.Validate(); err == nil {
		t.Fatal("expected forecast ordering error")
	}
	probability := 101.0
	snapshot.Forecast = []ForecastPoint{{At: now.Add(time.Hour), PrecipitationProbability: &probability}}
	if err := snapshot.Validate(); err == nil {
		t.Fatal("expected precipitation probability error")
	}
}

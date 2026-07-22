package weather

import (
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/astronomy"
)

func TestRankTargetForecastPrefersClearEligibleWindow(t *testing.T) {
	location := astronomy.Coordinates{Latitude: 30.2672, Longitude: -97.7431}
	target := astronomy.TargetCoordinates{RightAscension: 10, Declination: 20}
	at := time.Date(2026, 7, 22, 4, 0, 0, 0, time.UTC)
	position, err := astronomy.TargetPositionAt(at, location, target)
	if err != nil {
		t.Fatal(err)
	}
	minimum := position.Altitude - 1
	clear, cloudy, precipitation := 10.0, 80.0, 5.0
	ranked, err := RankTargetForecast([]ForecastPoint{
		{At: at, CloudCoverPercent: &cloudy, PrecipitationProbability: &precipitation, Dark: true},
		{At: at.Add(time.Hour), CloudCoverPercent: &clear, PrecipitationProbability: &precipitation, Dark: true},
	}, location, target, RankingConfig{MinimumAltitude: minimum, MaximumCloudCover: 50, MaximumPrecipitation: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 2 || !ranked[0].Eligible || ranked[0].Point.At != at.Add(time.Hour) {
		t.Fatalf("clear eligible window was not ranked first: %+v", ranked)
	}
	if ranked[1].Eligible || ranked[1].Reason != "cloud cover exceeds threshold" {
		t.Fatalf("cloud threshold was not explained: %+v", ranked[1])
	}
}

func TestRankTargetForecastDoesNotInventMissingWeather(t *testing.T) {
	location := astronomy.Coordinates{Latitude: 30.2672, Longitude: -97.7431}
	target := astronomy.TargetCoordinates{RightAscension: 10, Declination: 20}
	ranked, err := RankTargetForecast([]ForecastPoint{{At: time.Date(2026, 7, 22, 4, 0, 0, 0, time.UTC), Dark: true}}, location, target, RankingConfig{MinimumAltitude: -90, MaximumCloudCover: 50, MaximumPrecipitation: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 1 || ranked[0].Eligible || !strings.Contains(ranked[0].Reason, "cloud cover unavailable") {
		t.Fatalf("missing weather was treated as usable: %+v", ranked)
	}
}

func TestRankTargetForecastValidatesThresholds(t *testing.T) {
	_, err := RankTargetForecast(nil, astronomy.Coordinates{Latitude: 0, Longitude: 0}, astronomy.TargetCoordinates{RightAscension: 0, Declination: 0}, RankingConfig{MaximumCloudCover: 101, MaximumPrecipitation: 20})
	if err == nil {
		t.Fatal("expected invalid ranking threshold error")
	}
}

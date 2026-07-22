package routing

import (
	"strings"
	"testing"

	"github.com/jordanistan/nightops/internal/astronomy"
)

func TestBuildReturnsOfflineDistanceAndHonestTravelTime(t *testing.T) {
	plan, err := Build("Austin", astronomy.Coordinates{Latitude: 30.2672, Longitude: -97.7431}, "Inks Lake", astronomy.Coordinates{Latitude: 30.7373, Longitude: -98.3690})
	if err != nil {
		t.Fatal(err)
	}
	if plan.DistanceKm <= 0 || plan.DistanceMi <= 0 || plan.Provider != "offline geodesic" || !strings.Contains(plan.TravelTime, "unavailable") {
		t.Fatalf("unexpected route plan: %+v", plan)
	}
}

func TestBuildRejectsInvalidEndpoints(t *testing.T) {
	if _, err := Build("", astronomy.Coordinates{}, "Site", astronomy.Coordinates{}); err == nil {
		t.Fatal("expected missing endpoint label error")
	}
	if _, err := Build("Home", astronomy.Coordinates{Latitude: 91}, "Site", astronomy.Coordinates{}); err == nil {
		t.Fatal("expected invalid origin coordinates error")
	}
}

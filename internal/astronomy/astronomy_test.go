package astronomy

import (
	"math"
	"testing"
	"time"
)

func TestCalculateIsDeterministic(t *testing.T) {
	at := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	coordinates := Coordinates{Latitude: 30.2672, Longitude: -97.7431}
	first, err := Calculate(at, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Calculate(at, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("calculation changed between identical inputs: %+v != %+v", first, second)
	}
	if first.MoonAgeDays < 0 || first.MoonAgeDays >= SynodicMonth || first.MoonIllumination < 0 || first.MoonIllumination > 1 {
		t.Fatalf("invalid lunar result: %+v", first)
	}
	if math.Abs(first.SunAltitude) > 90 || first.SunAzimuth < 0 || first.SunAzimuth >= 360 {
		t.Fatalf("invalid solar result: %+v", first)
	}
}

func TestInvalidCoordinates(t *testing.T) {
	if _, err := Calculate(time.Now(), Coordinates{Latitude: 91}); err == nil {
		t.Fatal("expected invalid coordinate error")
	}
}

func TestCalculateVisibilityFindsTwilightTransitions(t *testing.T) {
	day := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	visibility, err := CalculateVisibility(day, Coordinates{Latitude: 30.2672, Longitude: -97.7431})
	if err != nil {
		t.Fatal(err)
	}
	if visibility.NoAstronomicalNight || visibility.AstronomicalDusk == nil || visibility.AstronomicalDawn == nil {
		t.Fatalf("expected astronomical night in Austin: %+v", visibility)
	}
	if !visibility.AstronomicalDusk.Before(*visibility.AstronomicalDawn) {
		t.Fatalf("twilight ordering invalid: dusk=%v dawn=%v", *visibility.AstronomicalDusk, *visibility.AstronomicalDawn)
	}
	if visibility.CivilDusk == nil || visibility.NauticalDusk == nil || !visibility.CivilDusk.Before(*visibility.NauticalDusk) || !visibility.NauticalDusk.Before(*visibility.AstronomicalDusk) {
		t.Fatalf("dusk ordering invalid: %+v", visibility)
	}
}

func TestTargetVisibilityUsesDarknessAndAltitude(t *testing.T) {
	day := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	visibility, err := CalculateTargetVisibility(day, Coordinates{Latitude: 30.2672, Longitude: -97.7431}, TargetCoordinates{RightAscension: 10.684583, Declination: 41.269278}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if visibility.NoDarkWindow || visibility.Start == nil || visibility.End == nil || visibility.BestAt == nil {
		t.Fatalf("expected M31 visibility window: %+v", visibility)
	}
	if !visibility.Start.Before(*visibility.End) || visibility.MaximumAltitude < 30 {
		t.Fatalf("invalid target visibility window: %+v", visibility)
	}
	if _, err := CalculateTargetVisibility(day, Coordinates{Latitude: 91}, TargetCoordinates{}, 30); err == nil {
		t.Fatal("expected invalid location error")
	}
}

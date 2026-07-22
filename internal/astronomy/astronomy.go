// Package astronomy provides deterministic, offline celestial calculations.
package astronomy

import (
	"errors"
	"math"
	"time"
)

const (
	degrees = math.Pi / 180
	// SynodicMonth is the mean lunar phase cycle in days.
	SynodicMonth = 29.530588853
)

// Coordinates identifies a point on Earth in decimal degrees.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// Validate checks coordinate ranges before calculations are performed.
func (c Coordinates) Validate() error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90 degrees")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180 degrees")
	}
	return nil
}

// SolarPosition contains the apparent Sun position for a time and location.
type SolarPosition struct {
	Altitude float64
	Azimuth  float64
}

// Snapshot is a reproducible offline astronomy calculation result.
type Snapshot struct {
	At               time.Time
	Coordinates      Coordinates
	SunAltitude      float64
	SunAzimuth       float64
	MoonAgeDays      float64
	MoonIllumination float64
}

// Visibility contains the approximate twilight transitions surrounding a UTC day.
// A nil transition means the requested solar altitude was not crossed in the
// search window, which is expected at some polar latitudes and seasons.
type Visibility struct {
	CivilDusk           *time.Time
	NauticalDusk        *time.Time
	AstronomicalDusk    *time.Time
	AstronomicalDawn    *time.Time
	NauticalDawn        *time.Time
	CivilDawn           *time.Time
	NoAstronomicalNight bool
}

// TargetCoordinates identifies a fixed celestial target in equatorial degrees.
type TargetCoordinates struct {
	RightAscension float64
	Declination    float64
}

// Validate checks a fixed target coordinate pair.
func (t TargetCoordinates) Validate() error {
	if t.RightAscension < 0 || t.RightAscension >= 360 {
		return errors.New("right ascension must be between 0 and 360 degrees")
	}
	if t.Declination < -90 || t.Declination > 90 {
		return errors.New("declination must be between -90 and 90 degrees")
	}
	return nil
}

// TargetPosition contains an apparent altitude and azimuth for a target.
type TargetPosition struct {
	Altitude float64
	Azimuth  float64
}

// TargetVisibility describes the first dark, above-altitude interval found in
// the 36-hour search window around at. Nil times mean no such interval exists.
type TargetVisibility struct {
	Start           *time.Time
	End             *time.Time
	BestAt          *time.Time
	MaximumAltitude float64
	NoDarkWindow    bool
}

// CalculateVisibility finds civil, nautical, and astronomical twilight
// transitions around the UTC day containing at. It uses the same deterministic
// solar model as Calculate and performs a binary search at each crossing.
func CalculateVisibility(at time.Time, coordinates Coordinates) (Visibility, error) {
	if err := coordinates.Validate(); err != nil {
		return Visibility{}, err
	}
	start := at.UTC().Truncate(24 * time.Hour)
	end := start.Add(36 * time.Hour)
	visibility := Visibility{}
	visibility.CivilDusk, visibility.CivilDawn = twilightPair(start, end, coordinates, -6)
	visibility.NauticalDusk, visibility.NauticalDawn = twilightPair(start, end, coordinates, -12)
	visibility.AstronomicalDusk, visibility.AstronomicalDawn = twilightPair(start, end, coordinates, -18)
	visibility.NoAstronomicalNight = visibility.AstronomicalDusk == nil || visibility.AstronomicalDawn == nil
	return visibility, nil
}

// TargetPositionAt calculates a fixed target's apparent horizontal position.
func TargetPositionAt(at time.Time, location Coordinates, target TargetCoordinates) (TargetPosition, error) {
	if err := location.Validate(); err != nil {
		return TargetPosition{}, err
	}
	if err := target.Validate(); err != nil {
		return TargetPosition{}, err
	}
	position := equatorialPosition(at.UTC(), location, target)
	return position, nil
}

// CalculateTargetVisibility finds a target's dark observing interval above a
// minimum altitude. The darkness threshold is astronomical twilight (-18°).
func CalculateTargetVisibility(at time.Time, location Coordinates, target TargetCoordinates, minimumAltitude float64) (TargetVisibility, error) {
	if err := location.Validate(); err != nil {
		return TargetVisibility{}, err
	}
	if err := target.Validate(); err != nil {
		return TargetVisibility{}, err
	}
	if minimumAltitude < -90 || minimumAltitude > 90 {
		return TargetVisibility{}, errors.New("minimum target altitude must be between -90 and 90 degrees")
	}
	start := at.UTC().Truncate(24 * time.Hour)
	end := start.Add(36 * time.Hour)
	const step = 10 * time.Minute
	var windowStart, windowEnd, bestAt *time.Time
	maximum := -90.0
	bestAltitude := -90.0
	previous := start
	previousVisible := targetVisible(previous, location, target, minimumAltitude)
	if previousVisible {
		value := start
		windowStart = &value
	}
	for current := start.Add(step); !current.After(end); current = current.Add(step) {
		position := equatorialPosition(current, location, target)
		if solarPosition(current, location).Altitude <= -18 && position.Altitude > maximum {
			maximum = position.Altitude
		}
		visible := position.Altitude >= minimumAltitude && solarPosition(current, location).Altitude <= -18
		if visible && position.Altitude > bestAltitude {
			value := current
			bestAt, bestAltitude = &value, position.Altitude
		}
		if windowStart == nil && !previousVisible && visible {
			value := current
			windowStart = &value
		}
		if windowStart != nil && windowEnd == nil && previousVisible && !visible {
			value := current
			windowEnd = &value
			break
		}
		previous, previousVisible = current, visible
	}
	_ = previous
	result := TargetVisibility{Start: windowStart, End: windowEnd, BestAt: bestAt, MaximumAltitude: maximum}
	result.NoDarkWindow = windowStart == nil || windowEnd == nil
	return result, nil
}

// Calculate returns solar position and lunar phase data without network access.
func Calculate(at time.Time, coordinates Coordinates) (Snapshot, error) {
	if err := coordinates.Validate(); err != nil {
		return Snapshot{}, err
	}
	at = at.UTC()
	sun := solarPosition(at, coordinates)
	age := moonAge(at)
	phase := 2 * math.Pi * age / SynodicMonth
	return Snapshot{At: at, Coordinates: coordinates, SunAltitude: sun.Altitude, SunAzimuth: sun.Azimuth, MoonAgeDays: age, MoonIllumination: (1 - math.Cos(phase)) / 2}, nil
}

func solarPosition(at time.Time, coordinates Coordinates) SolarPosition {
	jd := float64(at.Unix())/86400 + 2440587.5
	days := jd - 2451545.0
	meanLongitude := normalize(280.460 + 0.9856474*days)
	meanAnomaly := normalize(357.528+0.9856003*days) * degrees
	eclipticLongitude := normalize(meanLongitude+1.915*math.Sin(meanAnomaly)+0.020*math.Sin(2*meanAnomaly)) * degrees
	obliquity := (23.439 - 0.0000004*days) * degrees
	ra := math.Atan2(math.Cos(obliquity)*math.Sin(eclipticLongitude), math.Cos(eclipticLongitude))
	declination := math.Asin(math.Sin(obliquity) * math.Sin(eclipticLongitude))
	gmst := normalize(280.46061837+360.98564736629*days) * degrees
	hourAngle := normalizeRadians(gmst + coordinates.Longitude*degrees - ra)
	latitude := coordinates.Latitude * degrees
	altitude := math.Asin(math.Sin(latitude)*math.Sin(declination) + math.Cos(latitude)*math.Cos(declination)*math.Cos(hourAngle))
	azimuth := math.Atan2(math.Sin(hourAngle), math.Cos(hourAngle)*math.Sin(latitude)-math.Tan(declination)*math.Cos(latitude)) + math.Pi
	return SolarPosition{Altitude: altitude / degrees, Azimuth: normalize(azimuth / degrees)}
}

func equatorialPosition(at time.Time, location Coordinates, target TargetCoordinates) TargetPosition {
	jd := float64(at.Unix())/86400 + 2440587.5
	days := jd - 2451545.0
	gmst := normalize(280.46061837 + 360.98564736629*days)
	hourAngle := normalizeRadians((gmst + location.Longitude - target.RightAscension) * degrees)
	latitude := location.Latitude * degrees
	declination := target.Declination * degrees
	altitude := math.Asin(math.Sin(latitude)*math.Sin(declination) + math.Cos(latitude)*math.Cos(declination)*math.Cos(hourAngle))
	azimuth := math.Atan2(math.Sin(hourAngle), math.Cos(hourAngle)*math.Sin(latitude)-math.Tan(declination)*math.Cos(latitude)) + math.Pi
	return TargetPosition{Altitude: altitude / degrees, Azimuth: normalize(azimuth / degrees)}
}

func targetVisible(at time.Time, location Coordinates, target TargetCoordinates, minimumAltitude float64) bool {
	position := equatorialPosition(at, location, target)
	return position.Altitude >= minimumAltitude && solarPosition(at, location).Altitude <= -18
}

func twilightPair(start, end time.Time, coordinates Coordinates, altitude float64) (*time.Time, *time.Time) {
	const step = 10 * time.Minute
	var dusk, dawn *time.Time
	previousTime := start
	previousAltitude := solarPosition(previousTime, coordinates).Altitude
	for current := start.Add(step); !current.After(end); current = current.Add(step) {
		currentAltitude := solarPosition(current, coordinates).Altitude
		if dusk == nil && previousAltitude >= altitude && currentAltitude < altitude {
			crossing := crossingTime(previousTime, current, previousAltitude, currentAltitude, altitude, coordinates)
			dusk = &crossing
		} else if dusk != nil && dawn == nil && previousAltitude < altitude && currentAltitude >= altitude {
			crossing := crossingTime(previousTime, current, previousAltitude, currentAltitude, altitude, coordinates)
			dawn = &crossing
			break
		}
		previousTime, previousAltitude = current, currentAltitude
	}
	return dusk, dawn
}

func crossingTime(low, high time.Time, lowAltitude, highAltitude, threshold float64, coordinates Coordinates) time.Time {
	for range 10 {
		mid := low.Add(high.Sub(low) / 2)
		if solarPosition(mid, coordinates).Altitude < threshold {
			high = mid
		} else {
			low = mid
		}
	}
	_ = lowAltitude
	_ = highAltitude
	return low.Add(high.Sub(low) / 2)
}

func moonAge(at time.Time) float64 {
	const newMoon = 947182440.0 // 2000-01-06 18:14 UTC, Unix seconds.
	age := math.Mod((float64(at.Unix())-newMoon)/86400, SynodicMonth)
	if age < 0 {
		age += SynodicMonth
	}
	return age
}

func normalize(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	return value
}

func normalizeRadians(value float64) float64 {
	value = math.Mod(value+math.Pi, 2*math.Pi)
	if value < 0 {
		value += 2 * math.Pi
	}
	return value - math.Pi
}

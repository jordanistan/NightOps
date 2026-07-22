package weather

import (
	"errors"
	"sort"

	"github.com/jordanistan/nightops/internal/astronomy"
)

// RankingConfig defines the minimum target altitude and maximum weather values
// that qualify a forecast point as an observing window.
type RankingConfig struct {
	MinimumAltitude      float64
	MaximumCloudCover    float64
	MaximumPrecipitation float64
}

// RankedForecastPoint is a forecast point evaluated for one celestial target.
// Score is comparable only within the same target and location.
type RankedForecastPoint struct {
	Point          ForecastPoint
	TargetAltitude float64
	Score          int
	Eligible       bool
	Reason         string
}

// RankTargetForecast evaluates forecast points using local target altitude,
// astronomical darkness, and provider-reported weather values. It never
// fills missing provider fields with assumptions.
func RankTargetForecast(points []ForecastPoint, location astronomy.Coordinates, target astronomy.TargetCoordinates, config RankingConfig) ([]RankedForecastPoint, error) {
	if err := location.Validate(); err != nil {
		return nil, err
	}
	if err := target.Validate(); err != nil {
		return nil, err
	}
	if config.MinimumAltitude < -90 || config.MinimumAltitude > 90 || config.MaximumCloudCover < 0 || config.MaximumCloudCover > 100 || config.MaximumPrecipitation < 0 || config.MaximumPrecipitation > 100 {
		return nil, errors.New("ranking thresholds are outside their valid ranges")
	}
	ranked := make([]RankedForecastPoint, 0, len(points))
	for _, point := range points {
		position, err := astronomy.TargetPositionAt(point.At, location, target)
		if err != nil {
			return nil, err
		}
		result := RankedForecastPoint{Point: point, TargetAltitude: position.Altitude, Reason: "eligible"}
		switch {
		case !point.Dark:
			result.Reason = "daylight"
		case position.Altitude < config.MinimumAltitude:
			result.Reason = "target below minimum altitude"
		case point.CloudCoverPercent == nil:
			result.Reason = "cloud cover unavailable"
		case point.PrecipitationProbability == nil:
			result.Reason = "precipitation probability unavailable"
		case *point.CloudCoverPercent > config.MaximumCloudCover:
			result.Reason = "cloud cover exceeds threshold"
		case *point.PrecipitationProbability > config.MaximumPrecipitation:
			result.Reason = "precipitation exceeds threshold"
		default:
			result.Eligible = true
		}
		if point.CloudCoverPercent != nil && point.PrecipitationProbability != nil {
			result.Score = int((position.Altitude/90)*40 + ((100-*point.CloudCoverPercent)/100)*40 + ((100-*point.PrecipitationProbability)/100)*20)
			if result.Score < 0 {
				result.Score = 0
			}
			if result.Score > 100 {
				result.Score = 100
			}
		}
		ranked = append(ranked, result)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Eligible != ranked[j].Eligible {
			return ranked[i].Eligible
		}
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Point.At.Before(ranked[j].Point.At)
	})
	return ranked, nil
}

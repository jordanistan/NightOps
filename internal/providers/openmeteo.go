// Package providers contains optional external service adapters.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/weather"
)

// OpenMeteoProvider retrieves current conditions from the Open-Meteo forecast API.
type OpenMeteoProvider struct {
	Endpoint     string
	Client       *http.Client
	CacheMinutes int
	Now          func() time.Time
}

// NewOpenMeteoProvider creates a provider with safe production defaults.
func NewOpenMeteoProvider(endpoint string, cacheMinutes int) *OpenMeteoProvider {
	if endpoint == "" {
		endpoint = "https://api.open-meteo.com/v1/forecast"
	}
	if cacheMinutes == 0 {
		cacheMinutes = 60
	}
	return &OpenMeteoProvider{Endpoint: endpoint, Client: &http.Client{Timeout: 8 * time.Second}, CacheMinutes: cacheMinutes, Now: time.Now}
}

// Fetch retrieves current temperature and cloud cover for a coordinate.
func (p *OpenMeteoProvider) Fetch(ctx context.Context, latitude, longitude float64) (weather.Snapshot, error) {
	return p.fetch(ctx, latitude, longitude, false)
}

// FetchForecast retrieves current conditions and the next two days of hourly data.
func (p *OpenMeteoProvider) FetchForecast(ctx context.Context, latitude, longitude float64) (weather.Snapshot, error) {
	return p.fetch(ctx, latitude, longitude, true)
}

func (p *OpenMeteoProvider) fetch(ctx context.Context, latitude, longitude float64, includeForecast bool) (weather.Snapshot, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return weather.Snapshot{}, fmt.Errorf("weather coordinates out of range: %.4f, %.4f", latitude, longitude)
	}
	endpoint, err := url.Parse(p.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return weather.Snapshot{}, fmt.Errorf("invalid weather endpoint %q", p.Endpoint)
	}
	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', 6, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', 6, 64))
	query.Set("current", "temperature_2m,cloud_cover")
	if includeForecast {
		query.Set("hourly", "temperature_2m,cloud_cover,precipitation_probability")
		query.Set("forecast_days", "2")
	}
	query.Set("timezone", "auto")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return weather.Snapshot{}, fmt.Errorf("create weather request: %w", err)
	}
	response, err := p.client().Do(request)
	if err != nil {
		return weather.Snapshot{}, fmt.Errorf("fetch weather: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return weather.Snapshot{}, fmt.Errorf("read weather response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return weather.Snapshot{}, fmt.Errorf("weather provider returned HTTP %d", response.StatusCode)
	}
	var payload apiResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return weather.Snapshot{}, fmt.Errorf("decode weather response: %w", err)
	}
	if payload.Error {
		return weather.Snapshot{}, fmt.Errorf("weather provider error: %s", payload.Reason)
	}
	if payload.Current.Time == "" {
		return weather.Snapshot{}, fmt.Errorf("weather response omitted current observation time")
	}
	observed, err := parseLocalTime(payload.Current.Time, payload.Timezone)
	if err != nil {
		return weather.Snapshot{}, fmt.Errorf("parse weather observation time: %w", err)
	}
	if p.Now != nil {
		now := p.Now()
		if now.IsZero() {
			now = time.Now()
		}
		// Provider timestamps are authoritative, but keep test and clock-skewed
		// callers from receiving an already-expired snapshot.
		if observed.After(now.Add(5 * time.Minute)) {
			observed = now
		}
	}
	minutes := p.CacheMinutes
	if minutes <= 0 {
		minutes = 60
	}
	temperature := payload.Current.Temperature
	cloudCover := payload.Current.CloudCover
	snapshot := weather.Snapshot{LocationKey: LocationKey(latitude, longitude), Source: "open-meteo", ObservedAt: observed.UTC(), ExpiresAt: observed.UTC().Add(time.Duration(minutes) * time.Minute), TemperatureC: &temperature, CloudCoverPercent: &cloudCover, Payload: string(body)}
	if includeForecast {
		forecast, err := parseForecast(payload.Hourly, payload.Timezone)
		if err != nil {
			return weather.Snapshot{}, err
		}
		snapshot.Forecast = forecast
	}
	if err := snapshot.Validate(); err != nil {
		return weather.Snapshot{}, err
	}
	return snapshot, nil
}

func parseForecast(hourly hourlyResponse, timezone string) ([]weather.ForecastPoint, error) {
	if len(hourly.Time) == 0 {
		return nil, fmt.Errorf("weather response omitted hourly forecast")
	}
	if len(hourly.Temperature) != len(hourly.Time) || len(hourly.CloudCover) != len(hourly.Time) || len(hourly.PrecipitationProbability) != len(hourly.Time) {
		return nil, fmt.Errorf("weather hourly arrays have inconsistent lengths")
	}
	forecast := make([]weather.ForecastPoint, 0, len(hourly.Time))
	for index, value := range hourly.Time {
		at, err := parseLocalTime(value, timezone)
		if err != nil {
			return nil, fmt.Errorf("parse forecast time %q: %w", value, err)
		}
		forecast = append(forecast, weather.ForecastPoint{At: at.UTC(), TemperatureC: hourly.Temperature[index], CloudCoverPercent: hourly.CloudCover[index], PrecipitationProbability: hourly.PrecipitationProbability[index]})
	}
	return forecast, nil
}

// LocationKey returns the stable cache key for a coordinate pair.
func LocationKey(latitude, longitude float64) string {
	return strconv.FormatFloat(latitude, 'f', 6, 64) + "," + strconv.FormatFloat(longitude, 'f', 6, 64)
}

func (p *OpenMeteoProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

func parseLocalTime(value, timezone string) (time.Time, error) {
	location := time.UTC
	if strings.TrimSpace(timezone) != "" {
		loaded, err := time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, err
		}
		location = loaded
	}
	return time.ParseInLocation("2006-01-02T15:04", value, location)
}

type apiResponse struct {
	Error    bool   `json:"error"`
	Reason   string `json:"reason"`
	Timezone string `json:"timezone"`
	Current  struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		CloudCover  float64 `json:"cloud_cover"`
	} `json:"current"`
	Hourly hourlyResponse `json:"hourly"`
}

type hourlyResponse struct {
	Time                     []string   `json:"time"`
	Temperature              []*float64 `json:"temperature_2m"`
	CloudCover               []*float64 `json:"cloud_cover"`
	PrecipitationProbability []*float64 `json:"precipitation_probability"`
}

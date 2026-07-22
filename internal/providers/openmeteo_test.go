package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOpenMeteoProviderFetchesCurrentConditions(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.Query().Get("latitude"); got != "30.267200" {
			t.Errorf("latitude query = %q", got)
		}
		if got := r.URL.Query().Get("longitude"); got != "-97.743100" {
			t.Errorf("longitude query = %q", got)
		}
		if got := r.URL.Query().Get("current"); got != "temperature_2m,cloud_cover" {
			t.Errorf("current query = %q", got)
		}
		return jsonResponse(http.StatusOK, `{"timezone":"America/Chicago","current":{"time":"2026-07-21T20:00","temperature_2m":28.4,"cloud_cover":17}}`), nil
	})}

	provider := NewOpenMeteoProvider("https://weather.test/forecast", 45)
	provider.Client = client
	snapshot, err := provider.Fetch(context.Background(), 30.2672, -97.7431)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.LocationKey != "30.267200,-97.743100" || snapshot.Source != "open-meteo" {
		t.Fatalf("unexpected snapshot identity: %+v", snapshot)
	}
	if snapshot.TemperatureC == nil || *snapshot.TemperatureC != 28.4 || snapshot.CloudCoverPercent == nil || *snapshot.CloudCoverPercent != 17 {
		t.Fatalf("unexpected conditions: %+v", snapshot)
	}
	if !strings.Contains(snapshot.Payload, "cloud_cover") || snapshot.ExpiresAt.Sub(snapshot.ObservedAt) != 45*time.Minute {
		t.Fatalf("snapshot provenance or expiry missing: %+v", snapshot)
	}
}

func TestOpenMeteoProviderRejectsHTTPFailureAndInvalidCoordinates(t *testing.T) {
	provider := NewOpenMeteoProvider("https://weather.test/forecast", 60)
	provider.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, "upstream unavailable"), nil
	})}
	if _, err := provider.Fetch(context.Background(), 91, 0); err == nil {
		t.Fatal("expected coordinate validation error")
	}
	if _, err := provider.Fetch(context.Background(), 30, -97); err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("expected upstream error, got %v", err)
	}
}

func TestOpenMeteoProviderParsesHourlyForecast(t *testing.T) {
	provider := NewOpenMeteoProvider("https://weather.test/forecast", 60)
	provider.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("hourly") != "temperature_2m,cloud_cover,precipitation_probability" || r.URL.Query().Get("forecast_days") != "2" {
			t.Fatalf("forecast query missing: %s", r.URL.RawQuery)
		}
		return jsonResponse(http.StatusOK, `{"timezone":"America/Chicago","current":{"time":"2026-07-21T20:00","temperature_2m":28.4,"cloud_cover":17},"hourly":{"time":["2026-07-21T20:00","2026-07-21T21:00"],"temperature_2m":[28.4,27.8],"cloud_cover":[17,20],"precipitation_probability":[5,10]}}`), nil
	})}
	snapshot, err := provider.FetchForecast(context.Background(), 30.2672, -97.7431)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Forecast) != 2 || snapshot.Forecast[1].TemperatureC == nil || *snapshot.Forecast[1].TemperatureC != 27.8 || snapshot.Forecast[1].PrecipitationProbability == nil || *snapshot.Forecast[1].PrecipitationProbability != 10 {
		t.Fatalf("hourly forecast did not parse: %+v", snapshot.Forecast)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d", status), Body: ioNopCloser{strings.NewReader(body)}, Header: make(http.Header)}
}

type ioNopCloser struct{ *strings.Reader }

func (ioNopCloser) Close() error { return nil }

func TestLocationKeyIsStable(t *testing.T) {
	if got := LocationKey(30.2672, -97.7431); got != "30.267200,-97.743100" {
		t.Fatalf("unexpected location key: %s", got)
	}
}

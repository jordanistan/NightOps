package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/astronomy"
)

func TestOSRMProviderParsesDrivingRoute(t *testing.T) {
	provider := NewOSRMProvider("https://routing.test")
	provider.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.Contains(request.URL.Path, "/route/v1/driving/-97.743100,30.267200;-98.369000,30.737300") {
			t.Fatalf("unexpected OSRM path: %s", request.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"code":"Ok","routes":[{"distance":12345.6,"duration":3720}]}`), nil
	})}
	plan, err := provider.Route(context.Background(), astronomy.Coordinates{Latitude: 30.2672, Longitude: -97.7431}, astronomy.Coordinates{Latitude: 30.7373, Longitude: -98.369})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Provider != "osrm" || plan.DistanceKm < 12.3 || plan.DistanceKm > 12.4 || plan.TravelTime != "1 hours 2 minutes" {
		t.Fatalf("unexpected OSRM plan: %+v", plan)
	}
}

func TestOSRMProviderRejectsNoRoute(t *testing.T) {
	provider := NewOSRMProvider("https://routing.test")
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"code":"NoRoute","routes":[]}`), nil
	})}
	if _, err := provider.Route(context.Background(), astronomy.Coordinates{}, astronomy.Coordinates{}); err == nil {
		t.Fatal("expected no route error")
	}
}

func TestOSRMProviderRetriesTransientResponses(t *testing.T) {
	provider := NewOSRMProviderWithPolicy("https://routing.test", time.Second, 2, time.Second)
	attempts := 0
	sleeps := 0
	provider.Sleep = func(context.Context, time.Duration) error {
		sleeps++
		return nil
	}
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return jsonResponse(http.StatusBadGateway, "temporary failure"), nil
		}
		return jsonResponse(http.StatusOK, `{"code":"Ok","routes":[{"distance":1000,"duration":60}]}`), nil
	})}
	if _, err := provider.Route(context.Background(), astronomy.Coordinates{Latitude: 30, Longitude: -97}, astronomy.Coordinates{Latitude: 31, Longitude: -98}); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || sleeps != 2 {
		t.Fatalf("retry policy attempts=%d sleeps=%d", attempts, sleeps)
	}
}

func TestOSRMProviderDoesNotRetryNoRoute(t *testing.T) {
	provider := NewOSRMProviderWithPolicy("https://routing.test", time.Second, 3, 0)
	attempts := 0
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return jsonResponse(http.StatusOK, `{"code":"NoRoute","routes":[]}`), nil
	})}
	if _, err := provider.Route(context.Background(), astronomy.Coordinates{Latitude: 30, Longitude: -97}, astronomy.Coordinates{Latitude: 31, Longitude: -98}); err == nil {
		t.Fatal("expected no route error")
	}
	if attempts != 1 {
		t.Fatalf("semantic no-route response was retried %d times", attempts)
	}
}

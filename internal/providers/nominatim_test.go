package providers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNominatimResolvesStructuredUSZIP(t *testing.T) {
	provider := NewNominatimProvider("https://nominatim.test/search", time.Second)
	provider.Client = &http.Client{Transport: nominatimRoundTrip(func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		if query.Get("postalcode") != "78701" || query.Get("countrycodes") != "us" || query.Get("format") != "jsonv2" || query.Get("limit") != "1" {
			t.Fatalf("unexpected query: %v", query)
		}
		if request.Header.Get("User-Agent") == "" {
			t.Fatal("missing User-Agent")
		}
		return nominatimResponse(`[{"display_name":"Austin, Texas","lat":"30.2672","lon":"-97.7431"}]`), nil
	})}
	result, err := provider.Resolve(context.Background(), "78701")
	if err != nil || result.Label != "Austin, Texas" || result.Latitude != 30.2672 || result.Longitude != -97.7431 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestNominatimRejectsEmptyResultsAndInvalidZIP(t *testing.T) {
	provider := NewNominatimProvider("https://nominatim.test/search", time.Second)
	provider.Client = &http.Client{Transport: nominatimRoundTrip(func(*http.Request) (*http.Response, error) { return nominatimResponse(`[]`), nil })}
	if _, err := provider.Resolve(context.Background(), "Austin"); err == nil {
		t.Fatal("invalid ZIP was accepted")
	}
	if _, err := provider.Resolve(context.Background(), "78701"); err == nil || !strings.Contains(err.Error(), "no coordinate result") {
		t.Fatalf("empty result error=%v", err)
	}
}

type nominatimRoundTrip func(*http.Request) (*http.Response, error)

func (f nominatimRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func nominatimResponse(payload string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}
}

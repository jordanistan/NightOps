package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
)

func TestWikipediaProviderParsesReferenceAndImage(t *testing.T) {
	provider := NewWikipediaProvider("https://wiki.test/api/rest_v1/page/summary", time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.Contains(request.URL.Path, "Andromeda_Galaxy") {
			t.Fatalf("target title was not encoded in request: %s", request.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"title":"Andromeda Galaxy","extract":"A nearby galaxy.","thumbnail":{"source":"https://images.test/m31.jpg"},"content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Andromeda_Galaxy"}}}`), nil
	})}
	knowledge, err := provider.Fetch(context.Background(), domain.MissionTarget{ID: "m31", Name: "Andromeda Galaxy"})
	if err != nil {
		t.Fatal(err)
	}
	if knowledge.Status != "live" || knowledge.Summary != "A nearby galaxy." || knowledge.ImageURL == "" || knowledge.URL == "" {
		t.Fatalf("reference data was not parsed: %+v", knowledge)
	}
}

func TestWikipediaProviderSurfacesHTTPFailure(t *testing.T) {
	provider := NewWikipediaProvider("https://wiki.test/summary", time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{}`), nil
	})}
	if _, err := provider.Fetch(context.Background(), domain.MissionTarget{ID: "m31", Name: "Andromeda Galaxy"}); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected Wikipedia HTTP error, got %v", err)
	}
}

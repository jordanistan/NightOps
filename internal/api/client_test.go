package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientReadsStatusAndMissions(t *testing.T) {
	client := NewClient("http://nightops.test")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/api/v1/status" {
			return clientResponse(`{"service":"nightops","api_version":1,"offline_first":true,"sync_enabled":true}`), nil
		}
		if request.URL.Path == "/api/v1/missions/m-1" {
			return clientResponse(`{"id":"m-1","name":"First Light","status":"planned","created_at":"2026-07-22T00:00:00Z","updated_at":"2026-07-22T00:00:00Z"}`), nil
		}
		return clientResponse(`{"missions":[{"id":"m-1","name":"First Light","status":"planned","launch_site_name":"Home Base","timezone":"UTC","created_at":"2026-07-22T00:00:00Z"}]}`), nil
	})}
	status, err := client.Status(context.Background())
	if err != nil || !status.OfflineFirst || !status.SyncEnabled {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
	missions, err := client.Missions(context.Background())
	if err != nil || len(missions) != 1 || missions[0].Name != "First Light" {
		t.Fatalf("missions = %+v, err = %v", missions, err)
	}
	detail, err := client.Mission(context.Background(), "m-1")
	if err != nil || detail.ID != "m-1" || detail.CreatedAt.IsZero() {
		t.Fatalf("mission detail = %+v, err = %v", detail, err)
	}
}

func TestClientReportsAPIError(t *testing.T) {
	client := NewClient("http://nightops.test")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Body: io.NopCloser(strings.NewReader(`{"error":"sync API is disabled"}`)), Header: make(http.Header)}, nil
	})}
	if _, err := client.ExportSync(context.Background()); err == nil || !strings.Contains(err.Error(), "sync API is disabled") {
		t.Fatalf("error = %v", err)
	}
}

func TestClientRejectsMissingBaseURL(t *testing.T) {
	if _, err := NewClient("").Status(context.Background()); err == nil {
		t.Fatal("missing base URL was accepted")
	}
}

func clientResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

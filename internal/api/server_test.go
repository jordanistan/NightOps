package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	syncbundle "github.com/jordanistan/nightops/internal/sync"
)

func TestHandlerServesStatusMissionsAndSync(t *testing.T) {
	bundle, err := syncbundle.NewBundle(time.Now().UTC(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	merged := false
	server := New(Config{AllowSync: true}, Dependencies{
		ListMissions: func(context.Context) ([]Mission, error) {
			return []Mission{{ID: "mission-1", Name: "Test Mission", Status: "planned"}}, nil
		},
		FindMission: func(_ context.Context, id string) (Mission, error) {
			if id != "mission-1" {
				return Mission{}, errors.New("not found")
			}
			return Mission{ID: id, Name: "Test Mission", Status: "planned"}, nil
		},
		ExportSync: func(context.Context) (syncbundle.Bundle, error) { return bundle, nil },
		ImportSync: func(_ context.Context, incoming syncbundle.Bundle) (syncbundle.MergeReport, error) {
			merged = incoming.SchemaVersion == syncbundle.CurrentVersion && incoming.SourceDeviceID == ""
			return syncbundle.MergeReport{Added: 1, Skipped: 2}, nil
		},
	})
	handler := server.Handler()

	status := request(t, handler, http.MethodGet, "/api/v1/status", nil)
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), "offline_first") {
		t.Fatalf("status response=%d %s", status.Code, status.Body.String())
	}
	companion := request(t, handler, http.MethodGet, "/companion/", nil)
	if companion.Code != http.StatusOK || !strings.Contains(companion.Body.String(), "NightOps Companion") || !strings.Contains(companion.Body.String(), "Remote API credential") {
		t.Fatalf("companion response=%d %s", companion.Code, companion.Body.String())
	}
	manifest := request(t, handler, http.MethodGet, "/companion/manifest.json", nil)
	if manifest.Code != http.StatusOK || !strings.Contains(manifest.Body.String(), "NightOps Companion") {
		t.Fatalf("companion manifest response=%d %s", manifest.Code, manifest.Body.String())
	}
	missions := request(t, handler, http.MethodGet, "/api/v1/missions", nil)
	if missions.Code != http.StatusOK || !strings.Contains(missions.Body.String(), "Test Mission") {
		t.Fatalf("missions response=%d %s", missions.Code, missions.Body.String())
	}
	detail := request(t, handler, http.MethodGet, "/api/v1/missions/mission-1", nil)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "mission-1") {
		t.Fatalf("mission detail response=%d %s", detail.Code, detail.Body.String())
	}
	missing := request(t, handler, http.MethodGet, "/api/v1/missions/unknown", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing mission status=%d", missing.Code)
	}
	exported := request(t, handler, http.MethodGet, "/api/v1/sync", nil)
	if exported.Code != http.StatusOK || !strings.Contains(exported.Body.String(), "schema_version") {
		t.Fatalf("sync export response=%d %s", exported.Code, exported.Body.String())
	}
	body, _ := json.Marshal(bundle)
	imported := request(t, handler, http.MethodPost, "/api/v1/sync", bytes.NewReader(body))
	if imported.Code != http.StatusOK || !merged || !strings.Contains(imported.Body.String(), "added") {
		t.Fatalf("sync import response=%d merged=%v %s", imported.Code, merged, imported.Body.String())
	}
}

func TestHandlerDisablesSyncAndLimitsMethods(t *testing.T) {
	server := New(Config{AllowSync: false}, Dependencies{})
	handler := server.Handler()
	if response := request(t, handler, http.MethodGet, "/api/v1/sync", nil); response.Code != http.StatusForbidden {
		t.Fatalf("disabled sync status=%d", response.Code)
	}
	if response := request(t, handler, http.MethodPost, "/api/v1/missions", nil); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("invalid missions method status=%d", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/missing", nil); response.Code != http.StatusNotFound {
		t.Fatalf("missing route status=%d", response.Code)
	}
}

func TestHandlerRequiresConfiguredAuthorization(t *testing.T) {
	const envName = "NIGHTOPS_TEST_API_AUTH"
	t.Setenv(envName, "local-test-value")
	server := New(Config{AuthEnv: envName}, Dependencies{})
	handler := server.Handler()
	if response := request(t, handler, http.MethodGet, "/api/v1/status", nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing authorization status=%d", response.Code)
	}
	publicCompanion := request(t, handler, http.MethodGet, "/companion/", nil)
	if publicCompanion.Code != http.StatusOK {
		t.Fatalf("companion shell should be reachable before credential entry: status=%d", publicCompanion.Code)
	}
	authorized := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	authorized.Header.Set("Authorization", "Bearer "+os.Getenv(envName))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorized)
	if recorder.Code != http.StatusOK {
		t.Fatalf("authorized request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	authorizedCompanion := httptest.NewRequest(http.MethodGet, "/companion/", nil)
	authorizedCompanion.Header.Set("Authorization", "Bearer "+os.Getenv(envName))
	companionRecorder := httptest.NewRecorder()
	handler.ServeHTTP(companionRecorder, authorizedCompanion)
	if companionRecorder.Code != http.StatusOK {
		t.Fatalf("authorized companion status=%d", companionRecorder.Code)
	}
}

func TestHandlerReportsMissingAuthorizationConfiguration(t *testing.T) {
	const envName = "NIGHTOPS_TEST_API_AUTH_MISSING"
	t.Setenv(envName, "")
	server := New(Config{AuthEnv: envName}, Dependencies{})
	response := request(t, server.Handler(), http.MethodGet, "/api/v1/status", nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing auth configuration status=%d", response.Code)
	}
}

func request(t *testing.T, handler http.Handler, method, path string, body *bytes.Reader) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = body
	}
	request := httptest.NewRequest(method, path, reader)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

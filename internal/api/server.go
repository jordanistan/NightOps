// Package api exposes a small, opt-in HTTP boundary for local clients.
package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	syncbundle "github.com/jordanistan/nightops/internal/sync"
)

//go:embed web/*
var companionFiles embed.FS

// Mission is the stable API projection of an archived mission.
type Mission struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	LaunchSiteName string     `json:"launch_site_name"`
	Timezone       string     `json:"timezone"`
	EquipmentID    string     `json:"equipment_profile_id,omitempty"`
	PlannedStart   *time.Time `json:"planned_start,omitempty"`
	PlannedEnd     *time.Time `json:"planned_end,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Dependencies supplies read and sync operations without exposing SQLite to clients.
type Dependencies struct {
	ListMissions func(context.Context) ([]Mission, error)
	FindMission  func(context.Context, string) (Mission, error)
	ExportSync   func(context.Context) (syncbundle.Bundle, error)
	ImportSync   func(context.Context, syncbundle.Bundle) (syncbundle.MergeReport, error)
}

// Config controls the local HTTP listener.
type Config struct {
	ListenAddr string
	AllowSync  bool
	AuthEnv    string
}

// Server is a local, versioned HTTP API.
type Server struct {
	config Config
	deps   Dependencies
}

// New creates an API server. It does not open a listener.
func New(config Config, deps Dependencies) *Server {
	if strings.TrimSpace(config.ListenAddr) == "" {
		config.ListenAddr = "127.0.0.1:8787"
	}
	return &Server{config: config, deps: deps}
}

// Handler returns the deterministic HTTP handler for embedding or testing.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	companionRoot, err := fs.Sub(companionFiles, "web")
	if err != nil {
		panic("nightops companion assets are missing: " + err.Error())
	}
	companion := http.FileServer(http.FS(companionRoot))
	mux.Handle("/companion/", http.StripPrefix("/companion/", companion))
	mux.HandleFunc("/companion", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/companion/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/api/v1/missions", s.missions)
	mux.HandleFunc("/api/v1/missions/", s.mission)
	mux.HandleFunc("/api/v1/sync", s.sync)
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusNotFound, "not found")
	})
	return s.authentication(mux)
}

func (s *Server) authentication(next http.Handler) http.Handler {
	if strings.TrimSpace(s.config.AuthEnv) == "" {
		return next
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// The companion shell must be reachable before a user can enter the
		// optional in-memory credential. Its data and API requests remain
		// protected by this middleware.
		if request.URL.Path == "/companion" || strings.HasPrefix(request.URL.Path, "/companion/") {
			next.ServeHTTP(writer, request)
			return
		}
		expected := strings.TrimSpace(os.Getenv(s.config.AuthEnv))
		provided := strings.TrimSpace(request.Header.Get("Authorization"))
		if expected == "" {
			writeError(writer, http.StatusServiceUnavailable, "API authentication is not configured")
			return
		}
		if provided != "Bearer "+expected {
			writeError(writer, http.StatusUnauthorized, "API authentication required")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

// Run serves the API until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{Addr: s.config.ListenAddr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) status(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "GET required")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"service": "nightops", "api_version": 1, "offline_first": true, "sync_enabled": s.config.AllowSync, "companion_path": "/companion/"})
}

func (s *Server) missions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "GET required")
		return
	}
	if s.deps.ListMissions == nil {
		writeError(writer, http.StatusServiceUnavailable, "mission archive unavailable")
		return
	}
	missions, err := s.deps.ListMissions(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"missions": missions})
}

func (s *Server) mission(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "GET required")
		return
	}
	if s.deps.FindMission == nil {
		writeError(writer, http.StatusServiceUnavailable, "mission archive unavailable")
		return
	}
	id := strings.TrimPrefix(request.URL.Path, "/api/v1/missions/")
	if id == "" || strings.Contains(id, "/") {
		writeError(writer, http.StatusNotFound, "mission not found")
		return
	}
	mission, err := s.deps.FindMission(request.Context(), id)
	if err != nil {
		writeError(writer, http.StatusNotFound, "mission not found")
		return
	}
	writeJSON(writer, http.StatusOK, mission)
}

func (s *Server) sync(writer http.ResponseWriter, request *http.Request) {
	if !s.config.AllowSync {
		writeError(writer, http.StatusForbidden, "sync API is disabled")
		return
	}
	switch request.Method {
	case http.MethodGet:
		if s.deps.ExportSync == nil {
			writeError(writer, http.StatusServiceUnavailable, "sync export unavailable")
			return
		}
		bundle, err := s.deps.ExportSync(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, bundle)
	case http.MethodPost:
		if s.deps.ImportSync == nil {
			writeError(writer, http.StatusServiceUnavailable, "sync import unavailable")
			return
		}
		var bundle syncbundle.Bundle
		decoder := json.NewDecoder(io.LimitReader(request.Body, 32<<20))
		if err := decoder.Decode(&bundle); err != nil {
			writeError(writer, http.StatusBadRequest, fmt.Sprintf("decode sync bundle: %v", err))
			return
		}
		report, err := s.deps.ImportSync(request.Context(), bundle)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, report)
	default:
		writeError(writer, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, detail string) {
	writeJSON(writer, status, map[string]string{"error": detail})
}

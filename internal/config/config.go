// Package config loads and validates user configuration.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the complete user-editable NightOps configuration.
type Config struct {
	App             AppConfig             `yaml:"app"`
	Origin          OriginConfig          `yaml:"origin"`
	Obsidian        ObsidianConfig        `yaml:"obsidian"`
	Weather         WeatherConfig         `yaml:"weather"`
	Geocoding       GeocodingConfig       `yaml:"geocoding"`
	TargetKnowledge TargetKnowledgeConfig `yaml:"target_knowledge"`
	Routing         RoutingConfig         `yaml:"routing"`
	Telescope       TelescopeConfig       `yaml:"telescope"`
	AI              AIConfig              `yaml:"ai"`
	API             APIConfig             `yaml:"api"`
	Plugins         PluginConfig          `yaml:"plugins"`
	Astronomy       AstronomyConfig       `yaml:"astronomy"`
	Features        FeatureConfig         `yaml:"features"`
}

// Save writes a complete YAML configuration using an atomic replacement.
func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}

type AppConfig struct {
	Name       string `yaml:"name"`
	DataDir    string `yaml:"data_dir"`
	LogLevel   string `yaml:"log_level"`
	Theme      string `yaml:"theme"`
	ConfigPath string `yaml:"-"`
}
type OriginConfig struct {
	Mode         string  `yaml:"mode"`
	HomeBaseName string  `yaml:"home_base_name"`
	HomeBaseZIP  string  `yaml:"home_base_zip"`
	Latitude     float64 `yaml:"latitude"`
	Longitude    float64 `yaml:"longitude"`
	Timezone     string  `yaml:"timezone"`
}
type ObsidianConfig struct {
	Enabled  bool   `yaml:"enabled"`
	VaultDir string `yaml:"vault_dir"`
	NotesDir string `yaml:"notes_dir"`
}
type WeatherConfig struct {
	Provider                 string `yaml:"provider"`
	Endpoint                 string `yaml:"endpoint"`
	CacheMinutes             int    `yaml:"cache_minutes"`
	ForecastCloudCoverMax    int    `yaml:"forecast_cloud_cover_max"`
	ForecastPrecipitationMax int    `yaml:"forecast_precipitation_max"`
}

// GeocodingConfig controls the optional ZIP-to-coordinate provider and cache.
type GeocodingConfig struct {
	Provider       string `yaml:"provider"`
	Endpoint       string `yaml:"endpoint"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	CacheMinutes   int    `yaml:"cache_minutes"`
}
type RoutingConfig struct {
	Provider           string `yaml:"provider"`
	Endpoint           string `yaml:"endpoint"`
	CacheMinutes       int    `yaml:"cache_minutes"`
	TimeoutSeconds     int    `yaml:"timeout_seconds"`
	MaxRetries         int    `yaml:"max_retries"`
	RetryBackoffMillis int    `yaml:"retry_backoff_millis"`
}

// TelescopeConfig controls an optional Alpaca- or Dwarf-compatible telescope adapter.
type TelescopeConfig struct {
	Provider       string `yaml:"provider"`
	Endpoint       string `yaml:"endpoint"`
	DeviceNumber   int    `yaml:"device_number"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// AIConfig controls the optional local mission-brief provider.
type AIConfig struct {
	Provider       string `yaml:"provider"`
	Endpoint       string `yaml:"endpoint"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// APIConfig controls the optional local HTTP boundary for future clients.
type APIConfig struct {
	ListenAddr  string `yaml:"listen_addr"`
	AllowSync   bool   `yaml:"allow_sync"`
	AllowRemote bool   `yaml:"allow_remote"`
	AuthEnv     string `yaml:"auth_env"`
}

// PluginConfig controls local plugin manifest discovery.
type PluginConfig struct {
	Dir string `yaml:"dir"`
}

// AstronomyConfig controls local target visibility constraints.
type AstronomyConfig struct {
	MinimumTargetAltitude int `yaml:"minimum_target_altitude"`
}

// TargetKnowledgeConfig controls bounded live reference lookups for selected targets.
type TargetKnowledgeConfig struct {
	Endpoint       string `yaml:"endpoint"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}
type FeatureConfig struct {
	CommandPalette  bool `yaml:"command_palette"`
	Atlas           bool `yaml:"atlas"`
	Weather         bool `yaml:"weather"`
	Routing         bool `yaml:"routing"`
	Plugins         bool `yaml:"plugins"`
	Telescope       bool `yaml:"telescope"`
	AI              bool `yaml:"ai"`
	API             bool `yaml:"api"`
	Geocoding       bool `yaml:"geocoding"`
	TargetKnowledge bool `yaml:"target_knowledge"`
}

// Defaults returns a safe configuration for a new local installation.
func Defaults() Config {
	return Config{App: AppConfig{Name: "NightOps", DataDir: "~/.local/share/nightops", LogLevel: "info", Theme: "mission-control"}, Origin: OriginConfig{Mode: "home_base", HomeBaseName: "Home Base", Timezone: "UTC"}, Obsidian: ObsidianConfig{Enabled: true, VaultDir: "~/Documents/Obsidian", NotesDir: "NightOps"}, Weather: WeatherConfig{Provider: "open-meteo", Endpoint: "https://api.open-meteo.com/v1/forecast", CacheMinutes: 60, ForecastCloudCoverMax: 50, ForecastPrecipitationMax: 20}, Geocoding: GeocodingConfig{Provider: "nominatim", Endpoint: "https://nominatim.openstreetmap.org/search", CacheMinutes: 10080, TimeoutSeconds: 8}, TargetKnowledge: TargetKnowledgeConfig{Endpoint: "https://en.wikipedia.org/api/rest_v1/page/summary", TimeoutSeconds: 8}, Routing: RoutingConfig{Provider: "osrm", Endpoint: "https://router.project-osrm.org", CacheMinutes: 360}, Telescope: TelescopeConfig{Provider: "alpaca", DeviceNumber: 0, TimeoutSeconds: 8}, AI: AIConfig{Provider: "ollama", Endpoint: "http://127.0.0.1:11434/api/generate", TimeoutSeconds: 30}, API: APIConfig{ListenAddr: "127.0.0.1:8787"}, Features: FeatureConfig{CommandPalette: true, Weather: true, Geocoding: true, TargetKnowledge: true}}
}

// Load reads YAML from path, applying defaults for omitted fields.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks configuration values that would make startup unsafe.
func (c Config) Validate() error {
	if c.App.Name == "" || c.App.DataDir == "" {
		return errors.New("app name and data_dir are required")
	}
	if c.App.Theme != "mission-control" && c.App.Theme != "observatory" {
		return errors.New("app theme must be mission-control or observatory")
	}
	if c.Origin.Mode != "home_base" && c.Origin.Mode != "zip" && c.Origin.Mode != "gps" {
		return errors.New("origin mode must be home_base, zip, or gps")
	}
	if c.Origin.Latitude < -90 || c.Origin.Latitude > 90 {
		return errors.New("origin latitude must be between -90 and 90")
	}
	if c.Origin.Longitude < -180 || c.Origin.Longitude > 180 {
		return errors.New("origin longitude must be between -180 and 180")
	}
	if c.Weather.CacheMinutes < 0 {
		return errors.New("weather cache_minutes cannot be negative")
	}
	if c.Weather.ForecastCloudCoverMax < 0 || c.Weather.ForecastCloudCoverMax > 100 {
		return errors.New("weather forecast_cloud_cover_max must be between 0 and 100")
	}
	if c.Weather.ForecastPrecipitationMax < 0 || c.Weather.ForecastPrecipitationMax > 100 {
		return errors.New("weather forecast_precipitation_max must be between 0 and 100")
	}
	if c.Geocoding.CacheMinutes < 0 {
		return errors.New("geocoding cache_minutes cannot be negative")
	}
	if c.Geocoding.TimeoutSeconds != 0 && (c.Geocoding.TimeoutSeconds < 1 || c.Geocoding.TimeoutSeconds > 120) {
		return errors.New("geocoding timeout_seconds must be between 1 and 120")
	}
	if c.TargetKnowledge.TimeoutSeconds != 0 && (c.TargetKnowledge.TimeoutSeconds < 1 || c.TargetKnowledge.TimeoutSeconds > 120) {
		return errors.New("target_knowledge timeout_seconds must be between 1 and 120")
	}
	if c.Routing.CacheMinutes < 0 {
		return errors.New("routing cache_minutes cannot be negative")
	}
	if c.Routing.TimeoutSeconds != 0 && (c.Routing.TimeoutSeconds < 1 || c.Routing.TimeoutSeconds > 120) {
		return errors.New("routing timeout_seconds must be between 1 and 120")
	}
	if c.Routing.MaxRetries < 0 || c.Routing.MaxRetries > 5 {
		return errors.New("routing max_retries must be between 0 and 5")
	}
	if c.Routing.RetryBackoffMillis < 0 || c.Routing.RetryBackoffMillis > 5000 {
		return errors.New("routing retry_backoff_millis must be between 0 and 5000")
	}
	if c.Telescope.DeviceNumber < 0 {
		return errors.New("telescope device_number cannot be negative")
	}
	if c.Telescope.TimeoutSeconds != 0 && (c.Telescope.TimeoutSeconds < 1 || c.Telescope.TimeoutSeconds > 120) {
		return errors.New("telescope timeout_seconds must be between 1 and 120")
	}
	if c.AI.TimeoutSeconds != 0 && (c.AI.TimeoutSeconds < 1 || c.AI.TimeoutSeconds > 300) {
		return errors.New("ai timeout_seconds must be between 1 and 300")
	}
	if c.Features.API && !c.API.AllowRemote && !loopbackListenAddr(c.API.ListenAddr) {
		return errors.New("api listen_addr must be loopback unless allow_remote is true")
	}
	if c.Features.API && c.API.AllowRemote && strings.TrimSpace(c.API.AuthEnv) == "" {
		return errors.New("api auth_env is required when allow_remote is true")
	}
	if c.Astronomy.MinimumTargetAltitude != 0 && (c.Astronomy.MinimumTargetAltitude < 1 || c.Astronomy.MinimumTargetAltitude > 90) {
		return errors.New("astronomy minimum_target_altitude must be between 1 and 90")
	}
	return nil
}

func loopbackListenAddr(address string) bool {
	if strings.TrimSpace(address) == "" {
		return true
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ExpandPath expands a leading tilde and cleans a filesystem path.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Clean(path)
}

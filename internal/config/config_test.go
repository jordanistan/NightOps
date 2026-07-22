package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultsValidate(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedThemeIsRejected(t *testing.T) {
	cfg := Defaults()
	cfg.App.Theme = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("unsupported theme was accepted")
	}
}
func TestInvalidOrigin(t *testing.T) {
	cfg := Defaults()
	cfg.Origin.Mode = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestInvalidWeatherCacheDuration(t *testing.T) {
	cfg := Defaults()
	cfg.Weather.CacheMinutes = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected weather cache validation error")
	}
}

func TestInvalidForecastThresholds(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"cloud below zero":         func(cfg *Config) { cfg.Weather.ForecastCloudCoverMax = -1 },
		"cloud above one hundred":  func(cfg *Config) { cfg.Weather.ForecastCloudCoverMax = 101 },
		"precip below zero":        func(cfg *Config) { cfg.Weather.ForecastPrecipitationMax = -1 },
		"precip above one hundred": func(cfg *Config) { cfg.Weather.ForecastPrecipitationMax = 101 },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Defaults()
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected forecast threshold validation error")
			}
		})
	}
}

func TestInvalidTargetAltitude(t *testing.T) {
	for _, value := range []int{-1, 91} {
		cfg := Defaults()
		cfg.Astronomy.MinimumTargetAltitude = value
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected target altitude validation error for %d", value)
		}
	}
}

func TestInvalidRoutingCacheDuration(t *testing.T) {
	cfg := Defaults()
	cfg.Routing.CacheMinutes = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected routing cache validation error")
	}
}

func TestInvalidRoutingRetryPolicy(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"timeout too short": func(cfg *Config) { cfg.Routing.TimeoutSeconds = 0 - 1 },
		"timeout too long":  func(cfg *Config) { cfg.Routing.TimeoutSeconds = 121 },
		"retries negative":  func(cfg *Config) { cfg.Routing.MaxRetries = -1 },
		"retries too high":  func(cfg *Config) { cfg.Routing.MaxRetries = 6 },
		"backoff negative":  func(cfg *Config) { cfg.Routing.RetryBackoffMillis = -1 },
		"backoff too high":  func(cfg *Config) { cfg.Routing.RetryBackoffMillis = 5001 },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Defaults()
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected routing retry policy validation error")
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Origin.HomeBaseName = "Dark Site"
	cfg.Origin.HomeBaseZIP = "78701"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Origin.HomeBaseName != "Dark Site" || loaded.Origin.HomeBaseZIP != "78701" {
		t.Fatalf("saved home base did not round-trip: %+v", loaded.Origin)
	}
}

func TestPluginDirectoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Plugins.Dir = "/var/lib/nightops/plugins"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.Plugins.Dir != cfg.Plugins.Dir {
		t.Fatalf("plugin directory did not round-trip: %+v err=%v", loaded.Plugins, err)
	}
}

func TestTelescopeConfigurationRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Features.Telescope = true
	cfg.Telescope = TelescopeConfig{Provider: "alpaca", Endpoint: "http://127.0.0.1:11111", DeviceNumber: 2, TimeoutSeconds: 12}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.Telescope != cfg.Telescope || !loaded.Features.Telescope {
		t.Fatalf("telescope configuration did not round-trip: %+v err=%v", loaded.Telescope, err)
	}
}

func TestGeocodingConfigurationRoundTripAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Features.Geocoding = true
	cfg.Geocoding = GeocodingConfig{Provider: "nominatim", Endpoint: "https://geocoder.test/search", TimeoutSeconds: 12, CacheMinutes: 1440}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.Geocoding != cfg.Geocoding || !loaded.Features.Geocoding {
		t.Fatalf("geocoding configuration did not round-trip: %+v err=%v", loaded.Geocoding, err)
	}
	cfg.Geocoding.TimeoutSeconds = 121
	if err := cfg.Validate(); err == nil {
		t.Fatal("unbounded geocoding timeout was accepted")
	}
}

func TestTelescopeConfigurationRejectsUnsafeValues(t *testing.T) {
	cfg := Defaults()
	cfg.Telescope.DeviceNumber = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative telescope device number was accepted")
	}
	cfg = Defaults()
	cfg.Telescope.TimeoutSeconds = 121
	if err := cfg.Validate(); err == nil {
		t.Fatal("unbounded telescope timeout was accepted")
	}
}

func TestAIConfigurationRoundTripAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Features.AI = true
	cfg.AI = AIConfig{Provider: "ollama", Endpoint: "http://127.0.0.1:11434/api/generate", Model: "llama3", TimeoutSeconds: 45}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.AI != cfg.AI || !loaded.Features.AI {
		t.Fatalf("AI configuration did not round-trip: %+v err=%v", loaded.AI, err)
	}
	cfg.AI.TimeoutSeconds = 301
	if err := cfg.Validate(); err == nil {
		t.Fatal("unbounded AI timeout was accepted")
	}
}

func TestAPIConfigurationIsLoopbackByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Defaults()
	cfg.Features.API = true
	cfg.API = APIConfig{ListenAddr: "127.0.0.1:9999", AllowSync: true}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded.API != cfg.API || !loaded.Features.API {
		t.Fatalf("API configuration did not round-trip: %+v err=%v", loaded.API, err)
	}
	cfg.API.ListenAddr = "0.0.0.0:9999"
	if err := cfg.Validate(); err == nil {
		t.Fatal("remote API bind was accepted without explicit opt-in")
	}
	cfg.API.AllowRemote = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("remote API without authentication was accepted")
	}
	cfg.API.AuthEnv = "NIGHTOPS_API_AUTH"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("explicit authenticated remote API opt-in was rejected: %v", err)
	}
}

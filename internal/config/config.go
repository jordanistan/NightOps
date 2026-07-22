// Package config loads and validates user configuration.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the complete user-editable NightOps configuration.
type Config struct {
	App      AppConfig      `yaml:"app"`
	Origin   OriginConfig   `yaml:"origin"`
	Obsidian ObsidianConfig `yaml:"obsidian"`
	Features FeatureConfig  `yaml:"features"`
}
type AppConfig struct {
	Name     string `yaml:"name"`
	DataDir  string `yaml:"data_dir"`
	LogLevel string `yaml:"log_level"`
	Theme    string `yaml:"theme"`
}
type OriginConfig struct {
	Mode         string  `yaml:"mode"`
	HomeBaseName string  `yaml:"home_base_name"`
	Latitude     float64 `yaml:"latitude"`
	Longitude    float64 `yaml:"longitude"`
	Timezone     string  `yaml:"timezone"`
}
type ObsidianConfig struct {
	Enabled  bool   `yaml:"enabled"`
	VaultDir string `yaml:"vault_dir"`
	NotesDir string `yaml:"notes_dir"`
}
type FeatureConfig struct {
	CommandPalette bool `yaml:"command_palette"`
	Atlas          bool `yaml:"atlas"`
	Weather        bool `yaml:"weather"`
	Plugins        bool `yaml:"plugins"`
}

// Defaults returns a safe configuration for a new local installation.
func Defaults() Config {
	return Config{App: AppConfig{Name: "NightOps", DataDir: "~/.local/share/nightops", LogLevel: "info", Theme: "mission-control"}, Origin: OriginConfig{Mode: "home_base", HomeBaseName: "Home Base", Timezone: "UTC"}, Obsidian: ObsidianConfig{Enabled: true, VaultDir: "~/Documents/Obsidian", NotesDir: "NightOps"}, Features: FeatureConfig{CommandPalette: true}}
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
	if c.Origin.Mode != "home_base" && c.Origin.Mode != "zip" && c.Origin.Mode != "gps" {
		return errors.New("origin mode must be home_base, zip, or gps")
	}
	if c.Origin.Latitude < -90 || c.Origin.Latitude > 90 {
		return errors.New("origin latitude must be between -90 and 90")
	}
	if c.Origin.Longitude < -180 || c.Origin.Longitude > 180 {
		return errors.New("origin longitude must be between -180 and 180")
	}
	return nil
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

package main

import (
	"path/filepath"
	"testing"

	"github.com/jordanistan/nightops/internal/config"
)

func TestLoadConfigUsesPersistedDefaultWhenPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.Defaults()
	cfg.Origin.HomeBaseName = "Persisted Base"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadConfig("", path, config.Defaults())
	if err != nil || loaded.Origin.HomeBaseName != "Persisted Base" {
		t.Fatalf("loaded config=%+v err=%v", loaded.Origin, err)
	}
	if loaded.App.ConfigPath != path {
		t.Fatalf("persisted config path=%q, want %q", loaded.App.ConfigPath, path)
	}
}

func TestLoadConfigKeepsDefaultsWhenPersistedFileIsAbsent(t *testing.T) {
	fallback := config.Defaults()
	defaultPath := filepath.Join(t.TempDir(), "missing.yaml")
	loaded, err := loadConfig("", defaultPath, fallback)
	if err != nil || loaded.App.Name != fallback.App.Name {
		t.Fatalf("fallback config=%+v err=%v", loaded.App, err)
	}
	if loaded.App.ConfigPath != defaultPath {
		t.Fatalf("fallback config path=%q, want %q", loaded.App.ConfigPath, defaultPath)
	}
}

func TestLoadConfigExplicitPathTakesPrecedence(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.yaml")
	explicitPath := filepath.Join(t.TempDir(), "explicit.yaml")
	defaultConfig := config.Defaults()
	defaultConfig.Origin.HomeBaseName = "Default Base"
	explicitConfig := config.Defaults()
	explicitConfig.Origin.HomeBaseName = "Explicit Base"
	if err := config.Save(defaultPath, defaultConfig); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(explicitPath, explicitConfig); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadConfig(explicitPath, defaultPath, config.Defaults())
	if err != nil || loaded.Origin.HomeBaseName != "Explicit Base" {
		t.Fatalf("explicit config=%+v err=%v", loaded.Origin, err)
	}
}

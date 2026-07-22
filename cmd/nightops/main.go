package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/jordanistan/nightops/internal/app"
	"github.com/jordanistan/nightops/internal/config"
)

func main() {
	configPath := flag.String("config", "", "path to NightOps YAML configuration")
	flag.Parse()
	cfg := config.Defaults()
	loaded, err := loadConfig(*configPath, config.ExpandPath(filepath.Join(cfg.App.DataDir, "config.yaml")), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration:", err)
		os.Exit(1)
	}
	cfg = loaded
	log.SetLevel(parseLevel(cfg.App.LogLevel))
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, "nightops:", err)
		os.Exit(1)
	}
}

func loadConfig(explicitPath, defaultPath string, fallback config.Config) (config.Config, error) {
	path := explicitPath
	if path == "" {
		if _, err := os.Stat(defaultPath); err != nil {
			if os.IsNotExist(err) {
				fallback.App.ConfigPath = defaultPath
				return fallback, nil
			}
			return config.Config{}, fmt.Errorf("inspect default configuration: %w", err)
		}
		path = defaultPath
	}
	loaded, err := config.Load(path)
	if err != nil {
		return config.Config{}, err
	}
	loaded.App.ConfigPath = path
	return loaded, nil
}

func parseLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

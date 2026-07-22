package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/jordanistan/nightops/internal/app"
	"github.com/jordanistan/nightops/internal/config"
)

func main() {
	configPath := flag.String("config", "", "path to NightOps YAML configuration")
	flag.Parse()
	cfg := config.Defaults()
	if *configPath != "" {
		loaded, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "configuration:", err)
			os.Exit(1)
		}
		cfg = loaded
	}
	log.SetLevel(parseLevel(cfg.App.LogLevel))
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, "nightops:", err)
		os.Exit(1)
	}
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

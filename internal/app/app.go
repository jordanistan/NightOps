// Package app composes the NightOps runtime.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/jordanistan/nightops/internal/config"
	"github.com/jordanistan/nightops/internal/console"
	"github.com/jordanistan/nightops/internal/storage/sqlite"
)

// Runtime owns resources shared by the process lifecycle.
type Runtime struct{ store *sqlite.Store }

// Run opens local resources and starts the interactive console.
func Run(ctx context.Context, cfg config.Config) error {
	dataDir := config.ExpandPath(cfg.App.DataDir)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	store, err := sqlite.Open(ctx, filepath.Join(dataDir, "nightops.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	log.Info("local database ready", "path", filepath.Join(dataDir, "nightops.db"))
	model := console.New(console.MissionControlTheme())
	program := tea.NewProgram(model, tea.WithContext(ctx))
	_, err = program.Run()
	return err
}

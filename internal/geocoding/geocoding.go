// Package geocoding defines the optional ZIP-to-coordinate boundary and its
// durable local cache.
package geocoding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var zipPattern = regexp.MustCompile(`^\d{5}(?:-\d{4})?$`)

// Result is a provider-sourced coordinate result for a US ZIP code.
type Result struct {
	ZIP         string    `json:"zip"`
	Label       string    `json:"label"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RetrievedAt time.Time `json:"retrieved_at"`
}

// Validate checks the result before it can be cached or used for planning.
func (r Result) Validate() error {
	if !zipPattern.MatchString(strings.TrimSpace(r.ZIP)) {
		return fmt.Errorf("invalid ZIP %q", r.ZIP)
	}
	if strings.TrimSpace(r.Label) == "" {
		return errors.New("geocoding result label is required")
	}
	if r.Latitude < -90 || r.Latitude > 90 || r.Longitude < -180 || r.Longitude > 180 {
		return fmt.Errorf("geocoding coordinates out of range: %.6f, %.6f", r.Latitude, r.Longitude)
	}
	if r.RetrievedAt.IsZero() {
		return errors.New("geocoding result timestamp is required")
	}
	return nil
}

// Fresh reports whether the result is within the configured cache lifetime.
func (r Result) Fresh(now time.Time, maxAge time.Duration) bool {
	return maxAge > 0 && !r.RetrievedAt.IsZero() && !r.RetrievedAt.After(now) && now.Sub(r.RetrievedAt) <= maxAge
}

// Provider resolves a ZIP code using an optional external or local adapter.
type Provider interface {
	Resolve(context.Context, string) (Result, error)
}

// Cache persists resolved ZIP results for offline reuse.
type Cache interface {
	Find(string) (Result, error)
	Save(Result) error
}

// FileCache is an owner-readable, atomically replaced JSON cache.
type FileCache struct {
	Path string
	mu   sync.Mutex
}

// NewFileCache creates a cache at path.
func NewFileCache(path string) *FileCache { return &FileCache{Path: path} }

// Find retrieves a cached result by ZIP code.
func (c *FileCache) Find(zip string) (Result, error) {
	if c == nil || strings.TrimSpace(c.Path) == "" {
		return Result{}, errors.New("geocoding cache path is required")
	}
	zip = strings.TrimSpace(zip)
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := c.load()
	if err != nil {
		return Result{}, err
	}
	result, ok := entries[zip]
	if !ok {
		return Result{}, os.ErrNotExist
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

// Save validates and atomically stores a result.
func (c *FileCache) Save(result Result) error {
	if c == nil || strings.TrimSpace(c.Path) == "" {
		return errors.New("geocoding cache path is required")
	}
	if err := result.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := c.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if entries == nil {
		entries = make(map[string]Result)
	}
	entries[result.ZIP] = result
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode geocoding cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o700); err != nil {
		return fmt.Errorf("create geocoding cache directory: %w", err)
	}
	temporary := c.Path + ".tmp"
	if err := os.WriteFile(temporary, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write geocoding cache: %w", err)
	}
	if err := os.Rename(temporary, c.Path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace geocoding cache: %w", err)
	}
	return nil
}

func (c *FileCache) load() (map[string]Result, error) {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]Result)
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode geocoding cache: %w", err)
	}
	return entries, nil
}

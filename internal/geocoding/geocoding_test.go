package geocoding

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCacheRoundTripIsOwnerReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geocoding", "cache.json")
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	want := Result{ZIP: "78701", Label: "Austin, Texas", Latitude: 30.2672, Longitude: -97.7431, RetrievedAt: now}
	cache := NewFileCache(path)
	if err := cache.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Find(want.ZIP)
	if err != nil || got != want {
		t.Fatalf("cache result=%+v err=%v want=%+v", got, err, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("cache stat error=%v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions=%v", info.Mode().Perm())
	}
	if !got.Fresh(now.Add(time.Hour), 24*time.Hour) {
		t.Fatal("fresh cached result was considered stale")
	}
}

func TestResultRejectsInvalidCoordinatesAndZIP(t *testing.T) {
	result := Result{ZIP: "Austin", Label: "Austin", Latitude: 30, Longitude: -97, RetrievedAt: time.Now()}
	if err := result.Validate(); err == nil {
		t.Fatal("invalid ZIP was accepted")
	}
	result.ZIP = "78701"
	result.Latitude = 91
	if err := result.Validate(); err == nil {
		t.Fatal("invalid latitude was accepted")
	}
}

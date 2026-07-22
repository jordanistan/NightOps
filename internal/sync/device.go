package sync

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadOrCreateDeviceID returns the stable local identity used in exported bundles.
// The identifier is metadata, not a credential, and is stored owner-readable.
func LoadOrCreateDeviceID(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("device ID path is required")
	}
	if data, err := os.ReadFile(path); err == nil {
		value := strings.TrimSpace(string(data))
		if value != "" {
			return value, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read device ID: %w", err)
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate device ID: %w", err)
	}
	value := hex.EncodeToString(bytes)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create device ID directory: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(value+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write device ID: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return "", fmt.Errorf("replace device ID: %w", err)
	}
	return value, nil
}

// Package plugins discovers local plugin manifests without executing plugin code.
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,63}$`)

// Manifest describes a locally installed plugin and its declared capabilities.
// NightOps treats this as metadata only; it does not load or execute Entrypoint.
type Manifest struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	Description  string   `yaml:"description"`
	Entrypoint   string   `yaml:"entrypoint"`
	Capabilities []string `yaml:"capabilities"`
}

// LoadDirectory reads one manifest.yaml from each direct child directory.
func LoadDirectory(path string) ([]Manifest, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("plugin directory is required")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	manifests := make([]Manifest, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(path, entry.Name(), "manifest.yaml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read plugin %q: %w", entry.Name(), err)
		}
		var manifest Manifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("decode plugin %q: %w", entry.Name(), err)
		}
		if err := manifest.Validate(); err != nil {
			return nil, fmt.Errorf("plugin %q: %w", entry.Name(), err)
		}
		manifests = append(manifests, manifest)
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].ID < manifests[j].ID })
	return manifests, nil
}

// Validate checks the metadata required before a plugin can be displayed.
func (m Manifest) Validate() error {
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("id must be lowercase kebab-case and 2-64 characters")
	}
	if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("name and version are required")
	}
	if strings.TrimSpace(m.Entrypoint) == "" {
		return fmt.Errorf("entrypoint is required")
	}
	seen := map[string]struct{}{}
	for _, capability := range m.Capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return fmt.Errorf("capabilities cannot contain empty values")
		}
		if _, ok := seen[capability]; ok {
			return fmt.Errorf("capability %q is duplicated", capability)
		}
		seen[capability] = struct{}{}
	}
	return nil
}

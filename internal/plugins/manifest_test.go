package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDirectorySortsAndValidatesManifests(t *testing.T) {
	dir := t.TempDir()
	for id, body := range map[string]string{
		"zeta":  "id: zeta\nname: Zeta\nversion: 1.0.0\ndescription: test\nentrypoint: zeta\ncapabilities: [targets]\n",
		"alpha": "id: alpha\nname: Alpha\nversion: 2.0.0\ndescription: test\nentrypoint: alpha\ncapabilities: [weather, atlas]\n",
	} {
		pluginDir := filepath.Join(dir, id)
		if err := os.Mkdir(pluginDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "manifest.yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifests, err := LoadDirectory(dir)
	if err != nil || len(manifests) != 2 || manifests[0].ID != "alpha" || manifests[1].ID != "zeta" {
		t.Fatalf("plugin manifests were not loaded deterministically: %+v err=%v", manifests, err)
	}
}

func TestLoadDirectoryRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad")
	if err := os.Mkdir(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.yaml"), []byte("id: Bad\nname: Bad\nversion: 1\nentrypoint: run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadDirectory(dir)
	if err == nil || !strings.Contains(err.Error(), "lowercase kebab-case") {
		t.Fatalf("invalid plugin manifest was accepted: %v", err)
	}
}

func TestManifestRejectsDuplicateCapabilities(t *testing.T) {
	manifest := Manifest{ID: "valid-plugin", Name: "Valid", Version: "1", Entrypoint: "run", Capabilities: []string{"atlas", "atlas"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("expected duplicate capability validation error")
	}
}

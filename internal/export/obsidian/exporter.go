// Package obsidian renders mission records into an Obsidian-compatible vault.
package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
)

// Exporter writes canonical mission projections to a vault directory.
type Exporter struct {
	vaultDir string
	notesDir string
}

// New creates an exporter rooted at vaultDir.
func New(vaultDir, notesDir string) Exporter { return Exporter{vaultDir: vaultDir, notesDir: notesDir} }

// Mission writes one mission note using an atomic replacement.
func (e Exporter) Mission(mission domain.Mission, site domain.LaunchSite) error {
	dir := filepath.Join(e.vaultDir, e.notesDir, "Missions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(mission.Name)+".md")
	temp, err := os.CreateTemp(dir, ".mission-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	content := fmt.Sprintf("---\nid: %s\nstatus: %s\nlaunch_site: [[%s]]\ncreated_at: %s\n---\n\n# %s\n\n## Launch Site\n\n- Latitude: %.6f\n- Longitude: %.6f\n- Timezone: %s\n\n## Flight Recorder\n\nMission status: **%s**\n", mission.ID, mission.Status, site.Name, mission.CreatedAt.UTC().Format(time.RFC3339), mission.Name, site.Latitude, site.Longitude, site.Timezone, mission.Status)
	if _, err := temp.WriteString(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "mission"
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

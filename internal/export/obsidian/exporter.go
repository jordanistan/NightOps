// Package obsidian renders mission records into an Obsidian-compatible vault.
package obsidian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/projection"
	"github.com/jordanistan/nightops/internal/weather"
)

// Exporter writes canonical mission projections to a vault directory.
type Exporter struct {
	vaultDir string
	notesDir string
}

// Export writes a mission projection and satisfies the application export port.
func (e Exporter) Export(_ context.Context, mission domain.Mission, site domain.LaunchSite) error {
	if err := e.Location(site); err != nil {
		return err
	}
	return e.Mission(mission, site)
}

// ExportObservation appends a flight-recorder entry to the mission note.
func (e Exporter) ExportObservation(_ context.Context, mission domain.Mission, site domain.LaunchSite, observation domain.Observation) error {
	if err := e.ensureMissionNote(mission, site); err != nil {
		return err
	}
	if err := e.Target(mission, observation.TargetName); err != nil {
		return err
	}
	path := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name)+".md")
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(existing) + fmt.Sprintf("\n## Observation\n\n- Target: %s ([[Targets/%s]])\n- Recorded: %s\n- Notes: %s\n", safeMarkdown(observation.TargetName), safeName(observation.TargetName), observation.CreatedAt.UTC().Format(time.RFC3339), safeMarkdown(observation.Notes))
	return atomicWrite(path, content)
}

// ExportMissionTarget creates or updates a catalog target note with its
// mission sequence position and backlink without duplicating existing links.
func (e Exporter) ExportMissionTarget(_ context.Context, mission domain.Mission, site domain.LaunchSite, target domain.MissionTarget) error {
	if err := e.ensureMissionNote(mission, site); err != nil {
		return err
	}
	dir := filepath.Join(e.vaultDir, e.notesDir, "Targets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(target.Name)+".md")
	existing, _ := os.ReadFile(path)
	content := string(existing)
	if content == "" {
		content = fmt.Sprintf("---\nid: %s\nname: %s\nkind: %s\nright_ascension_deg: %.6f\ndeclination_deg: %.6f\nsource: %s\n---\n\n# %s\n\n## Missions\n", safeMarkdown(target.ID), safeMarkdown(target.Name), safeMarkdown(target.Kind), target.RightAscension, target.Declination, safeMarkdown(target.Source), safeMarkdown(target.Name))
	}
	link := fmt.Sprintf("- [[Missions/%s]]", safeName(mission.Name))
	if !strings.Contains(content, link) {
		content = strings.TrimRight(content, "\n") + "\n" + link + "\n"
	}
	if err := atomicWrite(path, content); err != nil {
		return err
	}
	missionTargetDir := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name), "Targets")
	if err := os.MkdirAll(missionTargetDir, 0o755); err != nil {
		return err
	}
	missionTargetPath := filepath.Join(missionTargetDir, safeName(target.Name)+".md")
	missionTargetContent := fmt.Sprintf("---\nmission_id: %s\ntarget_id: %s\nposition: %d\n---\n\n# %s\n\n- Mission: [[Missions/%s]]\n- Catalog target: [[Targets/%s]]\n- Kind: %s\n- Right ascension: %.6f°\n- Declination: %.6f°\n- Source: %s\n", safeMarkdown(mission.ID), safeMarkdown(target.ID), target.Position+1, safeMarkdown(target.Name), safeName(mission.Name), safeName(target.Name), safeMarkdown(target.Kind), target.RightAscension, target.Declination, safeMarkdown(target.Source))
	return atomicWrite(missionTargetPath, missionTargetContent)
}

// ExportMissionEquipment writes a reusable equipment profile and a mission-scoped
// copy that preserves the exact setup used for the mission.
func (e Exporter) ExportMissionEquipment(_ context.Context, mission domain.Mission, site domain.LaunchSite, profile domain.EquipmentProfile, items []domain.EquipmentItem) error {
	if err := e.ensureMissionNote(mission, site); err != nil {
		return err
	}
	profileName := safeName(profile.Name)
	globalDir := filepath.Join(e.vaultDir, e.notesDir, "Equipment")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		return err
	}
	globalPath := filepath.Join(globalDir, profileName+".md")
	existingGlobal, _ := os.ReadFile(globalPath)
	globalContent := equipmentNote(profile, items, fmt.Sprintf("[[Missions/%s]]", safeName(mission.Name))) + preservedNotes(string(existingGlobal))
	if err := atomicWrite(globalPath, globalContent); err != nil {
		return err
	}

	missionDir := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name), "Equipment")
	if err := os.MkdirAll(missionDir, 0o755); err != nil {
		return err
	}
	missionContent := equipmentNote(profile, items, fmt.Sprintf("[[Missions/%s]]", safeName(mission.Name)))
	missionContent = strings.Replace(missionContent, "id: "+safeMarkdown(profile.ID)+"\n", "mission_id: "+safeMarkdown(mission.ID)+"\nprofile_id: "+safeMarkdown(profile.ID)+"\n", 1)
	return atomicWrite(filepath.Join(missionDir, profileName+".md"), missionContent)
}

// ExportMissionProjection writes the complete live/cached mission knowledge
// graph in one idempotent operation. It is intentionally an optional richer
// boundary so the canonical mission workflow remains usable without Obsidian.
func (e Exporter) ExportMissionProjection(ctx context.Context, mission domain.Mission, site domain.LaunchSite, projection projection.Mission) error {
	if err := e.Mission(mission, site); err != nil {
		return err
	}
	path := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name)+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	contentText := replaceBeforeFlightRecorder(string(content), richMissionSections(mission, site, projection))
	if err := atomicWrite(path, contentText); err != nil {
		return err
	}
	for _, target := range projection.Targets {
		if err := e.writeTargetKnowledge(target, projection.TargetKnowledge[target.ID], mission, site); err != nil {
			return err
		}
		if err := e.writeMissionTarget(target, projection.TargetKnowledge[target.ID], mission); err != nil {
			return err
		}
	}
	if projection.Equipment != nil {
		if err := e.ExportMissionEquipment(ctx, mission, site, *projection.Equipment, projection.EquipmentItems); err != nil {
			return err
		}
	}
	return e.writeIndexes(mission, site, projection)
}

// ExportDebrief writes an idempotent debrief section to the mission note.
func (e Exporter) ExportDebrief(_ context.Context, mission domain.Mission, site domain.LaunchSite, debrief domain.Debrief) error {
	if err := e.ensureMissionNote(mission, site); err != nil {
		return err
	}
	path := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name)+".md")
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(existing)
	if marker := strings.Index(content, "\n## Debrief"); marker >= 0 {
		content = content[:marker]
	}
	content = strings.TrimRight(content, "\n") + fmt.Sprintf("\n\n## Debrief\n\n- Recorded: %s\n- Summary: %s\n", debrief.CreatedAt.UTC().Format(time.RFC3339), safeMarkdown(debrief.Summary))
	return atomicWrite(path, content)
}

// New creates an exporter rooted at vaultDir.
func New(vaultDir, notesDir string) Exporter { return Exporter{vaultDir: vaultDir, notesDir: notesDir} }

// Mission writes one mission note using an atomic replacement.
func (e Exporter) Mission(mission domain.Mission, site domain.LaunchSite) error {
	if err := e.Location(site); err != nil {
		return err
	}
	dir := filepath.Join(e.vaultDir, e.notesDir, "Missions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(mission.Name)+".md")
	existing, _ := os.ReadFile(path)
	previousRecorder := ""
	if marker := strings.Index(string(existing), "\n## Observation"); marker >= 0 {
		previousRecorder = string(existing)[marker:]
	}
	temp, err := os.CreateTemp(dir, ".mission-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	latitude, longitude := "unknown", "unknown"
	if site.Latitude != nil {
		latitude = fmt.Sprintf("%.6f", *site.Latitude)
	}
	if site.Longitude != nil {
		longitude = fmt.Sprintf("%.6f", *site.Longitude)
	}
	equipment := "none"
	if mission.EquipmentProfileID != "" {
		equipment = safeMarkdown(mission.EquipmentProfileID)
	}
	plannedStart, plannedEnd := "none", "none"
	if mission.PlannedStart != nil {
		plannedStart = mission.PlannedStart.UTC().Format(time.RFC3339)
	}
	if mission.PlannedEnd != nil {
		plannedEnd = mission.PlannedEnd.UTC().Format(time.RFC3339)
	}
	content := fmt.Sprintf("---\nid: %s\nname: %s\nstatus: %s\nlaunch_site: [[Locations/%s]]\nequipment_profile_id: %s\nmission_date: %s\nlive_session: %t\nplanned_start: %s\nplanned_end: %s\ncreated_at: %s\nupdated_at: %s\n---\n\n# %s\n\n## Launch Site\n\n- Date created: %s\n- Latitude: %s\n- Longitude: %s\n- Timezone: %s\n\n## Mission Window\n\n- Start: %s\n- End: %s\n\n## Flight Recorder\n\nMission status: **%s**\n%s", safeMarkdown(mission.ID), safeMarkdown(mission.Name), mission.Status, safeName(site.Name), equipment, missionDate(mission, site), mission.PlannedStart == nil, plannedStart, plannedEnd, mission.CreatedAt.UTC().Format(time.RFC3339), mission.UpdatedAt.UTC().Format(time.RFC3339), safeMarkdown(mission.Name), missionDate(mission, site), latitude, longitude, safeMarkdown(site.Timezone), plannedStart, plannedEnd, mission.Status, previousRecorder)
	if _, err := temp.WriteString(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func (e Exporter) ensureMissionNote(mission domain.Mission, site domain.LaunchSite) error {
	path := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name)+".md")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return e.Mission(mission, site)
}

func richMissionSections(mission domain.Mission, site domain.LaunchSite, projection projection.Mission) string {
	var body strings.Builder
	body.WriteString("\n\n## Live Conditions\n\n")
	if projection.Weather == nil {
		body.WriteString("Weather: **Unavailable** — no cached or live snapshot was available at export time.\n")
	} else {
		body.WriteString(weatherReport(*projection.Weather))
	}
	body.WriteString("\n## Equipment Checklist\n\n")
	if projection.Equipment == nil {
		body.WriteString("- [ ] No equipment profile selected\n")
	} else {
		fmt.Fprintf(&body, "- Profile: **%s** ([[Equipment/%s]])\n", safeMarkdown(projection.Equipment.Name), safeName(projection.Equipment.Name))
		if len(projection.EquipmentItems) == 0 {
			body.WriteString("- [ ] No inventory items recorded\n")
		} else {
			for _, item := range projection.EquipmentItems {
				marker := "[ ]"
				if !item.Required {
					marker = "[ ]"
				}
				fmt.Fprintf(&body, "- %s **%s** — %s (%s)\n", marker, safeMarkdown(item.Name), safeMarkdown(item.Category), requirement(item.Required))
			}
		}
	}
	body.WriteString("\n## Selected Targets\n\n")
	if len(projection.Targets) == 0 {
		body.WriteString("No catalog targets selected.\n")
	} else {
		for index, target := range projection.Targets {
			knowledge := projection.TargetKnowledge[target.ID]
			fmt.Fprintf(&body, "%d. [[Targets/%s]] — %s\n", index+1, safeName(target.Name), captureGuidance(target.Kind))
			if knowledge.Status == "unavailable" {
				body.WriteString("   - Reference: **Unavailable**\n")
			} else {
				fmt.Fprintf(&body, "   - Reference: [[Targets/%s]] · %s\n", safeName(target.Name), nonEmpty(knowledge.Status, "cached"))
			}
		}
	}
	body.WriteString(fmt.Sprintf("\n- Launch site: [[Locations/%s]]\n- Mission date: %s\n", safeName(site.Name), missionDate(mission, site)))
	return body.String()
}

func weatherReport(snapshot weather.Snapshot) string {
	var body strings.Builder
	freshness := "stale cache"
	if snapshot.Fresh(time.Now().UTC()) {
		freshness = "fresh snapshot"
	}
	fmt.Fprintf(&body, "- Source: **%s** (%s)\n- Observed: %s\n", safeMarkdown(snapshot.Source), freshness, snapshot.ObservedAt.UTC().Format(time.RFC3339))
	if snapshot.TemperatureC != nil {
		fmt.Fprintf(&body, "- Temperature: %.1f°C\n", *snapshot.TemperatureC)
	} else {
		body.WriteString("- Temperature: unavailable\n")
	}
	if snapshot.CloudCoverPercent != nil {
		fmt.Fprintf(&body, "- Cloud cover: %.0f%%\n", *snapshot.CloudCoverPercent)
	} else {
		body.WriteString("- Cloud cover: unavailable\n")
	}
	if len(snapshot.Forecast) > 0 {
		body.WriteString("\n### Hourly Forecast\n\n| Time (UTC) | Temperature | Clouds | Precipitation |\n| --- | ---: | ---: | ---: |\n")
		for _, point := range snapshot.Forecast {
			body.WriteString("| " + point.At.UTC().Format("2006-01-02 15:04") + " | " + floatOrUnknown(point.TemperatureC, "°C") + " | " + floatOrUnknown(point.CloudCoverPercent, "%") + " | " + floatOrUnknown(point.PrecipitationProbability, "%") + " |\n")
		}
	}
	return body.String()
}

func (e Exporter) writeTargetKnowledge(target domain.MissionTarget, knowledge domain.TargetKnowledge, mission domain.Mission, site domain.LaunchSite) error {
	dir := filepath.Join(e.vaultDir, e.notesDir, "Targets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(target.Name)+".md")
	existing, _ := os.ReadFile(path)
	content := string(existing)
	if content == "" {
		content = fmt.Sprintf("---\nid: %s\nname: %s\nkind: %s\nsource: %s\n---\n\n# %s\n", safeMarkdown(target.ID), safeMarkdown(target.Name), safeMarkdown(target.Kind), safeMarkdown(target.Source), safeMarkdown(target.Name))
	}
	content = ensureSection(content, "Reference", fmt.Sprintf("- Status: **%s**\n- Source: %s\n- Page: %s\n- Summary: %s\n", nonEmpty(knowledge.Status, "unavailable"), nonEmpty(knowledge.Source, "unavailable"), nonEmpty(linkOrUnknown(knowledge.URL), "unavailable"), nonEmpty(safeMarkdown(knowledge.Summary), "unavailable")))
	content = ensureSection(content, "Capture Guidance", "- "+captureGuidance(target.Kind)+"\n")
	if knowledge.ImageURL != "" {
		content = ensureSection(content, "Images", fmt.Sprintf("![](%s)\n\n- Source image: %s\n", knowledge.ImageURL, knowledge.ImageURL))
	}
	missionLine := fmt.Sprintf("- [[Missions/%s]] · %s · %s · %s", safeName(mission.Name), safeMarkdown(string(mission.Status)), safeMarkdown(site.Name), missionDate(mission, site))
	content = ensureListItem(content, "Missions", missionLine)
	return atomicWrite(path, content)
}

func (e Exporter) writeMissionTarget(target domain.MissionTarget, knowledge domain.TargetKnowledge, mission domain.Mission) error {
	dir := filepath.Join(e.vaultDir, e.notesDir, "Missions", safeName(mission.Name), "Targets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("---\nmission_id: %s\ntarget_id: %s\nposition: %d\nknowledge_status: %s\n---\n\n# %s\n\n- Mission: [[Missions/%s]]\n- Catalog target: [[Targets/%s]]\n- Kind: %s\n- Right ascension: %.6f°\n- Declination: %.6f°\n- Source: %s\n\n## Capture Guidance\n\n%s\n\n## Reference\n\n%s\n", safeMarkdown(mission.ID), safeMarkdown(target.ID), target.Position+1, nonEmpty(knowledge.Status, "unavailable"), safeMarkdown(target.Name), safeName(mission.Name), safeName(target.Name), safeMarkdown(target.Kind), target.RightAscension, target.Declination, safeMarkdown(target.Source), captureGuidance(target.Kind), nonEmpty(safeMarkdown(knowledge.Summary), "Reference unavailable"))
	return atomicWrite(filepath.Join(dir, safeName(target.Name)+".md"), content)
}

func replaceBeforeFlightRecorder(content, generated string) string {
	marker := "\n## Flight Recorder"
	if index := strings.Index(content, marker); index >= 0 {
		return strings.TrimRight(content[:index], "\n") + generated + content[index:]
	}
	return strings.TrimRight(content, "\n") + generated + "\n\n## Flight Recorder\n"
}

func ensureSection(content, title, body string) string {
	marker := "\n## " + title
	section := marker + "\n\n" + strings.TrimRight(body, "\n") + "\n"
	if index := strings.Index(content, marker); index >= 0 {
		end := strings.Index(content[index+len(marker):], "\n## ")
		if end >= 0 {
			end += index + len(marker)
			return content[:index] + section + content[end:]
		}
		return content[:index] + section
	}
	return strings.TrimRight(content, "\n") + "\n" + section
}

func ensureListItem(content, title, item string) string {
	if strings.Contains(content, item) {
		return content
	}
	marker := "\n## " + title
	if index := strings.Index(content, marker); index >= 0 {
		end := strings.Index(content[index+len(marker):], "\n## ")
		if end >= 0 {
			end += index + len(marker)
			return content[:end] + "\n" + item + content[end:]
		}
		return strings.TrimRight(content, "\n") + "\n" + item + "\n"
	}
	return ensureSection(content, title, item)
}

func missionDate(mission domain.Mission, site domain.LaunchSite) string {
	location := time.UTC
	if site.Timezone != "" {
		if loaded, err := time.LoadLocation(site.Timezone); err == nil {
			location = loaded
		}
	}
	return mission.CreatedAt.In(location).Format("2006-01-02")
}

func requirement(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func captureGuidance(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "galaxy":
		return "Use a wider field and moderate integration; preserve the bright core while exposing faint outer structure."
	case "nebula":
		return "Use a narrowband or light-pollution-aware filter when available; capture multiple shorter subs for the brightest regions."
	case "cluster":
		return "Use a field of view that includes the surrounding star field; keep stars sharp with short, well-focused subs."
	default:
		return "Confirm focus, framing, and exposure with a short test capture before committing the sequence."
	}
}

func floatOrUnknown(value *float64, suffix string) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f%s", *value, suffix)
}

func linkOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unavailable"
	}
	return "[source page](" + safeMarkdown(value) + ")"
}

func (e Exporter) writeIndexes(mission domain.Mission, site domain.LaunchSite, projection projection.Mission) error {
	root := filepath.Join(e.vaultDir, e.notesDir)
	indexes := []struct {
		dir   string
		title string
		line  string
	}{
		{root, "NightOps Mission Knowledge Base", fmt.Sprintf("- [[Missions/%s]] · %s · [[Locations/%s]] · %s", safeName(mission.Name), safeMarkdown(string(mission.Status)), safeName(site.Name), missionDate(mission, site))},
		{filepath.Join(root, "Missions"), "Missions", fmt.Sprintf("- [[%s]] · %s · %s", safeName(mission.Name), safeMarkdown(string(mission.Status)), missionDate(mission, site))},
		{filepath.Join(root, "Locations"), "Locations", fmt.Sprintf("- [[%s]] · %s", safeName(site.Name), safeMarkdown(site.Source))},
	}
	if projection.Equipment != nil {
		indexes = append(indexes, struct {
			dir   string
			title string
			line  string
		}{filepath.Join(root, "Equipment"), "Equipment", fmt.Sprintf("- [[%s]] · [[Missions/%s]]", safeName(projection.Equipment.Name), safeName(mission.Name))})
	}
	for _, target := range projection.Targets {
		indexes = append(indexes, struct {
			dir   string
			title string
			line  string
		}{filepath.Join(root, "Targets"), "Targets", fmt.Sprintf("- [[%s]] · [[Missions/%s]]", safeName(target.Name), safeName(mission.Name))})
	}
	for _, index := range indexes {
		if err := os.MkdirAll(index.dir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(index.dir, "Index.md")
		content, _ := os.ReadFile(path)
		if len(content) == 0 {
			content = []byte(fmt.Sprintf("---\ntype: index\nsection: %s\n---\n\n# %s\n", safeMarkdown(index.title), safeMarkdown(index.title)))
		}
		updated := ensureListItem(string(content), "Records", index.line)
		if err := atomicWrite(path, updated); err != nil {
			return err
		}
	}
	return nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// Location writes a linked launch-site note while preserving any user notes.
func (e Exporter) Location(site domain.LaunchSite) error {
	dir := filepath.Join(e.vaultDir, e.notesDir, "Locations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(site.Name)+".md")
	existing, _ := os.ReadFile(path)
	notes := preservedNotes(string(existing))
	latitude, longitude := "unknown", "unknown"
	if site.Latitude != nil {
		latitude = fmt.Sprintf("%.6f", *site.Latitude)
	}
	if site.Longitude != nil {
		longitude = fmt.Sprintf("%.6f", *site.Longitude)
	}
	content := fmt.Sprintf("---\nid: %s\nsource: %s\nlatitude: %s\nlongitude: %s\ntimezone: %s\n---\n\n# %s\n\n- Latitude: %s\n- Longitude: %s\n- Timezone: %s\n- Source: %s\n%s", safeMarkdown(site.ID), safeMarkdown(site.Source), latitude, longitude, safeMarkdown(site.Timezone), safeMarkdown(site.Name), latitude, longitude, safeMarkdown(site.Timezone), safeMarkdown(site.Source), notes)
	return atomicWrite(path, content)
}

// Target writes a target note and adds a backlink to the mission that recorded it.
func (e Exporter) Target(mission domain.Mission, targetName string) error {
	dir := filepath.Join(e.vaultDir, e.notesDir, "Targets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName(targetName)+".md")
	existing, _ := os.ReadFile(path)
	content := string(existing)
	if content == "" {
		content = fmt.Sprintf("---\nname: %s\n---\n\n# %s\n\n## Missions\n", safeMarkdown(targetName), safeMarkdown(targetName))
	}
	link := fmt.Sprintf("- [[Missions/%s]]", safeName(mission.Name))
	if !strings.Contains(content, link) {
		content += "\n" + link + "\n"
	}
	return atomicWrite(path, content)
}

func equipmentNote(profile domain.EquipmentProfile, items []domain.EquipmentItem, missionLink string) string {
	var body strings.Builder
	fmt.Fprintf(&body, "---\nid: %s\nname: %s\ndescription: %s\n---\n\n# %s\n\n- Mission: %s\n\n## Inventory\n\n", safeMarkdown(profile.ID), safeMarkdown(profile.Name), safeMarkdown(profile.Description), safeMarkdown(profile.Name), missionLink)
	if len(items) == 0 {
		body.WriteString("No inventory items recorded.\n")
		return body.String()
	}
	for _, item := range items {
		required := "optional"
		if item.Required {
			required = "required"
		}
		fmt.Fprintf(&body, "- **%s** — %s (%s)\n", safeMarkdown(item.Name), safeMarkdown(item.Category), required)
	}
	return body.String()
}

func atomicWrite(path, content string) error {
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".note-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.WriteString(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}

func preservedNotes(existing string) string {
	if marker := strings.Index(existing, "\n## Notes"); marker >= 0 {
		return existing[marker:]
	}
	return ""
}

func safeMarkdown(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(value))
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
	value := b.String()
	if value == "" || value == "." || value == ".." {
		return "note"
	}
	return value
}

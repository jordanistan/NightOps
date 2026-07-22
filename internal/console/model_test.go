package console

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func enterKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }
func downKey() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyDown} }

func typeText(model Model, value string) Model {
	for _, character := range value {
		updated, _ := model.Update(runeKey(character))
		model = updated.(Model)
	}
	return model
}

func skipBlastOff(model Model) Model {
	if model.route == RouteMissionReview {
		updated, _ := model.Update(enterKey())
		model = updated.(Model)
	}
	for range 10 {
		updated, _ := model.Update(blastOffTickMsg{})
		model = updated.(Model)
	}
	return model
}

func launchModel(options ...Options) Model {
	model := New(MissionControlTheme(), options...)
	updated, _ := model.Update(runeKey(' '))
	return updated.(Model)
}

func TestBootTransitionsToLaunch(t *testing.T) {
	model := New(MissionControlTheme())
	if !strings.Contains(model.View(), "NIGHTOPS SYSTEM INITIALIZATION") {
		t.Fatal("boot screen missing initialization heading")
	}
	for range bootStages {
		updated, _ := model.Update(bootTickMsg{})
		model = updated.(Model)
	}
	if model.route != RouteLaunch {
		t.Fatalf("boot did not transition to launch: %s", model.route)
	}
}

func TestBootCanBeSkipped(t *testing.T) {
	model := launchModel()
	if model.route != RouteLaunch {
		t.Fatal("space did not skip boot")
	}
}

func TestLaunchStartsWithTargetSelectionAndBuildsLiveMissionWindow(t *testing.T) {
	start := time.Date(2026, 7, 22, 21, 14, 0, 0, time.UTC)
	end := start.Add(6 * time.Hour)
	target := TargetSite{ID: "m31", Name: "Andromeda Galaxy", Kind: "galaxy"}
	model := launchModel(Options{
		HomeBaseName:       "Home",
		HomeBaseConfigured: true,
		TargetsEnabled:     true,
		Targets:            []TargetSite{target},
		MissionWindow: func(_ Origin, selected []TargetSite) (*time.Time, *time.Time, string) {
			if len(selected) == 0 {
				return nil, nil, "select targets to calculate a target window"
			}
			return &start, &end, "live tonight · 1 target(s) evaluated above 30°"
		},
		WeatherAssessment: func(_ Origin, selected []TargetSite) string {
			return fmt.Sprintf("GO · %d/%d selected target(s) have a dark, weather-qualified window", len(selected), len(selected))
		},
	})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteTargetBrowser || !strings.Contains(model.View(), "TONIGHT'S TARGETS") {
		t.Fatalf("launch did not start with target selection: route=%s", model.route)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.plannedStart == nil || !strings.Contains(model.View(), "GO · 1/1") || !strings.Contains(model.View(), "live tonight") {
		t.Fatalf("target-first planning did not build automatic mission context: route=%s view=%s", model.route, model.View())
	}
}

func TestHomeBaseWithoutConfigurationOpensSetup(t *testing.T) {
	model := launchModel()
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	if updated.(Model).route != RouteHomeBaseSetup {
		t.Fatalf("home base did not open setup: %s", updated.(Model).route)
	}
}

func TestConfiguredHomeBaseOpensMissionPlanning(t *testing.T) {
	created := false
	model := launchModel(Options{HomeBaseName: "Clarksville", HomeBaseLocation: "Austin, Texas", HomeBaseConfigured: true, CreateMission: func(origin Origin, _ string, _, _ *time.Time) (string, error) {
		created = origin.Label == "Clarksville"
		return "mission-1", nil
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	mission := updated.(Model)
	if mission.route != RouteMissionPlanning {
		t.Fatalf("configured home base did not open planning: %s", mission.route)
	}
	if mission.missionPlan.origin.Label != "Clarksville" {
		t.Fatalf("origin was not passed to planning: %+v", mission.missionPlan.origin)
	}
	updated, _ = mission.Update(enterKey())
	mission = updated.(Model)
	if mission.route == RouteBlastOff {
		mission = skipBlastOff(mission)
	}
	mission = skipBlastOff(mission)
	if !created || mission.route != RouteDeepSpace || mission.operation.missionID != "mission-1" {
		t.Fatalf("mission confirmation did not enter operation: created=%v route=%s id=%q", created, mission.route, mission.operation.missionID)
	}
}

func TestMissionReviewPrecedesPersistenceAndLaunch(t *testing.T) {
	created := false
	model := launchModel(Options{
		HomeBaseName:       "Clarksville",
		HomeBaseConfigured: true,
		CreateMission: func(Origin, string, *time.Time, *time.Time) (string, error) {
			created = true
			return "mission-review", nil
		},
	})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionReview || created {
		t.Fatalf("planning did not stop at review: route=%s created=%v", model.route, created)
	}
	for _, expected := range []string{"FINAL REVIEW", "ORIGIN", "TARGET SEQUENCE", "OBSERVING WINDOW", "LAUNCH + OPEN OBSIDIAN", "LAUNCH + CONTINUE IN NIGHTOPS"} {
		if !strings.Contains(model.View(), expected) {
			t.Fatalf("review screen missing %q: %s", expected, model.View())
		}
	}
	updated, _ = model.Update(downKey())
	model = updated.(Model)
	updated, _ = model.Update(downKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning {
		t.Fatalf("review edit action did not return to planning: %s", model.route)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionReview {
		t.Fatalf("planning did not reopen review: %s", model.route)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteBlastOff || !created || model.operation.missionID != "mission-review" {
		t.Fatalf("review launch did not persist mission: route=%s created=%v id=%q", model.route, created, model.operation.missionID)
	}
}

func TestDeepSpaceDoesNotAdvertiseUnavailableTelescope(t *testing.T) {
	model := launchModel()
	model.route = RouteDeepSpace
	model.missionPlan.origin = Origin{Kind: "ZIP Code", Label: "ZIP 78701"}
	model.missionPlan.targets = []TargetSite{{ID: "m42", Name: "Orion Nebula", Kind: "nebula", Source: "fixture"}}
	if strings.Contains(model.View(), "Enter Slew") {
		t.Fatal("deep space advertised telescope control without an adapter")
	}
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteDeepSpace || !strings.Contains(model.missionPlan.status, "not configured") {
		t.Fatalf("unavailable telescope action was not handled honestly: route=%s status=%q", model.route, model.missionPlan.status)
	}
}

func TestDisabledAtlasCannotOpenImport(t *testing.T) {
	model := launchModel(Options{SaveAtlas: func(string, string) ([]AtlasSite, error) { return nil, nil }})
	model.route = RouteSettings
	updated, _ := model.Update(runeKey('i'))
	model = updated.(Model)
	if model.route != RouteSettings {
		t.Fatalf("disabled Atlas import opened from Settings: %s", model.route)
	}
}

func TestSettingsOpensConfiguredObsidianVault(t *testing.T) {
	opened := false
	model := launchModel(Options{OpenObsidianVault: func() error {
		opened = true
		return nil
	}})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('o'))
	model = updated.(Model)
	if model.route != RouteSettings || !opened || !strings.Contains(model.settings.notice, "opened") {
		t.Fatalf("Obsidian open action failed: route=%s opened=%v notice=%q", model.route, opened, model.settings.notice)
	}
}

func TestMissionScheduleFlowPersistsLocalWindow(t *testing.T) {
	var start, end *time.Time
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, ScheduleTimezone: "America/Chicago", CreateMission: func(_ Origin, _ string, plannedStart, plannedEnd *time.Time) (string, error) {
		start, end = plannedStart, plannedEnd
		return "mission-1", nil
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionSchedule {
		t.Fatalf("mission schedule did not open: %s", model.route)
	}
	model = typeText(model, "2026-07-22 21:00")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "2026-07-23 01:00")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.plannedStart == nil || !strings.Contains(model.View(), "MISSION WINDOW") {
		t.Fatalf("mission schedule did not return to planning: route=%s view=%s", model.route, model.View())
	}
	for range 4 {
		updated, _ = model.Update(runeKey('k'))
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	model = skipBlastOff(model)
	if model.route != RouteDeepSpace || start == nil || end == nil || end.Sub(*start) != 4*time.Hour {
		t.Fatalf("mission schedule was not passed to creation: route=%s start=%v end=%v", model.route, start, end)
	}
}

func TestOperationLifecycleAndObservationEntry(t *testing.T) {
	started, activated, recorded, completed, debriefed := false, false, false, false, ""
	model := launchModel(Options{
		HomeBaseName: "Clarksville", HomeBaseConfigured: true,
		CreateMission:     func(Origin, string, *time.Time, *time.Time) (string, error) { return "mission-1", nil },
		StartMission:      func(string) error { started = true; return nil },
		ActivateMission:   func(string) error { activated = true; return nil },
		RecordObservation: func(_, target, notes string) error { recorded = target == "M 42" && notes == "Clear view"; return nil },
		CompleteMission:   func(string) error { completed = true; return nil },
		RecordDebrief:     func(_, summary string) error { debriefed = summary; return nil },
	})
	for range 2 {
		updated, _ := model.Update(enterKey())
		model = updated.(Model)
	}
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	model = skipBlastOff(model)
	updated, _ = model.Update(runeKey('o'))
	model = updated.(Model)
	if model.route != RouteOperation {
		t.Fatalf("mission did not enter operation: %s", model.route)
	}
	for range 2 {
		updated, _ = model.Update(enterKey())
		model = updated.(Model)
	}
	if !started || !activated || model.operation.status != "ACTIVE" {
		t.Fatalf("operation did not activate: started=%v activated=%v status=%s", started, activated, model.operation.status)
	}
	updated, _ = model.Update(runeKey('a'))
	model = updated.(Model)
	if model.route != RouteObservationEntry {
		t.Fatalf("observation entry did not open: %s", model.route)
	}
	model = typeText(model, "M 42")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "Clear view")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !recorded || model.route != RouteOperation || len(model.operation.observations) != 1 {
		t.Fatalf("observation was not recorded: recorded=%v route=%s observations=%v", recorded, model.route, model.operation.observations)
	}
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if !completed || model.route != RouteDebrief || model.debrief.count != 1 {
		t.Fatalf("debrief did not open: completed=%v route=%s count=%d", completed, model.route, model.debrief.count)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteDebrief || model.debrief.error == "" {
		t.Fatalf("empty debrief was accepted: route=%s error=%q", model.route, model.debrief.error)
	}
	model = typeText(model, "Clear skies and excellent transparency.")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteDebrief || !model.debrief.saved || debriefed != "Clear skies and excellent transparency." {
		t.Fatalf("debrief was not saved: route=%s saved=%v summary=%q", model.route, model.debrief.saved, debriefed)
	}
	updated, _ = model.Update(enterKey())
	if updated.(Model).route != RouteLaunch {
		t.Fatalf("saved debrief did not return to launch: %s", updated.(Model).route)
	}
}

func TestDebriefPersistenceFailureStaysOnScreen(t *testing.T) {
	model := launchModel(Options{RecordDebrief: func(string, string) error { return fmt.Errorf("database offline") }})
	model.route = RouteDebrief
	model.debrief = newDebrief()
	model.debrief.missionID = "mission-1"
	model.debrief.summary.Focus()
	model = typeText(model, "A useful field note")
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteDebrief || model.debrief.saved || !strings.Contains(model.debrief.error, "database offline") {
		t.Fatalf("debrief persistence failure was not surfaced: route=%s saved=%v error=%q", model.route, model.debrief.saved, model.debrief.error)
	}
}

func TestHomeBaseSetupSavesAndOpensPlanning(t *testing.T) {
	saved := false
	model := launchModel(Options{SaveHomeBase: func(homeBase HomeBase) error {
		saved = homeBase.Name == "Dark Site" && homeBase.ZIP == "78701"
		return nil
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	model = typeText(model, "Dark Site")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "78701")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !saved || model.route != RouteMissionPlanning || !model.options.HomeBaseConfigured {
		t.Fatalf("home base save flow failed: saved=%v route=%s", saved, model.route)
	}
}

func TestHomeBaseSetupDoesNotClaimSaveWithoutPersistence(t *testing.T) {
	model := launchModel()
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	model = typeText(model, "Dark Site")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "78701")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteHomeBaseSetup || !strings.Contains(model.homeBase.error, "persistence") {
		t.Fatalf("missing Home Base persistence was not surfaced: route=%s error=%q", model.route, model.homeBase.error)
	}
}

func TestSaveAsHomeBaseDoesNotClaimSaveWithoutPersistence(t *testing.T) {
	model := launchModel()
	model.setMissionOrigin(Origin{Kind: "ZIP Code", Label: "ZIP 78701", ZIP: "78701"})
	model.route = RouteMissionPlanning
	model.missionPlan.selected = 2
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteError || !strings.Contains(model.error.detail, "persistence") {
		t.Fatalf("missing Home Base persistence was not surfaced from planning: route=%s error=%+v", model.route, model.error)
	}
}

func TestChangingHomeBaseZIPClearsStaleCoordinates(t *testing.T) {
	latitude, longitude := 30.2672, -97.7431
	model := launchModel(Options{HomeBaseZIP: "78701", HomeBaseLatitude: &latitude, HomeBaseLongitude: &longitude})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	model = typeText(model, "New Site")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "78620")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.missionPlan.origin.Latitude != nil || model.missionPlan.origin.Longitude != nil {
		t.Fatalf("changed Home Base ZIP retained stale coordinates: %+v", model.missionPlan.origin)
	}
}

func TestAtlasOriginPreservesSiteTimezone(t *testing.T) {
	model := launchModel(Options{AtlasEnabled: true, AtlasStatus: "READY", AtlasLocations: []AtlasSite{{ID: "site-1", Name: "West Site", Latitude: 30, Longitude: -97, Timezone: "America/Denver"}}})
	model.previousRoute, model.route = RouteLaunch, RouteAtlasBrowser
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.missionPlan.origin.Timezone != "America/Denver" {
		t.Fatalf("Atlas timezone was discarded: %+v", model.missionPlan.origin)
	}
}

func TestCurrentLocationOpensGPSAndUnavailableError(t *testing.T) {
	model := launchModel()
	model.beginOrigin("g")
	if model.route != RouteGPSAcquisition || model.gps.status != "UNAVAILABLE" {
		t.Fatalf("GPS unavailable did not show acquisition state: route=%s status=%s", model.route, model.gps.status)
	}
	if !strings.Contains(model.View(), "Enter ZIP Code Instead") {
		t.Fatal("GPS unavailable state did not offer ZIP fallback")
	}
	updated, _ := model.Update(gpsUnavailableMsg{})
	model = updated.(Model)
	if model.route != RouteError || !strings.Contains(model.error.title, "GPS") {
		t.Fatalf("GPS unavailable did not produce error route: route=%s error=%+v", model.route, model.error)
	}
	if !strings.Contains(model.View(), "No GPS adapter is configured") {
		t.Fatal("GPS error did not explain the unavailable adapter")
	}
}

func TestZIPValidSubmissionOpensMissionPlanning(t *testing.T) {
	model := launchModel()
	model.beginOrigin("z")
	if model.route != RouteZIPEntry {
		t.Fatalf("ZIP option did not open ZIP entry: %s", model.route)
	}
	model = typeText(model, "78701")
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.origin.ZIP != "78701" {
		t.Fatalf("valid ZIP did not open planning: route=%s origin=%+v", model.route, model.missionPlan.origin)
	}
	if !strings.Contains(model.View(), "unavailable until coordinates are known") {
		t.Fatal("planning should disclose unavailable coordinates")
	}
}

func TestZIPResolutionUsesProviderCoordinates(t *testing.T) {
	latitude, longitude := 30.2672, -97.7431
	model := launchModel(Options{ResolveZIP: func(zip string) (Origin, error) {
		return Origin{Kind: "ZIP Code", Label: "ZIP " + zip + " · Austin, Texas", ZIP: zip, Timezone: "America/Chicago", Latitude: &latitude, Longitude: &longitude}, nil
	}})
	model.beginOrigin("z")
	model = typeText(model, "78701")
	updated, command := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteZIPEntry || !model.zipEntry.pending || command == nil {
		t.Fatalf("ZIP provider request did not start: route=%s pending=%v command=%v", model.route, model.zipEntry.pending, command)
	}
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.origin.Latitude == nil || *model.missionPlan.origin.Latitude != latitude {
		t.Fatalf("ZIP provider result did not enter planning: route=%s origin=%+v", model.route, model.missionPlan.origin)
	}
}

func TestZIPResolutionFailureFallsBackToUnknownCoordinates(t *testing.T) {
	model := launchModel(Options{ResolveZIP: func(string) (Origin, error) { return Origin{}, fmt.Errorf("provider unavailable") }})
	model.beginOrigin("z")
	model = typeText(model, "78701")
	updated, command := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.origin.Latitude != nil || !strings.Contains(model.missionPlan.status, "unavailable") {
		t.Fatalf("ZIP failure did not preserve honest fallback: route=%s origin=%+v status=%q", model.route, model.missionPlan.origin, model.missionPlan.status)
	}
}

func TestZIPInvalidSubmissionStaysOnEntry(t *testing.T) {
	model := launchModel()
	model.beginOrigin("z")
	model.zipEntry.input.SetValue("")
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteZIPEntry || model.zipEntry.error == "" {
		t.Fatal("invalid ZIP did not stay on entry with an error")
	}
}

func TestEscapeReturnsToPreviousScreen(t *testing.T) {
	model := launchModel()
	updated, _ := model.Update(runeKey('?'))
	model = updated.(Model)
	if model.route != RouteHelp {
		t.Fatal("help did not open")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	if model.route != RouteLaunch {
		t.Fatal("escape did not return to launch")
	}
	updated, _ = model.Update(runeKey('s'))
	model = updated.(Model)
	if model.route != RouteSettings {
		t.Fatal("settings did not open")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(Model).route != RouteLaunch {
		t.Fatal("settings escape did not return to launch")
	}
}

func TestRouteTransitionsPersistAcrossMessages(t *testing.T) {
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	model = updated.(Model)
	if model.route != RouteMissionPlanning {
		t.Fatalf("window update overwrote route: %s", model.route)
	}
	if !strings.Contains(model.View(), "MISSION PLANNING") {
		t.Fatal("mission planning view was not retained")
	}
}

func TestStatusLabelsReflectCapabilities(t *testing.T) {
	model := launchModel(Options{DatabaseStatus: "READY", ObsidianStatus: "DISABLED", AtlasStatus: "NOT CONFIGURED", GPSStatus: "UNAVAILABLE"})
	view := model.View()
	for _, status := range []string{"READY", "DISABLED", "NOT CONFIGURED", "UNAVAILABLE"} {
		if !strings.Contains(view, status) {
			t.Errorf("status %q missing from launch view", status)
		}
	}
}

func TestMissionPlanningShowsWeatherSummary(t *testing.T) {
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, WeatherSummary: func(Origin) string { return "28.4°C · clouds 17% · open-meteo" }})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !strings.Contains(model.View(), "28.4°C · clouds 17% · open-meteo") {
		t.Fatal("mission planning view omitted weather summary")
	}
}

func TestMissionPlanningTargetBrowserSelectsTarget(t *testing.T) {
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, TargetsEnabled: true, Targets: []TargetSite{{ID: "m31", Name: "Andromeda Galaxy", Kind: "galaxy"}}, TargetSummary: func(Origin, TargetSite) string { return "30°+ 22:00 CDT–04:00 CDT · peak 78°" }, TargetForecastSummary: func(Origin, TargetSite, []ForecastPoint) string {
		return "score 84 · 22:00 CDT · target 68° · clouds 12% · precip 4%"
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteTargetBrowser {
		t.Fatalf("target action did not open target browser: %s", model.route)
	}
	updated, _ = model.Update(runeKey(' '))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.target != "Andromeda Galaxy" || !strings.Contains(model.View(), "30°+ 22:00 CDT") || !strings.Contains(model.View(), "score 84") {
		t.Fatalf("target selection did not return to planning: route=%s target=%q view=%s", model.route, model.missionPlan.target, model.View())
	}
}

func TestMissionPlanningBuildsOrderedMultiTargetSequence(t *testing.T) {
	var saved []TargetSite
	targets := []TargetSite{{ID: "m31", Name: "Andromeda Galaxy", Kind: "galaxy"}, {ID: "m42", Name: "Orion Nebula", Kind: "nebula"}}
	model := launchModel(Options{
		HomeBaseName: "Home", HomeBaseConfigured: true, TargetsEnabled: true, Targets: targets,
		CreateMission:      func(Origin, string, *time.Time, *time.Time) (string, error) { return "mission-1", nil },
		SaveMissionTargets: func(_ string, selected []TargetSite) error { saved = selected; return nil },
	})
	for range 2 {
		updated, _ := model.Update(enterKey())
		model = updated.(Model)
	}
	for range 4 {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey(' '))
	model = updated.(Model)
	updated, _ = model.Update(downKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey(' '))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.route != RouteMissionPlanning || len(model.missionPlan.targets) != 2 || model.missionPlan.targets[1].Name != "Orion Nebula" {
		t.Fatalf("multi-target selection did not return in order: route=%s targets=%+v", model.route, model.missionPlan.targets)
	}
	for range 4 {
		updated, _ = model.Update(runeKey('k'))
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	model = skipBlastOff(model)
	if model.route != RouteDeepSpace || len(saved) != 2 || saved[0].ID != "m31" || saved[1].ID != "m42" {
		t.Fatalf("multi-target mission was not persisted: route=%s saved=%+v", model.route, saved)
	}
}

func TestMissionPlanningForecastBrowserSelectsHour(t *testing.T) {
	temperature := 21.5
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, ForecastTimezone: "UTC", ForecastPoints: func(Origin) []ForecastPoint {
		return []ForecastPoint{{At: time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC), TemperatureC: &temperature}, {At: time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC), TemperatureC: &temperature}}
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteForecastBrowser {
		t.Fatalf("forecast action did not open browser: %s", model.route)
	}
	updated, _ = model.Update(downKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || !strings.Contains(model.View(), "21:00 UTC 21.5°C") || !strings.Contains(model.View(), "Forecast hour selected") {
		t.Fatalf("forecast selection did not return to planning: route=%s view=%s", model.route, model.View())
	}
}

func TestForecastBrowserFiltersByDarknessAndCloudCover(t *testing.T) {
	temperature := 20.0
	cloudClear, cloudHeavy := 10.0, 80.0
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, ForecastTimezone: "UTC", ForecastPoints: func(Origin) []ForecastPoint {
		return []ForecastPoint{{At: time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC), TemperatureC: &temperature, CloudCoverPercent: &cloudHeavy, Dark: false}, {At: time.Date(2026, 7, 22, 22, 0, 0, 0, time.UTC), TemperatureC: &temperature, CloudCoverPercent: &cloudClear, Dark: true}}
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey('d'))
	model = updated.(Model)
	if strings.Contains(model.View(), "18:00 UTC") || !strings.Contains(model.View(), "FILTERS  DARKNESS") {
		t.Fatalf("darkness filter did not narrow forecast: %s", model.View())
	}
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if !strings.Contains(model.View(), "CLOUD ≤50%") {
		t.Fatal("cloud filter was not shown")
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || !strings.Contains(model.View(), "22:00 UTC 20.0°C") {
		t.Fatalf("filtered hour was not selected: route=%s view=%s", model.route, model.View())
	}
}

func TestForecastBrowserUsesConfiguredThresholds(t *testing.T) {
	temperature := 20.0
	cloudAcceptable, cloudTooHigh := 40.0, 60.0
	precipitation := 10.0
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, ForecastTimezone: "UTC", ForecastCloudMax: 40, ForecastPrecipMax: 10, ForecastThresholdsConfigured: true, ForecastPoints: func(Origin) []ForecastPoint {
		return []ForecastPoint{
			{At: time.Date(2026, 7, 22, 22, 0, 0, 0, time.UTC), TemperatureC: &temperature, CloudCoverPercent: &cloudAcceptable, PrecipitationProbability: &precipitation, Dark: true},
			{At: time.Date(2026, 7, 22, 23, 0, 0, 0, time.UTC), TemperatureC: &temperature, CloudCoverPercent: &cloudTooHigh, PrecipitationProbability: &precipitation, Dark: true},
		}
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if !strings.Contains(model.View(), "CLOUD ≤40%") || strings.Contains(model.View(), "23:00 UTC") {
		t.Fatalf("configured cloud threshold was not applied: %s", model.View())
	}
}

func TestMissionPlanningRouteConsoleIsFunctional(t *testing.T) {
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, RouteEnabled: func(Origin) bool { return true }, RouteSummary: func(Origin) string {
		return "FROM Home\nTO Launch Site\nSTRAIGHT-LINE  12.0 km / 7.5 mi\nTRAVEL TIME    unavailable without a routing provider"
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteRoutePlan || !strings.Contains(model.View(), "STRAIGHT-LINE  12.0 km") {
		t.Fatalf("route plan did not open: route=%s view=%s", model.route, model.View())
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || !strings.Contains(model.View(), "Route plan reviewed") {
		t.Fatalf("route plan did not return to planning: route=%s view=%s", model.route, model.View())
	}
}

func TestSettingsCreatesEquipmentProfile(t *testing.T) {
	saved := false
	model := launchModel(Options{SaveEquipment: func(profile EquipmentProfile) error {
		saved = profile.Name == "Visual Rig" && profile.Description == "80mm refractor"
		return nil
	}})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('e'))
	model = updated.(Model)
	model = typeText(model, "Visual Rig")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "80mm refractor")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !saved || model.route != RouteSettings || len(model.options.EquipmentProfiles) != 1 {
		t.Fatalf("equipment profile setup failed: saved=%v route=%s profiles=%+v", saved, model.route, model.options.EquipmentProfiles)
	}
}

func TestSettingsImportsAtlasCatalog(t *testing.T) {
	importedPath, importedVersion := "", ""
	model := launchModel(Options{AtlasEnabled: true, AtlasStatus: "READY", SaveAtlas: func(path, version string) ([]AtlasSite, error) {
		importedPath, importedVersion = path, version
		return []AtlasSite{{ID: "site-1", Name: "Imported Site"}}, nil
	}})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('i'))
	model = updated.(Model)
	if model.route != RouteAtlasImport {
		t.Fatalf("Atlas import console did not open: %s", model.route)
	}
	model = typeText(model, "/tmp/catalog.csv")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "community-v2")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSettings || importedPath != "/tmp/catalog.csv" || importedVersion != "community-v2" || model.options.AtlasStatus != "READY" || len(model.options.AtlasLocations) != 1 {
		t.Fatalf("Atlas import flow failed: route=%s path=%q version=%q status=%q locations=%+v", model.route, importedPath, importedVersion, model.options.AtlasStatus, model.options.AtlasLocations)
	}
}

func TestPluginRegistryShowsValidatedMetadataWithoutExecution(t *testing.T) {
	model := launchModel(Options{PluginsStatus: "READY", PluginsDir: "/tmp/nightops/plugins", Plugins: []PluginManifest{{ID: "weather-pack", Name: "Weather Pack", Version: "1.2.0", Entrypoint: "bin/weather", Description: "Forecast tools", Capabilities: []string{"weather"}}}})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('p'))
	model = updated.(Model)
	if model.route != RoutePlugins || !strings.Contains(model.View(), "Plugin code is not executed") || !strings.Contains(model.View(), "weather-pack") {
		t.Fatalf("plugin registry did not show safe metadata: route=%s view=%s", model.route, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(Model).route != RouteSettings {
		t.Fatal("plugin registry escape did not return to settings")
	}
}

func TestMissionPlanningSelectsEquipmentProfile(t *testing.T) {
	createdID := ""
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, EquipmentProfiles: []EquipmentProfile{{ID: "rig-1", Name: "Visual Rig"}}, CreateMission: func(_ Origin, equipmentID string, _, _ *time.Time) (string, error) {
		createdID = equipmentID
		return "mission-1", nil
	}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteEquipmentBrowser {
		t.Fatalf("equipment action did not open browser: %s", model.route)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.equipmentID != "rig-1" {
		t.Fatalf("equipment selection did not return to planning: route=%s equipment=%+v", model.route, model.missionPlan)
	}
	for range 4 {
		updated, _ = model.Update(runeKey('k'))
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = skipBlastOff(updated.(Model))
	if model.route != RouteDeepSpace || createdID != "rig-1" {
		t.Fatalf("equipment selection was not persisted on mission creation: route=%s id=%q", model.route, createdID)
	}
}

func TestEquipmentInventoryAndReadinessFlow(t *testing.T) {
	saved := false
	model := launchModel(Options{EquipmentProfiles: []EquipmentProfile{{ID: "rig-1", Name: "Visual Rig"}}, SaveEquipmentItem: func(item EquipmentItem) error {
		saved = item.ProfileID == "rig-1" && item.Name == "Eyepiece case" && item.Category == "Optics" && item.Required
		return nil
	}})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('v'))
	model = updated.(Model)
	if model.route != RouteEquipmentInventory {
		t.Fatalf("inventory did not open: %s", model.route)
	}
	updated, _ = model.Update(runeKey('n'))
	model = updated.(Model)
	model = typeText(model, "Eyepiece case")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	model = typeText(model, "Optics")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !saved || model.route != RouteEquipmentInventory || len(model.options.EquipmentItems) != 1 {
		t.Fatalf("inventory item was not saved: saved=%v route=%s items=%+v", saved, model.route, model.options.EquipmentItems)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteEquipmentReadiness || !strings.Contains(model.View(), "READY") {
		t.Fatalf("readiness console did not confirm required inventory: route=%s view=%s", model.route, model.View())
	}
}

func TestMissionPlanningOpensEquipmentReadiness(t *testing.T) {
	model := launchModel(Options{HomeBaseName: "Home", HomeBaseConfigured: true, EquipmentProfiles: []EquipmentProfile{{ID: "rig-1", Name: "Visual Rig"}}, EquipmentItems: []EquipmentItem{{ID: "item-1", ProfileID: "rig-1", Name: "Eyepiece", Category: "Optics", Required: true}}})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	for range 4 {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(downKey())
	model = updated.(Model)
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteEquipmentReadiness || !strings.Contains(model.View(), "READY") {
		t.Fatalf("mission planning readiness action failed: route=%s view=%s", model.route, model.View())
	}
}

func TestAtlasOptionIsFeatureFlagged(t *testing.T) {
	if len(launchModel().actions()) != 4 {
		t.Fatal("disabled Atlas should be hidden")
	}
	if len(launchModel(Options{AtlasEnabled: true, AtlasStatus: "READY", AtlasLocations: []AtlasSite{{ID: "site-1", Name: "Dark Site", Latitude: 30, Longitude: -97, Timezone: "UTC"}}}).actions()) != 5 {
		t.Fatal("enabled Atlas should be visible")
	}
}

func TestAtlasBrowserSelectsMissionOrigin(t *testing.T) {
	model := launchModel(Options{AtlasEnabled: true, AtlasStatus: "READY", AtlasLocations: []AtlasSite{{ID: "site-1", Name: "Dark Site", Latitude: 30, Longitude: -97, Timezone: "UTC", Bortle: "4.5"}}})
	model.previousRoute, model.route = RouteLaunch, RouteAtlasBrowser
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.origin.Label != "Dark Site" {
		t.Fatalf("Atlas selection did not enter planning: route=%s origin=%+v", model.route, model.missionPlan.origin)
	}
}

func TestAtlasExportFlowPreservesSettingsRoute(t *testing.T) {
	exportedPath := ""
	model := launchModel(Options{
		AtlasEnabled:           true,
		AtlasStatus:            "READY",
		AtlasExportDefaultPath: "/tmp/nightops-atlas.csv",
		ExportAtlas: func(path string) error {
			exportedPath = path
			return nil
		},
	})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('x'))
	model = updated.(Model)
	if model.route != RouteAtlasExport {
		t.Fatalf("Atlas export did not open: %s", model.route)
	}
	model.atlasExport.path.SetValue("")
	model = typeText(model, "/tmp/community-atlas.csv")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSettings || exportedPath != "/tmp/community-atlas.csv" || !strings.Contains(model.settings.notice, "Atlas contribution CSV exported") {
		t.Fatalf("Atlas export flow failed: route=%s path=%q notice=%q", model.route, exportedPath, model.settings.notice)
	}
}

func TestAtlasExportRejectsEmptyPath(t *testing.T) {
	model := launchModel(Options{AtlasEnabled: true, AtlasStatus: "READY", ExportAtlas: func(string) error { return nil }})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('x'))
	model = updated.(Model)
	model.atlasExport.path.SetValue("")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteAtlasExport || model.atlasExport.error == "" {
		t.Fatalf("empty Atlas export path did not stay in form: route=%s error=%q", model.route, model.atlasExport.error)
	}
}

func TestCommandPaletteFiltersAndExecutesHelp(t *testing.T) {
	model := launchModel(Options{CommandPaletteEnabled: true})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updated.(Model)
	if model.route != RouteCommandPalette {
		t.Fatalf("Ctrl+K did not open command palette: %s", model.route)
	}
	model = typeText(model, "help")
	if !strings.Contains(model.View(), "Help") || strings.Contains(model.View(), "SkyBase Atlas") {
		t.Fatalf("palette filter did not narrow commands: %s", model.View())
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteHelp {
		t.Fatalf("filtered Help command did not open Help: %s", model.route)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(Model).route != RouteLaunch {
		t.Fatalf("Help did not return to the originating route: %s", updated.(Model).route)
	}
}

func TestCommandPaletteHidesUnavailableCommands(t *testing.T) {
	model := launchModel(Options{CommandPaletteEnabled: true, AtlasEnabled: false, PluginsStatus: "DISABLED"})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updated.(Model)
	view := model.View()
	if strings.Contains(view, "SkyBase Atlas") || strings.Contains(view, "Plugin Registry") || strings.Contains(view, "Equipment Browser") {
		t.Fatalf("palette exposed unavailable capability: %s", view)
	}
	if !strings.Contains(view, "Launch Console") || !strings.Contains(view, "Settings") {
		t.Fatalf("palette omitted core commands: %s", view)
	}
}

func TestCommandPaletteEscapePreservesRoute(t *testing.T) {
	model := launchModel(Options{CommandPaletteEnabled: true})
	updated, _ := model.Update(runeKey('?'))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	if model.route != RouteLaunch {
		t.Fatalf("help setup failed: %s", model.route)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(Model).route != RouteLaunch {
		t.Fatalf("palette escape did not preserve route: %s", updated.(Model).route)
	}
}

func TestConfiguredTelescopeSlewActionUsesSelectedTarget(t *testing.T) {
	slewed := false
	target := TargetSite{ID: "m42", Name: "M 42", Kind: "nebula", RightAscension: 5.59, Declination: -5.45}
	model := launchModel(Options{
		HomeBaseName:       "Home",
		HomeBaseConfigured: true,
		TargetsEnabled:     true,
		Targets:            []TargetSite{target},
		SlewToTarget: func(selected TargetSite) error {
			slewed = selected.ID == target.ID && selected.RightAscension == target.RightAscension
			return nil
		},
		TelescopeName: "Dwarf Test",
	})
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteTargetBrowser {
		t.Fatalf("launch did not open target selection: %s", model.route)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.route != RouteMissionPlanning {
		t.Fatalf("mission planning did not open: %s", model.route)
	}
	for range model.targetActionIndex() {
		updated, _ = model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	updated, _ = model.Update(runeKey(' '))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('c'))
	model = updated.(Model)
	if model.route != RouteMissionPlanning || model.missionPlan.target != "M 42" {
		t.Fatalf("target selection did not return to planning: route=%s target=%q", model.route, model.missionPlan.target)
	}
	for range model.telescopeActionIndex() {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if !slewed || !strings.Contains(model.missionPlan.status, "Dwarf Test") || model.route != RouteMissionPlanning {
		t.Fatalf("telescope action failed: slewed=%v route=%s status=%q", slewed, model.route, model.missionPlan.status)
	}
}

func TestTelescopeActionIsHiddenWithoutConfiguredAdapter(t *testing.T) {
	target := TargetSite{ID: "m42", Name: "M 42", Kind: "nebula", RightAscension: 5.59, Declination: -5.45}
	model := launchModel(Options{TargetsEnabled: true, Targets: []TargetSite{target}})
	model.missionPlan.target = target.Name
	if model.telescopeActionIndex() != -1 || strings.Contains(model.View(), "SLEW TELESCOPE") {
		t.Fatalf("telescope action was visible without an adapter: %v", model.missionActions())
	}
}

func TestSettingsBackupFlowCreatesRequestedFile(t *testing.T) {
	createdPath := ""
	model := launchModel(Options{
		BackupStatus:      "READY",
		BackupDefaultPath: "/tmp/nightops-default.db",
		CreateBackup: func(path string) error {
			createdPath = path
			return nil
		},
	})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	if model.route != RouteSettings {
		t.Fatalf("settings did not open: %s", model.route)
	}
	updated, _ = model.Update(runeKey('b'))
	model = updated.(Model)
	if model.route != RouteBackup {
		t.Fatalf("backup console did not open: %s", model.route)
	}
	model.backup.path.SetValue("")
	model = typeText(model, "/tmp/nightops-test.db")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSettings || createdPath != "/tmp/nightops-test.db" || !strings.Contains(model.settings.notice, "backup created") {
		t.Fatalf("backup flow failed: route=%s path=%q notice=%q", model.route, createdPath, model.settings.notice)
	}
}

func TestBackupRejectsEmptyPathWithoutLeavingForm(t *testing.T) {
	model := launchModel(Options{CreateBackup: func(string) error { return nil }})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('b'))
	model = updated.(Model)
	model.backup.path.SetValue("")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteBackup || model.backup.error == "" {
		t.Fatalf("empty backup path did not stay in form: route=%s error=%q", model.route, model.backup.error)
	}
}

func TestMissionBriefUsesConfiguredProviderAndReturnsAsynchronously(t *testing.T) {
	called := false
	model := launchModel(Options{
		HomeBaseName:       "Clarksville",
		HomeBaseConfigured: true,
		AIStatus:           "STANDBY",
		AIModel:            "llama3",
		GenerateMissionBrief: func(input MissionBriefInput) (string, error) {
			called = input.Origin == "Home Base · Clarksville"
			return "Confirmed origin; target visibility unknown.", nil
		},
	})
	for range 2 {
		updated, _ := model.Update(enterKey())
		model = updated.(Model)
	}
	if model.route != RouteMissionPlanning {
		t.Fatalf("planning did not open: %s", model.route)
	}
	for range model.aiBriefActionIndex() {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, command := model.Update(enterKey())
	model = updated.(Model)
	if command == nil || !model.missionPlan.briefPending {
		t.Fatalf("brief request did not become pending: command=%v pending=%v", command != nil, model.missionPlan.briefPending)
	}
	result := command()
	updated, _ = model.Update(result)
	model = updated.(Model)
	if !called || model.route != RouteMissionPlanning || model.missionPlan.brief == "" || model.missionPlan.briefPending {
		t.Fatalf("brief result was not retained: called=%v route=%s brief=%q pending=%v", called, model.route, model.missionPlan.brief, model.missionPlan.briefPending)
	}
}

func TestMissionBriefProviderFailureBecomesErrorRoute(t *testing.T) {
	model := launchModel(Options{HomeBaseConfigured: true, GenerateMissionBrief: func(MissionBriefInput) (string, error) { return "", fmt.Errorf("local model unavailable") }})
	for range 2 {
		updated, _ := model.Update(enterKey())
		model = updated.(Model)
	}
	for range model.aiBriefActionIndex() {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, command := model.Update(enterKey())
	model = updated.(Model)
	if command == nil {
		t.Fatal("AI error path did not return command")
	}
	updated, _ = model.Update(command())
	model = updated.(Model)
	if model.route != RouteError || !strings.Contains(model.error.detail, "local model unavailable") {
		t.Fatalf("AI failure did not become error route: route=%s error=%+v", model.route, model.error)
	}
}

func TestMissionArchiveLoadsSelectsAndReturnsToLaunch(t *testing.T) {
	planned := time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC)
	model := launchModel(Options{LoadMissions: func() ([]MissionSummary, error) {
		return []MissionSummary{{ID: "mission-1", Name: "Dark Site Run", Status: "completed", LaunchSiteName: "Dark Site", PlannedStart: &planned, CreatedAt: planned}}, nil
	}})
	for range 3 {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionArchive || len(model.missionArchive.missions) != 1 {
		t.Fatalf("archive did not load: route=%s missions=%d", model.route, len(model.missionArchive.missions))
	}
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionDetail || !strings.Contains(model.View(), "Dark Site Run") {
		t.Fatalf("archive detail did not open: route=%s view=%s", model.route, model.View())
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if updated.(Model).route != RouteLaunch {
		t.Fatalf("archive did not return to launch: %s", updated.(Model).route)
	}
}

func TestMissionArchiveEmptyStateIsNotDead(t *testing.T) {
	model := launchModel(Options{LoadMissions: func() ([]MissionSummary, error) { return nil, nil }})
	for range 3 {
		updated, _ := model.Update(downKey())
		model = updated.(Model)
	}
	updated, _ := model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteMissionArchive || !strings.Contains(model.View(), "No missions have been recorded yet") {
		t.Fatalf("archive empty state missing: route=%s view=%s", model.route, model.View())
	}
	updated, _ = model.Update(enterKey())
	if updated.(Model).route != RouteMissionArchive {
		t.Fatal("Enter on empty archive created a dead transition")
	}
}

func TestSyncExportAndImportSettingsFlows(t *testing.T) {
	exported := ""
	imported := ""
	model := launchModel(Options{
		SyncExportDefaultPath: "/tmp/nightops-sync.json",
		ExportSync:            func(path string) error { exported = path; return nil },
		ImportSync:            func(path string) (string, error) { imported = path; return "2 added · 1 updated · 3 skipped", nil },
	})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('y'))
	model = updated.(Model)
	if model.route != RouteSyncExport {
		t.Fatalf("sync export did not open: %s", model.route)
	}
	model.syncExport.path.SetValue("")
	model = typeText(model, "/tmp/export.json")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSettings || exported != "/tmp/export.json" {
		t.Fatalf("sync export failed: route=%s path=%q", model.route, exported)
	}
	updated, _ = model.Update(runeKey('u'))
	model = updated.(Model)
	model = typeText(model, "/tmp/import.json")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSettings || imported != "/tmp/import.json" || !strings.Contains(model.settings.notice, "2 added") {
		t.Fatalf("sync import failed: route=%s path=%q notice=%q", model.route, imported, model.settings.notice)
	}
}

func TestSyncImportFailureStaysInForm(t *testing.T) {
	model := launchModel(Options{ImportSync: func(string) (string, error) { return "", fmt.Errorf("invalid bundle") }})
	updated, _ := model.Update(runeKey('s'))
	model = updated.(Model)
	updated, _ = model.Update(runeKey('u'))
	model = updated.(Model)
	model = typeText(model, "/tmp/bad.json")
	updated, _ = model.Update(enterKey())
	model = updated.(Model)
	if model.route != RouteSyncImport || !strings.Contains(model.syncImport.error, "invalid bundle") {
		t.Fatalf("sync import failure left form: route=%s error=%q", model.route, model.syncImport.error)
	}
}

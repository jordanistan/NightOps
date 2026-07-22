// Package console implements the Bubble Tea application shell.
package console

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Route identifies the active application console.
type Route int

const (
	RouteBoot Route = iota
	RouteLaunch
	RouteHomeBaseSetup
	RouteGPSAcquisition
	RouteZIPEntry
	RouteAtlasBrowser
	RouteAtlasImport
	RouteAtlasExport
	RouteTargetBrowser
	RouteForecastBrowser
	RouteRoutePlan
	RouteEquipmentSetup
	RouteEquipmentBrowser
	RouteEquipmentInventory
	RouteEquipmentItemSetup
	RouteEquipmentReadiness
	RouteMissionSchedule
	RouteBackup
	RoutePlugins
	RouteCommandPalette
	RouteMissionArchive
	RouteMissionDetail
	RouteSyncExport
	RouteSyncImport
	RouteMissionPlanning
	RouteMissionReview
	RouteBlastOff
	RouteDeepSpace
	RouteOperation
	RouteObservationEntry
	RouteDebrief
	RouteSettings
	RouteHelp
	RouteError
)

func (r Route) String() string {
	switch r {
	case RouteBoot:
		return "Boot"
	case RouteLaunch:
		return "Launch"
	case RouteHomeBaseSetup:
		return "Home Base Setup"
	case RouteGPSAcquisition:
		return "GPS Acquisition"
	case RouteZIPEntry:
		return "ZIP Entry"
	case RouteAtlasBrowser:
		return "Atlas Browser"
	case RouteAtlasImport:
		return "Atlas Import"
	case RouteAtlasExport:
		return "Atlas Export"
	case RouteTargetBrowser:
		return "Target Browser"
	case RouteForecastBrowser:
		return "Forecast Browser"
	case RouteRoutePlan:
		return "Route Plan"
	case RouteEquipmentSetup:
		return "Equipment Setup"
	case RouteEquipmentBrowser:
		return "Equipment Browser"
	case RouteEquipmentInventory:
		return "Equipment Inventory"
	case RouteEquipmentItemSetup:
		return "Equipment Item Setup"
	case RouteEquipmentReadiness:
		return "Equipment Readiness"
	case RouteMissionSchedule:
		return "Mission Schedule"
	case RouteBackup:
		return "Database Backup"
	case RoutePlugins:
		return "Plugins"
	case RouteCommandPalette:
		return "Command Palette"
	case RouteMissionArchive:
		return "Mission Archive"
	case RouteMissionDetail:
		return "Mission Detail"
	case RouteSyncExport:
		return "Sync Export"
	case RouteSyncImport:
		return "Sync Import"
	case RouteMissionPlanning:
		return "Mission Planning"
	case RouteMissionReview:
		return "Mission Review"
	case RouteBlastOff:
		return "Blast Off"
	case RouteDeepSpace:
		return "Deep Space"
	case RouteOperation:
		return "Operation"
	case RouteObservationEntry:
		return "Observation Entry"
	case RouteDebrief:
		return "Debrief"
	case RouteSettings:
		return "Settings"
	case RouteHelp:
		return "Help"
	case RouteError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Origin is the selected mission location. Coordinates are optional until a real provider supplies them.
type Origin struct {
	Kind      string
	Label     string
	ZIP       string
	Timezone  string
	Latitude  *float64
	Longitude *float64
}

// Options supplies initialized capabilities and profile data to the UI.
type Options struct {
	ThemeName                    string
	HomeBaseName                 string
	HomeBaseLocation             string
	HomeBaseZIP                  string
	HomeBaseTimezone             string
	HomeBaseLatitude             *float64
	HomeBaseLongitude            *float64
	HomeBaseConfigured           bool
	DatabaseStatus               string
	ObsidianStatus               string
	OpenObsidianVault            func() error
	AtlasStatus                  string
	GPSStatus                    string
	GeocodingStatus              string
	WeatherStatus                string
	TelescopeStatus              string
	TelescopeName                string
	BackupStatus                 string
	BackupDefaultPath            string
	AIStatus                     string
	AIModel                      string
	APIStatus                    string
	AtlasEnabled                 bool
	AtlasLocations               []AtlasSite
	Version                      string
	LocationProvider             LocationProvider
	ResolveZIP                   func(string) (Origin, error)
	SaveHomeBase                 func(HomeBase) error
	CreateMission                func(Origin, string, *time.Time, *time.Time) (string, error)
	SaveMissionTargets           func(string, []TargetSite) error
	ScheduleTimezone             string
	PluginsStatus                string
	PluginsDir                   string
	Plugins                      []PluginManifest
	CommandPaletteEnabled        bool
	StartMission                 func(string) error
	ActivateMission              func(string) error
	RecordObservation            func(string, string, string) error
	RecordDebrief                func(string, string) error
	CompleteMission              func(string) error
	AstronomySummary             func(Origin) string
	WeatherSummary               func(Origin) string
	WeatherAssessment            func(Origin, []TargetSite) string
	ForecastSummary              func(Origin) string
	ForecastPoints               func(Origin) []ForecastPoint
	MissionWindow                func(Origin, []TargetSite) (*time.Time, *time.Time, string)
	ForecastTimezone             string
	ForecastCloudMax             int
	ForecastPrecipMax            int
	ForecastThresholdsConfigured bool
	RouteEnabled                 func(Origin) bool
	RouteSummary                 func(Origin) string
	EquipmentProfiles            []EquipmentProfile
	EquipmentItems               []EquipmentItem
	SaveEquipment                func(EquipmentProfile) error
	SaveEquipmentItem            func(EquipmentItem) error
	SaveAtlas                    func(path, version string) ([]AtlasSite, error)
	ExportAtlas                  func(path string) error
	AtlasExportDefaultPath       string
	TargetSummary                func(Origin, TargetSite) string
	TargetForecastSummary        func(Origin, TargetSite, []ForecastPoint) string
	SlewToTarget                 func(TargetSite) error
	SlewToTargetAt               func(Origin, TargetSite) error
	CreateBackup                 func(string) error
	GenerateMissionBrief         func(MissionBriefInput) (string, error)
	LoadMissions                 func() ([]MissionSummary, error)
	ExportSync                   func(string) error
	ImportSync                   func(string) (string, error)
	SyncExportDefaultPath        string
	TargetsEnabled               bool
	Targets                      []TargetSite
}

// LocationProvider acquires a real location from a platform adapter.
type LocationProvider interface {
	Acquire(context.Context) (Origin, error)
}

// MissionBriefInput is the factual planning context supplied to an AI provider.
type MissionBriefInput struct {
	Origin       string
	Target       string
	TargetWindow string
	Conditions   string
	Forecast     string
	Astronomy    string
	Route        string
	Equipment    string
}

// MissionSummary is the read-only projection displayed by the offline archive.
type MissionSummary struct {
	ID                 string
	Name               string
	Status             string
	LaunchSiteName     string
	Timezone           string
	EquipmentProfileID string
	PlannedStart       *time.Time
	PlannedEnd         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// HomeBase is the supported local profile form.
type HomeBase struct {
	Name string
	ZIP  string
}

type bootTickMsg struct{}
type locationResultMsg struct {
	origin Origin
	err    error
}
type zipResolutionMsg struct {
	origin Origin
	err    error
}
type gpsUnavailableMsg struct{}
type missionBriefResultMsg struct {
	brief string
	err   error
}
type blastOffTickMsg struct{}

type launchAction struct {
	key         string
	label       string
	description string
}

type LaunchModel struct{ selected int }
type launchSelection struct {
	selectedOrigin string
	notice         string
	interacted     bool
}
type HomeBaseSetupModel struct {
	name  textinput.Model
	zip   textinput.Model
	focus int
	error string
}
type AtlasImportModel struct {
	path    textinput.Model
	version textinput.Model
	focus   int
	error   string
}
type AtlasExportModel struct {
	path  textinput.Model
	error string
}
type GPSModel struct{ status string }
type ZIPEntryModel struct {
	input   textinput.Model
	error   string
	pending bool
}
type MissionScheduleModel struct {
	start textinput.Model
	end   textinput.Model
	focus int
	error string
}
type BackupModel struct {
	path  textinput.Model
	error string
}
type SyncExportModel struct {
	path  textinput.Model
	error string
}
type SyncImportModel struct {
	path  textinput.Model
	error string
}
type MissionPlanModel struct {
	origin           Origin
	plannedStart     *time.Time
	plannedEnd       *time.Time
	status           string
	astronomy        string
	weather          string
	weatherDecision  string
	forecast         string
	forecastPoints   []ForecastPoint
	forecastSelected int
	darkOnly         bool
	cloudOnly        bool
	precipOnly       bool
	route            string
	target           string
	targets          []TargetSite
	targetInfo       string
	targetForecast   string
	missionWindow    string
	equipmentID      string
	equipment        string
	brief            string
	briefPending     bool
	selected         int
}
type OperationModel struct {
	missionID    string
	status       string
	observations []string
}
type missionArchiveModel struct {
	selected int
	missions []MissionSummary
	error    string
}

// AtlasSite is the UI projection of a catalog launch site.
type AtlasSite struct {
	ID        string
	Name      string
	Latitude  float64
	Longitude float64
	Timezone  string
	Bortle    string
}

// TargetSite is the UI projection of an offline celestial target.
type TargetSite struct {
	ID             string
	Name           string
	Kind           string
	RightAscension float64
	Declination    float64
	Source         string
}

// EquipmentProfile is the UI projection of a reusable observing setup.
type EquipmentProfile struct {
	ID          string
	Name        string
	Description string
}

// EquipmentItem is the UI projection of one inventory item.
type EquipmentItem struct {
	ID        string
	ProfileID string
	Name      string
	Category  string
	Required  bool
}

// PluginManifest is the UI projection of validated local plugin metadata.
type PluginManifest struct {
	ID           string
	Name         string
	Version      string
	Description  string
	Entrypoint   string
	Capabilities []string
}

// PaletteCommand is one locally executable command exposed by the command palette.
type PaletteCommand struct {
	Key         string
	Label       string
	Description string
}

type commandPaletteModel struct {
	input    textinput.Model
	selected int
	error    string
}

type EquipmentItemSetupModel struct {
	name     textinput.Model
	category textinput.Model
	focus    int
	required bool
	error    string
}

// ForecastPoint is the UI projection of one cached or provider-reported hour.
type ForecastPoint struct {
	At                       time.Time
	TemperatureC             *float64
	CloudCoverPercent        *float64
	PrecipitationProbability *float64
	Dark                     bool
}
type ObservationEntryModel struct {
	target textinput.Model
	notes  textinput.Model
	focus  int
	error  string
}
type DebriefModel struct {
	missionID string
	count     int
	summary   textinput.Model
	error     string
	saved     bool
}
type SettingsModel struct{ notice string }
type HelpModel struct{}
type ErrorModel struct {
	title   string
	detail  string
	context string
}

// Model is the root state machine and owns every child console model.
type Model struct {
	route          Route
	previousRoute  Route
	spinner        spinner.Model
	bootStage      int
	theme          Theme
	width          int
	height         int
	options        Options
	launch         LaunchModel
	launchState    launchSelection
	homeBase       HomeBaseSetupModel
	gps            GPSModel
	zipEntry       ZIPEntryModel
	schedule       MissionScheduleModel
	backup         BackupModel
	atlasImport    AtlasImportModel
	atlasExport    AtlasExportModel
	equipmentSetup struct {
		name        textinput.Model
		description textinput.Model
		focus       int
		error       string
	}
	equipmentItemSetup EquipmentItemSetupModel
	missionPlan        MissionPlanModel
	missionReview      struct{ selected int }
	blastOff           int
	deepSpaceSelected  int
	operation          OperationModel
	atlas              struct{ selected int }
	target             struct {
		selected int
		chosen   map[string]bool
	}
	equipment          struct{ selected int }
	equipmentInventory struct{ selected int }
	equipmentReadiness struct{ profileID string }
	plugins            struct{ selected int }
	palette            commandPaletteModel
	missionArchive     missionArchiveModel
	missionDetail      MissionSummary
	syncExport         SyncExportModel
	syncImport         SyncImportModel
	observation        ObservationEntryModel
	debrief            DebriefModel
	settings           SettingsModel
	help               HelpModel
	error              ErrorModel
	quitting           bool
}

var bootStages = []string{
	"INITIALIZING NIGHTOPS",
	"LOADING SKYBASE ATLAS",
	"CHECKING LOCAL DATABASE",
	"CALIBRATING CELESTIAL NAVIGATION",
	"MISSION CONTROL ONLINE",
}

var zipPattern = regexp.MustCompile(`^\d{5}(-\d{4})?$`)

// New creates the NightOps root model. Optional options preserve a useful default for embedders.
func New(theme Theme, provided ...Options) Model {
	spinnerModel := spinner.New()
	spinnerModel.Spinner = spinner.Line
	options := Options{
		HomeBaseName:     "Home Base",
		HomeBaseLocation: "Not configured",
		DatabaseStatus:   "READY",
		ObsidianStatus:   "READY",
		AtlasStatus:      "DISABLED",
		GPSStatus:        "UNAVAILABLE",
		WeatherStatus:    "DISABLED",
		TelescopeStatus:  "DISABLED",
		BackupStatus:     "DISABLED",
		AIStatus:         "DISABLED",
		APIStatus:        "DISABLED",
		Version:          "v0.1.0-dev",
	}
	if len(provided) > 0 {
		options = provided[0]
	}
	return Model{
		route:              RouteBoot,
		spinner:            spinnerModel,
		theme:              theme,
		options:            options,
		homeBase:           newHomeBaseSetup(),
		zipEntry:           newZIPEntry(),
		schedule:           newMissionSchedule(),
		backup:             newBackup(),
		syncExport:         newSyncExport(),
		syncImport:         newSyncImport(),
		atlasImport:        newAtlasImport(),
		atlasExport:        newAtlasExport(),
		palette:            newCommandPalette(),
		equipmentSetup:     newEquipmentSetup(),
		equipmentItemSetup: newEquipmentItemSetup(),
		observation:        newObservationEntry(),
		debrief:            newDebrief(),
		missionPlan:        MissionPlanModel{status: "Origin selected; mission is ready to configure."},
		target: struct {
			selected int
			chosen   map[string]bool
		}{chosen: make(map[string]bool)},
	}
}

func newCommandPalette() commandPaletteModel {
	input := textinput.New()
	input.Prompt = "  COMMAND   "
	input.CharLimit = 64
	input.Width = 48
	return commandPaletteModel{input: input}
}

func newObservationEntry() ObservationEntryModel {
	target := textinput.New()
	target.Prompt = "  TARGET    "
	target.CharLimit = 80
	target.Width = 36
	notes := textinput.New()
	notes.Prompt = "  NOTES     "
	notes.CharLimit = 240
	notes.Width = 48
	return ObservationEntryModel{target: target, notes: notes}
}

func newDebrief() DebriefModel {
	summary := textinput.New()
	summary.Prompt = "  SUMMARY   "
	summary.CharLimit = 500
	summary.Width = 72
	return DebriefModel{summary: summary}
}

func newEquipmentSetup() struct {
	name        textinput.Model
	description textinput.Model
	focus       int
	error       string
} {
	name := textinput.New()
	name.Prompt = "  NAME        "
	name.CharLimit = 64
	name.Width = 42
	description := textinput.New()
	description.Prompt = "  DESCRIPTION "
	description.CharLimit = 160
	description.Width = 60
	return struct {
		name        textinput.Model
		description textinput.Model
		focus       int
		error       string
	}{name: name, description: description}
}

func newEquipmentItemSetup() EquipmentItemSetupModel {
	name := textinput.New()
	name.Prompt = "  ITEM NAME "
	name.CharLimit = 64
	name.Width = 42
	category := textinput.New()
	category.Prompt = "  CATEGORY  "
	category.CharLimit = 48
	category.Width = 32
	return EquipmentItemSetupModel{name: name, category: category, required: true}
}

func newHomeBaseSetup() HomeBaseSetupModel {
	name := textinput.New()
	name.Prompt = "  LABEL     "
	name.CharLimit = 64
	name.Width = 32
	zip := textinput.New()
	zip.Prompt = "  ZIP CODE  "
	zip.CharLimit = 10
	zip.Width = 16
	return HomeBaseSetupModel{name: name, zip: zip}
}

func newZIPEntry() ZIPEntryModel {
	input := textinput.New()
	input.Prompt = "  ZIP CODE  "
	input.CharLimit = 10
	input.Width = 16
	return ZIPEntryModel{input: input}
}

func newMissionSchedule() MissionScheduleModel {
	start := textinput.New()
	start.Prompt = "  START      "
	start.CharLimit = 16
	start.Width = 24
	end := textinput.New()
	end.Prompt = "  END        "
	end.CharLimit = 16
	end.Width = 24
	return MissionScheduleModel{start: start, end: end}
}

func newBackup() BackupModel {
	path := textinput.New()
	path.Prompt = "  BACKUP PATH "
	path.CharLimit = 240
	path.Width = 72
	return BackupModel{path: path}
}

func newSyncExport() SyncExportModel {
	path := textinput.New()
	path.Prompt = "  BUNDLE PATH "
	path.CharLimit = 240
	path.Width = 72
	return SyncExportModel{path: path}
}

func newSyncImport() SyncImportModel {
	path := textinput.New()
	path.Prompt = "  BUNDLE PATH "
	path.CharLimit = 240
	path.Width = 72
	return SyncImportModel{path: path}
}

func newAtlasImport() AtlasImportModel {
	path := textinput.New()
	path.Prompt = "  CSV PATH  "
	path.CharLimit = 240
	path.Width = 64
	version := textinput.New()
	version.Prompt = "  VERSION   "
	version.CharLimit = 64
	version.Width = 32
	return AtlasImportModel{path: path, version: version}
}

func newAtlasExport() AtlasExportModel {
	path := textinput.New()
	path.Prompt = "  CSV PATH   "
	path.CharLimit = 240
	path.Width = 64
	return AtlasExportModel{path: path}
}

// Init starts the staged system initialization animation.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg { return bootTickMsg{} }))
}

// Update delegates each message to the child model belonging to the active route.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case spinner.TickMsg:
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(message)
		return m, command
	case bootTickMsg:
		if m.route != RouteBoot {
			return m, nil
		}
		m.bootStage++
		if m.bootStage >= len(bootStages) {
			m.route = RouteLaunch
			return m, nil
		}
		return m, tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg { return bootTickMsg{} })
	case locationResultMsg:
		if m.route != RouteGPSAcquisition {
			return m, nil
		}
		if message.err != nil {
			m.error = ErrorModel{title: "GPS UNAVAILABLE", detail: message.err.Error(), context: "GPS"}
			m.route = RouteError
			return m, nil
		}
		m.setMissionOrigin(message.origin)
		m.missionPlan.status = "Current location acquired. Select tonight's targets."
		return m, m.beginTargetSelection()
	case gpsUnavailableMsg:
		if m.route == RouteGPSAcquisition && m.options.LocationProvider == nil {
			m.error = ErrorModel{title: "GPS UNAVAILABLE", detail: "No GPS adapter is configured for this installation.", context: "GPS"}
			m.route = RouteError
		}
		return m, nil
	case zipResolutionMsg:
		if m.route != RouteZIPEntry || !m.zipEntry.pending {
			return m, nil
		}
		m.zipEntry.pending = false
		if message.err != nil {
			zip := strings.TrimSpace(m.zipEntry.input.Value())
			m.setMissionOrigin(Origin{Kind: "ZIP Code", Label: "ZIP " + zip, ZIP: zip, Timezone: m.options.ScheduleTimezone})
			m.missionPlan.status = "ZIP accepted. Geocoding is unavailable; coordinates remain unknown."
			return m, m.beginTargetSelection()
		}
		m.setMissionOrigin(message.origin)
		m.missionPlan.status = "ZIP origin resolved from the configured geocoding provider."
		return m, m.beginTargetSelection()
	case missionBriefResultMsg:
		if m.route != RouteMissionPlanning || !m.missionPlan.briefPending {
			return m, nil
		}
		m.missionPlan.briefPending = false
		if message.err != nil {
			m.error = ErrorModel{title: "AI BRIEF UNAVAILABLE", detail: message.err.Error(), context: "AI"}
			m.route = RouteError
			return m, nil
		}
		m.missionPlan.brief = message.brief
		m.missionPlan.status = "Mission brief generated from the configured local provider."
		return m, nil
	case blastOffTickMsg:
		if m.route != RouteBlastOff {
			return m, nil
		}
		if m.blastOff <= 0 {
			m.route = RouteDeepSpace
			return m, nil
		}
		m.blastOff--
		if m.blastOff == 0 {
			if m.operation.missionID == "" {
				m.missionPlan.status = "Launch data loaded. Select targets and configure the mission."
				m.route = RouteMissionPlanning
				return m, nil
			}
			if m.options.StartMission != nil {
				if err := m.options.StartMission(m.operation.missionID); err != nil {
					m.error = ErrorModel{title: "MISSION LAUNCH FAILED", detail: err.Error(), context: "MISSION"}
					m.route = RouteError
					return m, nil
				}
			}
			if m.options.ActivateMission != nil {
				if err := m.options.ActivateMission(m.operation.missionID); err != nil {
					m.error = ErrorModel{title: "OPERATION ACTIVATION FAILED", detail: err.Error(), context: "MISSION"}
					m.route = RouteError
					return m, nil
				}
			}
			m.operation.status = "ACTIVE"
			m.route = RouteDeepSpace
			return m, nil
		}
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg { return blastOffTickMsg{} })
	case tea.KeyMsg:
		return m, m.handleKey(message)
	}
	return m, nil
}

func (m *Model) handleKey(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if m.route != RouteHomeBaseSetup && m.route != RouteZIPEntry && m.route != RouteBackup && m.route != RouteAtlasExport && m.route != RouteSyncExport && m.route != RouteSyncImport && m.route != RouteCommandPalette && (key == "ctrl+c" || key == "q") {
		m.quitting = true
		return tea.Quit
	}
	if m.route == RouteBoot {
		if key == "enter" || key == "space" || key == " " {
			m.bootStage = len(bootStages)
			m.route = RouteLaunch
		}
		return nil
	}
	if key == "ctrl+k" && m.options.CommandPaletteEnabled && m.route != RouteCommandPalette {
		m.previousRoute = m.route
		m.route = RouteCommandPalette
		m.palette.input.SetValue("")
		m.palette.selected = 0
		m.palette.error = ""
		m.palette.input.Focus()
		return textinput.Blink
	}
	switch m.route {
	case RouteLaunch:
		return m.updateLaunch(key)
	case RouteHomeBaseSetup:
		return m.updateHomeBase(message)
	case RouteEquipmentSetup:
		return m.updateEquipmentSetup(message)
	case RouteGPSAcquisition:
		if key == "esc" {
			m.route = m.previousRoute
		}
	case RouteZIPEntry:
		return m.updateZIP(message)
	case RouteAtlasBrowser:
		return m.updateAtlas(key)
	case RouteAtlasImport:
		return m.updateAtlasImport(message)
	case RouteAtlasExport:
		return m.updateAtlasExport(message)
	case RouteTargetBrowser:
		return m.updateTarget(key)
	case RouteForecastBrowser:
		return m.updateForecast(key)
	case RouteRoutePlan:
		return m.updateRoutePlan(key)
	case RouteEquipmentBrowser:
		return m.updateEquipment(key)
	case RouteEquipmentInventory:
		return m.updateEquipmentInventory(key)
	case RouteEquipmentItemSetup:
		return m.updateEquipmentItemSetup(message)
	case RouteEquipmentReadiness:
		if key == "esc" {
			m.route = m.previousRoute
		}
	case RouteMissionSchedule:
		return m.updateMissionSchedule(message)
	case RouteBackup:
		return m.updateBackup(message)
	case RoutePlugins:
		return m.updatePlugins(key)
	case RouteCommandPalette:
		return m.updateCommandPalette(message)
	case RouteMissionArchive:
		return m.updateMissionArchive(key)
	case RouteMissionDetail:
		if key == "esc" || key == "b" {
			m.route = RouteMissionArchive
		}
		return nil
	case RouteSyncExport:
		return m.updateSyncExport(message)
	case RouteSyncImport:
		return m.updateSyncImport(message)
	case RouteMissionPlanning:
		return m.updateMissionPlan(key)
	case RouteMissionReview:
		return m.updateMissionReview(key)
	case RouteBlastOff:
		if key == "enter" && m.operation.missionID == "" {
			m.blastOff = 0
			m.missionPlan.status = "Launch data loaded. Select targets and configure the mission."
			m.route = RouteMissionPlanning
		} else if key == "esc" {
			m.route = RouteMissionPlanning
		}
	case RouteDeepSpace:
		return m.updateDeepSpace(key)
	case RouteOperation:
		return m.updateOperation(key)
	case RouteObservationEntry:
		return m.updateObservation(message)
	case RouteDebrief:
		return m.updateDebrief(message)
	case RouteSettings:
		if key == "esc" {
			m.route = m.previousRoute
		} else if key == "h" {
			m.previousRoute, m.route = RouteSettings, RouteHomeBaseSetup
			m.homeBase.focus = 0
			m.homeBase.name.Focus()
		} else if key == "e" {
			m.previousRoute, m.route = RouteSettings, RouteEquipmentSetup
			m.equipmentSetup.focus = 0
			m.equipmentSetup.name.Focus()
		} else if key == "v" && m.options.SaveEquipmentItem != nil {
			m.previousRoute, m.route = RouteSettings, RouteEquipmentInventory
			m.equipmentInventory.selected = 0
		} else if key == "i" && m.options.SaveAtlas != nil && m.options.AtlasEnabled {
			m.previousRoute, m.route = RouteSettings, RouteAtlasImport
			m.atlasImport.focus = 0
			m.atlasImport.error = ""
			m.atlasImport.path.Focus()
		} else if key == "x" && m.options.ExportAtlas != nil && m.options.AtlasEnabled && m.options.AtlasStatus == "READY" {
			m.previousRoute, m.route = RouteSettings, RouteAtlasExport
			m.atlasExport.path.SetValue(m.options.AtlasExportDefaultPath)
			m.atlasExport.error = ""
			m.atlasExport.path.Focus()
			return textinput.Blink
		} else if key == "p" && m.options.PluginsStatus != "DISABLED" {
			m.previousRoute, m.route = RouteSettings, RoutePlugins
			m.plugins.selected = 0
		} else if key == "b" && m.options.CreateBackup != nil {
			m.previousRoute, m.route = RouteSettings, RouteBackup
			m.backup.path.SetValue(m.options.BackupDefaultPath)
			m.backup.error = ""
			m.backup.path.Focus()
			return textinput.Blink
		} else if key == "y" && m.options.ExportSync != nil {
			m.previousRoute, m.route = RouteSettings, RouteSyncExport
			m.syncExport.path.SetValue(m.options.SyncExportDefaultPath)
			m.syncExport.error = ""
			m.syncExport.path.Focus()
			return textinput.Blink
		} else if key == "u" && m.options.ImportSync != nil {
			m.previousRoute, m.route = RouteSettings, RouteSyncImport
			m.syncImport.path.SetValue("")
			m.syncImport.error = ""
			m.syncImport.path.Focus()
			return textinput.Blink
		} else if key == "?" {
			m.previousRoute, m.route = RouteSettings, RouteHelp
		} else if key == "o" && m.options.OpenObsidianVault != nil {
			if err := m.options.OpenObsidianVault(); err != nil {
				m.settings.notice = "Obsidian vault could not be opened: " + err.Error()
			} else {
				m.settings.notice = "Obsidian vault opened."
			}
		}
	case RouteHelp:
		if key == "esc" || key == "?" {
			m.route = m.previousRoute
		}
	case RouteError:
		return m.updateError(key)
	}
	return nil
}

func (m *Model) updateLaunch(key string) tea.Cmd {
	actions := m.actions()
	switch key {
	case "up", "k":
		m.launchState.interacted = true
		if m.launch.selected > 0 {
			m.launch.selected--
		}
	case "down", "j":
		m.launchState.interacted = true
		if m.launch.selected < len(actions)-1 {
			m.launch.selected++
		}
	case "enter":
		if len(actions) > 0 {
			if !m.launchState.interacted && m.launch.selected == 0 {
				return m.startGuidedLaunch()
			}
			return m.selectLaunchAction(actions[m.launch.selected])
		}
	case "s":
		m.previousRoute, m.route = RouteLaunch, RouteSettings
	case "?":
		m.previousRoute, m.route = RouteLaunch, RouteHelp
	}
	return nil
}

func (m *Model) startGuidedLaunch() tea.Cmd {
	if !m.options.HomeBaseConfigured {
		return m.beginOrigin("h")
	}
	label := m.options.HomeBaseName
	if strings.TrimSpace(label) == "" || label == "ZIP "+m.options.HomeBaseZIP {
		label = "Home Base"
	}
	m.setMissionOrigin(Origin{Kind: "Home Base", Label: label, ZIP: m.options.HomeBaseZIP, Timezone: m.options.HomeBaseTimezone, Latitude: m.options.HomeBaseLatitude, Longitude: m.options.HomeBaseLongitude})
	if !m.options.TargetsEnabled || len(m.options.Targets) == 0 {
		m.missionPlan.status = "Launch data loaded. No catalog targets are available; continue with manual observations."
		m.blastOff = 10
		m.operation = OperationModel{}
		m.previousRoute, m.route = RouteLaunch, RouteBlastOff
		return tea.Tick(time.Second, func(time.Time) tea.Msg { return blastOffTickMsg{} })
	}
	return m.beginTargetSelection()
}

func (m *Model) selectLaunchAction(action launchAction) tea.Cmd {
	if action.key == "archive" {
		m.previousRoute, m.route = RouteLaunch, RouteMissionArchive
		return m.loadMissionArchive()
	}
	if action.key == "launch" {
		if !m.options.HomeBaseConfigured {
			return m.beginOrigin("h")
		}
		label := m.options.HomeBaseName
		if strings.TrimSpace(label) == "" || label == "ZIP "+m.options.HomeBaseZIP {
			label = "Home Base"
		}
		m.setMissionOrigin(Origin{Kind: "Home Base", Label: label, ZIP: m.options.HomeBaseZIP, Timezone: m.options.HomeBaseTimezone, Latitude: m.options.HomeBaseLatitude, Longitude: m.options.HomeBaseLongitude})
		if !m.options.TargetsEnabled || len(m.options.Targets) == 0 {
			m.missionPlan.status = "Launch data loaded. No catalog targets are available; continue with manual observations."
			m.blastOff = 10
			m.operation = OperationModel{}
			m.previousRoute, m.route = RouteLaunch, RouteBlastOff
			return tea.Tick(time.Second, func(time.Time) tea.Msg { return blastOffTickMsg{} })
		}
		return m.beginTargetSelection()
	}
	m.launchState.selectedOrigin = action.key
	m.launchState.notice = action.label + " selected. Press Enter on LAUNCH MISSION to continue."
	if len(m.actions()) > 0 {
		m.launch.selected = len(m.actions()) - 1
	}
	return nil
}

func (m *Model) beginTargetSelection() tea.Cmd {
	if !m.options.TargetsEnabled || len(m.options.Targets) == 0 {
		m.missionPlan.status = "Target catalog unavailable; mission can continue with manual observations."
		m.route = RouteMissionPlanning
		return nil
	}
	m.target.selected = 0
	m.target.chosen = make(map[string]bool)
	m.previousRoute, m.route = RouteMissionPlanning, RouteTargetBrowser
	m.missionPlan.status = "Select tonight's targets. NightOps will calculate the window and weather after confirmation."
	return nil
}

func (m *Model) beginOrigin(originKey string) tea.Cmd {
	switch originKey {
	case "h":
		if m.options.HomeBaseConfigured {
			label := m.options.HomeBaseName
			if strings.TrimSpace(label) == "" || label == "ZIP "+m.options.HomeBaseZIP {
				label = "Home Base"
			}
			m.setMissionOrigin(Origin{Kind: "Home Base", Label: label, ZIP: m.options.HomeBaseZIP, Timezone: m.options.HomeBaseTimezone, Latitude: m.options.HomeBaseLatitude, Longitude: m.options.HomeBaseLongitude})
			m.missionPlan.status = "Home Base selected. Select tonight's targets."
			return m.beginTargetSelection()
		} else {
			m.previousRoute, m.route = RouteLaunch, RouteHomeBaseSetup
			m.homeBase.focus = 0
			m.homeBase.name.Focus()
		}
	case "g":
		m.previousRoute, m.route = RouteLaunch, RouteGPSAcquisition
		m.gps.status = "ACQUIRING CURRENT LOCATION"
		if m.options.LocationProvider == nil {
			m.gps.status = "UNAVAILABLE"
			return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return gpsUnavailableMsg{} })
		}
		return func() tea.Msg {
			origin, err := m.options.LocationProvider.Acquire(context.Background())
			return locationResultMsg{origin: origin, err: err}
		}
	case "z":
		m.previousRoute, m.route = RouteLaunch, RouteZIPEntry
		m.zipEntry.error = ""
		m.zipEntry.pending = false
		m.zipEntry.input.SetValue("")
		m.zipEntry.input.Focus()
		return textinput.Blink
	case "a":
		if m.options.AtlasEnabled && m.options.AtlasStatus == "READY" && len(m.options.AtlasLocations) > 0 {
			m.previousRoute, m.route = RouteLaunch, RouteAtlasBrowser
			m.atlas.selected = 0
		}
	}
	return nil
}

func (m *Model) updateHomeBase(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.homeBase.name.Blur()
		m.homeBase.zip.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.homeBase.focus = 1 - m.homeBase.focus
		if m.homeBase.focus == 0 {
			m.homeBase.zip.Blur()
			m.homeBase.name.Focus()
		} else {
			m.homeBase.name.Blur()
			m.homeBase.zip.Focus()
		}
		return nil
	}
	if key == "enter" && m.homeBase.focus == 1 {
		zip := strings.TrimSpace(m.homeBase.zip.Value())
		if !zipPattern.MatchString(zip) {
			m.homeBase.error = "Enter a valid 5-digit ZIP or ZIP+4 code."
			return nil
		}
		name := strings.TrimSpace(m.homeBase.name.Value())
		if name == "" {
			m.homeBase.error = "A location label is required."
			return nil
		}
		preserveCoordinates := m.options.HomeBaseZIP != "" && m.options.HomeBaseZIP == zip
		latitude, longitude := m.options.HomeBaseLatitude, m.options.HomeBaseLongitude
		if !preserveCoordinates {
			latitude, longitude = nil, nil
		}
		if m.options.SaveHomeBase == nil {
			m.homeBase.error = "Home Base persistence is not initialized."
			return nil
		}
		if err := m.options.SaveHomeBase(HomeBase{Name: name, ZIP: zip}); err != nil {
			m.homeBase.error = err.Error()
			return nil
		}
		m.options.HomeBaseName, m.options.HomeBaseZIP = name, zip
		m.options.HomeBaseLocation = "ZIP " + zip
		m.options.HomeBaseLatitude, m.options.HomeBaseLongitude = latitude, longitude
		m.options.HomeBaseConfigured = true
		if m.options.HomeBaseTimezone != "" {
			m.options.ForecastTimezone = m.options.HomeBaseTimezone
		}
		m.missionPlan.origin = Origin{Kind: "Home Base", Label: name, ZIP: zip, Timezone: m.options.HomeBaseTimezone, Latitude: latitude, Longitude: longitude}
		m.missionPlan.status = "Home Base saved. Select tonight's targets."
		m.homeBase.name.Blur()
		m.homeBase.zip.Blur()
		return m.beginTargetSelection()
	}
	var command tea.Cmd
	if m.homeBase.focus == 0 {
		m.homeBase.name, command = m.homeBase.name.Update(message)
	} else {
		m.homeBase.zip, command = m.homeBase.zip.Update(message)
	}
	return command
}

func (m *Model) updateZIP(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.zipEntry.input.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		zip := strings.TrimSpace(m.zipEntry.input.Value())
		if !zipPattern.MatchString(zip) {
			m.zipEntry.error = "Enter a valid 5-digit ZIP or ZIP+4 code."
			return nil
		}
		if m.options.ResolveZIP != nil {
			m.zipEntry.input.Blur()
			m.zipEntry.pending = true
			m.zipEntry.error = ""
			return func() tea.Msg {
				origin, err := m.options.ResolveZIP(zip)
				return zipResolutionMsg{origin: origin, err: err}
			}
		}
		m.setMissionOrigin(Origin{Kind: "ZIP Code", Label: "ZIP " + zip, ZIP: zip, Timezone: m.options.ScheduleTimezone})
		m.missionPlan.status = "ZIP origin accepted. Geocoding is not configured; coordinates are unknown."
		m.zipEntry.input.Blur()
		return m.beginTargetSelection()
	}
	var command tea.Cmd
	m.zipEntry.input, command = m.zipEntry.input.Update(message)
	return command
}

func (m *Model) updateAtlas(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	switch key {
	case "up", "k":
		if m.atlas.selected > 0 {
			m.atlas.selected--
		}
	case "down", "j":
		if m.atlas.selected < len(m.options.AtlasLocations)-1 {
			m.atlas.selected++
		}
	case "enter":
		if len(m.options.AtlasLocations) == 0 {
			m.error = ErrorModel{title: "ATLAS UNAVAILABLE", detail: "No local Atlas locations are loaded.", context: "ATLAS"}
			m.route = RouteError
			return nil
		}
		site := m.options.AtlasLocations[m.atlas.selected]
		latitude, longitude := site.Latitude, site.Longitude
		m.setMissionOrigin(Origin{Kind: "SkyBase Atlas", Label: site.Name, Timezone: site.Timezone, Latitude: &latitude, Longitude: &longitude})
		m.missionPlan.status = "Atlas observing site selected. Select tonight's targets."
		return m.beginTargetSelection()
	}
	return nil
}

func (m *Model) updateAtlasImport(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.atlasImport.path.Blur()
		m.atlasImport.version.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.atlasImport.focus = 1 - m.atlasImport.focus
		if m.atlasImport.focus == 0 {
			m.atlasImport.version.Blur()
			m.atlasImport.path.Focus()
		} else {
			m.atlasImport.path.Blur()
			m.atlasImport.version.Focus()
		}
		return nil
	}
	if key == "enter" && m.atlasImport.focus == 1 {
		path := strings.TrimSpace(m.atlasImport.path.Value())
		version := strings.TrimSpace(m.atlasImport.version.Value())
		if path == "" || version == "" {
			m.atlasImport.error = "A CSV path and catalog version are required."
			return nil
		}
		if m.options.SaveAtlas == nil {
			m.atlasImport.error = "Atlas import is not initialized."
			return nil
		}
		locations, err := m.options.SaveAtlas(path, version)
		if err != nil {
			m.atlasImport.error = err.Error()
			return nil
		}
		if len(locations) == 0 {
			m.atlasImport.error = "The imported catalog contains no locations."
			return nil
		}
		m.options.AtlasLocations = locations
		m.options.AtlasStatus = "READY"
		m.options.AtlasEnabled = true
		m.atlasImport.path.Blur()
		m.atlasImport.version.Blur()
		m.atlasImport.error = ""
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	if m.atlasImport.focus == 0 {
		m.atlasImport.path, command = m.atlasImport.path.Update(message)
	} else {
		m.atlasImport.version, command = m.atlasImport.version.Update(message)
	}
	return command
}

func (m *Model) updateAtlasExport(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.atlasExport.path.Blur()
		m.atlasExport.error = ""
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		path := strings.TrimSpace(m.atlasExport.path.Value())
		if path == "" {
			m.atlasExport.error = "A CSV export path is required."
			return nil
		}
		if m.options.ExportAtlas == nil {
			m.atlasExport.error = "Atlas export is not initialized."
			return nil
		}
		if err := m.options.ExportAtlas(path); err != nil {
			m.atlasExport.error = err.Error()
			return nil
		}
		m.atlasExport.path.Blur()
		m.atlasExport.error = ""
		m.settings.notice = "Atlas contribution CSV exported to " + path
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	m.atlasExport.path, command = m.atlasExport.path.Update(message)
	return command
}

func (m *Model) updateTarget(key string) tea.Cmd {
	if key == "esc" || key == "b" {
		m.route = m.previousRoute
		return nil
	}
	if m.target.chosen == nil {
		m.target.chosen = make(map[string]bool)
	}
	switch key {
	case "up", "k":
		if m.target.selected > 0 {
			m.target.selected--
		}
	case "down", "j":
		if m.target.selected < len(m.options.Targets)-1 {
			m.target.selected++
		}
	case " ", "space", "enter":
		if len(m.options.Targets) == 0 {
			m.error = ErrorModel{title: "TARGET CATALOG UNAVAILABLE", detail: "No offline celestial targets are loaded.", context: "TARGET"}
			m.route = RouteError
			return nil
		}
		target := m.options.Targets[m.target.selected]
		m.target.chosen[target.ID] = !m.target.chosen[target.ID]
		m.missionPlan.targets = m.selectedTargets()
		if len(m.missionPlan.targets) > 0 {
			m.missionPlan.target = m.missionPlan.targets[0].Name
			if m.options.TargetSummary != nil {
				m.missionPlan.targetInfo = m.options.TargetSummary(m.missionPlan.origin, m.missionPlan.targets[0])
			}
			if m.options.TargetForecastSummary != nil {
				m.missionPlan.targetForecast = m.options.TargetForecastSummary(m.missionPlan.origin, m.missionPlan.targets[0], m.missionPlan.forecastPoints)
			}
		}
		m.refreshMissionAutomation()
		m.missionPlan.status = fmt.Sprintf("%d target(s) selected for this mission.", len(m.missionPlan.targets))
	case "c":
		m.missionPlan.targets = m.selectedTargets()
		if len(m.missionPlan.targets) > 0 {
			m.missionPlan.target = m.missionPlan.targets[0].Name
		}
		m.refreshMissionAutomation()
		m.missionPlan.status = fmt.Sprintf("Target sequence saved locally: %d target(s).", len(m.missionPlan.targets))
		m.route = m.previousRoute
	}
	return nil
}

func (m Model) selectedTargets() []TargetSite {
	selected := make([]TargetSite, 0, len(m.target.chosen))
	for _, target := range m.options.Targets {
		if m.target.chosen[target.ID] {
			selected = append(selected, target)
		}
	}
	return selected
}

func (m *Model) updateForecast(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	switch key {
	case "d":
		m.missionPlan.darkOnly = !m.missionPlan.darkOnly
		m.missionPlan.forecastSelected = 0
	case "c":
		m.missionPlan.cloudOnly = !m.missionPlan.cloudOnly
		m.missionPlan.forecastSelected = 0
	case "p":
		m.missionPlan.precipOnly = !m.missionPlan.precipOnly
		m.missionPlan.forecastSelected = 0
	case "up", "k":
		if m.missionPlan.forecastSelected > 0 {
			m.missionPlan.forecastSelected--
		}
	case "down", "j":
		if m.missionPlan.forecastSelected < len(m.filteredForecastPoints())-1 {
			m.missionPlan.forecastSelected++
		}
	case "enter":
		points := m.filteredForecastPoints()
		if len(points) == 0 {
			m.error = ErrorModel{title: "FORECAST UNAVAILABLE", detail: "No hourly forecast points are available for this origin.", context: "WEATHER"}
			m.route = RouteError
			return nil
		}
		m.missionPlan.forecast = formatForecastPoint(points[m.missionPlan.forecastSelected], m.options.ForecastTimezone)
		m.missionPlan.status = "Forecast hour selected for mission planning."
		m.route = RouteMissionPlanning
	}
	return nil
}

func (m *Model) updateRoutePlan(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		m.missionPlan.status = "Route plan reviewed. Driving time requires a routing provider."
		m.route = m.previousRoute
	}
	return nil
}

func (m *Model) updateEquipmentSetup(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.equipmentSetup.name.Blur()
		m.equipmentSetup.description.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.equipmentSetup.focus = 1 - m.equipmentSetup.focus
		if m.equipmentSetup.focus == 0 {
			m.equipmentSetup.description.Blur()
			m.equipmentSetup.name.Focus()
		} else {
			m.equipmentSetup.name.Blur()
			m.equipmentSetup.description.Focus()
		}
		return nil
	}
	if key == "enter" && m.equipmentSetup.focus == 1 {
		name := strings.TrimSpace(m.equipmentSetup.name.Value())
		if name == "" {
			m.equipmentSetup.error = "Equipment profile name is required."
			return nil
		}
		if m.options.SaveEquipment == nil {
			m.equipmentSetup.error = "Equipment storage is not initialized."
			return nil
		}
		id := fmt.Sprintf("equipment-%d", time.Now().UnixNano())
		profile := EquipmentProfile{ID: id, Name: name, Description: strings.TrimSpace(m.equipmentSetup.description.Value())}
		if err := m.options.SaveEquipment(profile); err != nil {
			m.equipmentSetup.error = err.Error()
			return nil
		}
		m.options.EquipmentProfiles = append(m.options.EquipmentProfiles, profile)
		m.equipmentSetup.name.Blur()
		m.equipmentSetup.description.Blur()
		m.equipmentSetup.error = ""
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	if m.equipmentSetup.focus == 0 {
		m.equipmentSetup.name, command = m.equipmentSetup.name.Update(message)
	} else {
		m.equipmentSetup.description, command = m.equipmentSetup.description.Update(message)
	}
	return command
}

func (m *Model) updateEquipment(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	switch key {
	case "up", "k":
		if m.equipment.selected > 0 {
			m.equipment.selected--
		}
	case "down", "j":
		if m.equipment.selected < len(m.options.EquipmentProfiles)-1 {
			m.equipment.selected++
		}
	case "enter":
		if len(m.options.EquipmentProfiles) == 0 {
			m.error = ErrorModel{title: "EQUIPMENT UNAVAILABLE", detail: "Create an equipment profile in Settings first.", context: "EQUIPMENT"}
			m.route = RouteError
			return nil
		}
		profile := m.options.EquipmentProfiles[m.equipment.selected]
		m.missionPlan.equipmentID = profile.ID
		m.missionPlan.equipment = profile.Name
		m.missionPlan.status = "Equipment profile selected; mission is ready to configure."
		m.route = RouteMissionPlanning
	}
	return nil
}

func (m *Model) updateEquipmentInventory(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	if len(m.options.EquipmentProfiles) == 0 {
		return nil
	}
	switch key {
	case "up", "k":
		if m.equipmentInventory.selected > 0 {
			m.equipmentInventory.selected--
		}
	case "down", "j":
		if m.equipmentInventory.selected < len(m.options.EquipmentProfiles)-1 {
			m.equipmentInventory.selected++
		}
	case "n":
		m.previousRoute, m.route = RouteEquipmentInventory, RouteEquipmentItemSetup
		m.equipmentItemSetup = newEquipmentItemSetup()
		m.equipmentItemSetup.name.Focus()
		return textinput.Blink
	case "enter":
		profile := m.options.EquipmentProfiles[m.equipmentInventory.selected]
		m.equipmentReadiness.profileID = profile.ID
		m.previousRoute, m.route = RouteEquipmentInventory, RouteEquipmentReadiness
	}
	return nil
}

func (m *Model) updateEquipmentItemSetup(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.equipmentItemSetup.name.Blur()
		m.equipmentItemSetup.category.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "ctrl+r" {
		m.equipmentItemSetup.required = !m.equipmentItemSetup.required
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.equipmentItemSetup.focus = 1 - m.equipmentItemSetup.focus
		if m.equipmentItemSetup.focus == 0 {
			m.equipmentItemSetup.category.Blur()
			m.equipmentItemSetup.name.Focus()
		} else {
			m.equipmentItemSetup.name.Blur()
			m.equipmentItemSetup.category.Focus()
		}
		return nil
	}
	if key == "enter" && m.equipmentItemSetup.focus == 1 {
		name := strings.TrimSpace(m.equipmentItemSetup.name.Value())
		category := strings.TrimSpace(m.equipmentItemSetup.category.Value())
		if name == "" || category == "" {
			m.equipmentItemSetup.error = "Item name and category are required."
			return nil
		}
		if m.options.SaveEquipmentItem == nil || len(m.options.EquipmentProfiles) == 0 {
			m.equipmentItemSetup.error = "Equipment inventory is not initialized."
			return nil
		}
		profile := m.options.EquipmentProfiles[m.equipmentInventory.selected]
		item := EquipmentItem{ID: fmt.Sprintf("item-%d", time.Now().UnixNano()), ProfileID: profile.ID, Name: name, Category: category, Required: m.equipmentItemSetup.required}
		if err := m.options.SaveEquipmentItem(item); err != nil {
			m.equipmentItemSetup.error = err.Error()
			return nil
		}
		m.options.EquipmentItems = append(m.options.EquipmentItems, item)
		m.equipmentItemSetup.name.Blur()
		m.equipmentItemSetup.category.Blur()
		m.equipmentItemSetup.error = ""
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	if m.equipmentItemSetup.focus == 0 {
		m.equipmentItemSetup.name, command = m.equipmentItemSetup.name.Update(message)
	} else {
		m.equipmentItemSetup.category, command = m.equipmentItemSetup.category.Update(message)
	}
	return command
}

func (m *Model) updateMissionSchedule(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.schedule.start.Blur()
		m.schedule.end.Blur()
		m.route = m.previousRoute
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.schedule.focus = 1 - m.schedule.focus
		if m.schedule.focus == 0 {
			m.schedule.end.Blur()
			m.schedule.start.Focus()
		} else {
			m.schedule.start.Blur()
			m.schedule.end.Focus()
		}
		return nil
	}
	if key == "enter" && m.schedule.focus == 1 {
		location := m.scheduleLocation()
		start, startErr := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(m.schedule.start.Value()), location)
		end, endErr := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(m.schedule.end.Value()), location)
		if startErr != nil || endErr != nil {
			m.schedule.error = "Use local time format YYYY-MM-DD HH:MM for both fields."
			return nil
		}
		if !end.After(start) {
			m.schedule.error = "Mission window must end after it starts."
			return nil
		}
		start, end = start.UTC(), end.UTC()
		m.missionPlan.plannedStart, m.missionPlan.plannedEnd = &start, &end
		m.missionPlan.status = "Mission window saved locally for planning."
		m.schedule.start.Blur()
		m.schedule.end.Blur()
		m.schedule.error = ""
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	if m.schedule.focus == 0 {
		m.schedule.start, command = m.schedule.start.Update(message)
	} else {
		m.schedule.end, command = m.schedule.end.Update(message)
	}
	return command
}

func (m *Model) updatePlugins(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	switch key {
	case "up", "k":
		if m.plugins.selected > 0 {
			m.plugins.selected--
		}
	case "down", "j":
		if m.plugins.selected < len(m.options.Plugins)-1 {
			m.plugins.selected++
		}
	}
	return nil
}

func (m *Model) updateBackup(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.backup.path.Blur()
		m.backup.error = ""
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		path := strings.TrimSpace(m.backup.path.Value())
		if path == "" {
			m.backup.error = "A backup path is required."
			return nil
		}
		if m.options.CreateBackup == nil {
			m.backup.error = "Database backup is not initialized."
			return nil
		}
		if err := m.options.CreateBackup(path); err != nil {
			m.backup.error = err.Error()
			return nil
		}
		m.backup.path.Blur()
		m.backup.error = ""
		m.settings.notice = "Database backup created at " + path
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	m.backup.path, command = m.backup.path.Update(message)
	return command
}

func (m *Model) updateSyncExport(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.syncExport.path.Blur()
		m.syncExport.error = ""
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		path := strings.TrimSpace(m.syncExport.path.Value())
		if path == "" {
			m.syncExport.error = "A sync bundle path is required."
			return nil
		}
		if m.options.ExportSync == nil {
			m.syncExport.error = "Sync export is not initialized."
			return nil
		}
		if err := m.options.ExportSync(path); err != nil {
			m.syncExport.error = err.Error()
			return nil
		}
		m.syncExport.path.Blur()
		m.syncExport.error = ""
		m.settings.notice = "Sync bundle exported to " + path
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	m.syncExport.path, command = m.syncExport.path.Update(message)
	return command
}

func (m *Model) updateSyncImport(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.syncImport.path.Blur()
		m.syncImport.error = ""
		m.route = m.previousRoute
		return nil
	}
	if key == "enter" {
		path := strings.TrimSpace(m.syncImport.path.Value())
		if path == "" {
			m.syncImport.error = "A sync bundle path is required."
			return nil
		}
		if m.options.ImportSync == nil {
			m.syncImport.error = "Sync import is not initialized."
			return nil
		}
		result, err := m.options.ImportSync(path)
		if err != nil {
			m.syncImport.error = err.Error()
			return nil
		}
		m.syncImport.path.Blur()
		m.syncImport.error = ""
		m.settings.notice = "Sync bundle merged: " + result
		m.route = m.previousRoute
		return nil
	}
	var command tea.Cmd
	m.syncImport.path, command = m.syncImport.path.Update(message)
	return command
}

func (m *Model) updateCommandPalette(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.palette.input.Blur()
		m.palette.error = ""
		m.route = m.previousRoute
		return nil
	}
	commands := m.filteredPaletteCommands()
	switch key {
	case "up", "k":
		if m.palette.selected > 0 {
			m.palette.selected--
		}
		return nil
	case "down", "j":
		if m.palette.selected < len(commands)-1 {
			m.palette.selected++
		}
		return nil
	case "enter":
		if len(commands) == 0 {
			m.palette.error = "No matching local commands. Clear the filter and try again."
			return nil
		}
		return m.executePaletteCommand(commands[m.palette.selected])
	}
	var command tea.Cmd
	m.palette.input, command = m.palette.input.Update(message)
	m.palette.selected = 0
	m.palette.error = ""
	return command
}

func (m *Model) loadMissionArchive() tea.Cmd {
	m.missionArchive.selected = 0
	m.missionArchive.error = ""
	if m.options.LoadMissions == nil {
		m.missionArchive.error = "Mission archive is not initialized."
		return nil
	}
	missions, err := m.options.LoadMissions()
	if err != nil {
		m.missionArchive.error = err.Error()
		return nil
	}
	m.missionArchive.missions = missions
	return nil
}

func (m *Model) updateMissionArchive(key string) tea.Cmd {
	if key == "esc" {
		m.route = m.previousRoute
		return nil
	}
	switch key {
	case "up", "k":
		if m.missionArchive.selected > 0 {
			m.missionArchive.selected--
		}
	case "down", "j":
		if m.missionArchive.selected < len(m.missionArchive.missions)-1 {
			m.missionArchive.selected++
		}
	case "enter":
		if len(m.missionArchive.missions) > 0 {
			m.missionDetail = m.missionArchive.missions[m.missionArchive.selected]
			m.route = RouteMissionDetail
		}
	}
	return nil
}

func (m Model) paletteCommands() []PaletteCommand {
	commands := []PaletteCommand{
		{Key: "launch", Label: "Launch Console", Description: "Return to the mission-origin launch screen"},
		{Key: "settings", Label: "Settings", Description: "Review supported local configuration"},
		{Key: "help", Label: "Help", Description: "Open keyboard and console guidance"},
	}
	if m.options.AtlasEnabled && m.options.AtlasStatus == "READY" && len(m.options.AtlasLocations) > 0 {
		commands = append(commands, PaletteCommand{Key: "atlas", Label: "SkyBase Atlas", Description: "Browse available local launch sites"})
	}
	if len(m.options.EquipmentProfiles) > 0 {
		commands = append(commands, PaletteCommand{Key: "equipment", Label: "Equipment Browser", Description: "Review available observing setups"})
	}
	if m.options.PluginsStatus != "DISABLED" {
		commands = append(commands, PaletteCommand{Key: "plugins", Label: "Plugin Registry", Description: "Inspect validated local plugin metadata"})
	}
	if m.options.CreateBackup != nil {
		commands = append(commands, PaletteCommand{Key: "backup", Label: "Database Backup", Description: "Create a consistent local SQLite backup"})
	}
	if m.options.ExportAtlas != nil && m.options.AtlasEnabled && m.options.AtlasStatus == "READY" {
		commands = append(commands, PaletteCommand{Key: "atlas-export", Label: "Export SkyBase Atlas", Description: "Write a provenance-preserving contribution CSV"})
	}
	if m.options.LoadMissions != nil {
		commands = append(commands, PaletteCommand{Key: "missions", Label: "Mission Archive", Description: "Review locally persisted missions"})
	}
	if m.options.OpenObsidianVault != nil {
		commands = append(commands, PaletteCommand{Key: "obsidian", Label: "Open Obsidian Vault", Description: "Open the configured vault at the NightOps notes directory"})
	}
	if m.options.ExportSync != nil {
		commands = append(commands, PaletteCommand{Key: "sync-export", Label: "Export Sync Bundle", Description: "Create a portable offline JSON bundle"})
	}
	if m.options.ImportSync != nil {
		commands = append(commands, PaletteCommand{Key: "sync-import", Label: "Import Sync Bundle", Description: "Merge a portable bundle without deleting local data"})
	}
	commands = append(commands, PaletteCommand{Key: "quit", Label: "Quit NightOps", Description: "Exit the application"})
	return commands
}

func (m Model) filteredPaletteCommands() []PaletteCommand {
	query := strings.ToLower(strings.TrimSpace(m.palette.input.Value()))
	commands := m.paletteCommands()
	if query == "" {
		return commands
	}
	filtered := make([]PaletteCommand, 0, len(commands))
	for _, command := range commands {
		searchable := strings.ToLower(command.Key + " " + command.Label + " " + command.Description)
		if strings.Contains(searchable, query) {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

func (m *Model) executePaletteCommand(command PaletteCommand) tea.Cmd {
	returnTo := m.previousRoute
	m.palette.input.Blur()
	m.palette.error = ""
	switch command.Key {
	case "launch":
		m.route = RouteLaunch
		m.launch.selected = 0
	case "settings":
		m.previousRoute, m.route = returnTo, RouteSettings
	case "help":
		m.previousRoute, m.route = returnTo, RouteHelp
	case "atlas":
		m.previousRoute, m.route = returnTo, RouteAtlasBrowser
		m.atlas.selected = 0
	case "equipment":
		m.previousRoute, m.route = returnTo, RouteEquipmentBrowser
		m.equipment.selected = 0
	case "plugins":
		m.previousRoute, m.route = returnTo, RoutePlugins
		m.plugins.selected = 0
	case "backup":
		m.previousRoute, m.route = returnTo, RouteBackup
		m.backup.path.SetValue(m.options.BackupDefaultPath)
		m.backup.error = ""
		m.backup.path.Focus()
		return textinput.Blink
	case "atlas-export":
		m.previousRoute, m.route = returnTo, RouteAtlasExport
		m.atlasExport.path.SetValue(m.options.AtlasExportDefaultPath)
		m.atlasExport.error = ""
		m.atlasExport.path.Focus()
		return textinput.Blink
	case "missions":
		m.previousRoute, m.route = returnTo, RouteMissionArchive
		return m.loadMissionArchive()
	case "obsidian":
		if m.options.OpenObsidianVault == nil {
			m.palette.error = "Obsidian vault opening is not configured."
			return nil
		}
		if err := m.options.OpenObsidianVault(); err != nil {
			m.palette.error = "Obsidian vault could not be opened: " + err.Error()
		}
	case "sync-export":
		m.previousRoute, m.route = returnTo, RouteSyncExport
		m.syncExport.path.SetValue(m.options.SyncExportDefaultPath)
		m.syncExport.error = ""
		m.syncExport.path.Focus()
		return textinput.Blink
	case "sync-import":
		m.previousRoute, m.route = returnTo, RouteSyncImport
		m.syncImport.path.SetValue("")
		m.syncImport.error = ""
		m.syncImport.path.Focus()
		return textinput.Blink
	case "quit":
		m.quitting = true
		return tea.Quit
	}
	return nil
}

func (m Model) scheduleLocation() *time.Location {
	timezone := m.missionPlan.origin.Timezone
	if timezone == "" {
		timezone = m.options.ScheduleTimezone
	}
	if timezone != "" {
		if location, err := time.LoadLocation(timezone); err == nil {
			return location
		}
	}
	return time.UTC
}

func (m *Model) openMissionSchedule() tea.Cmd {
	location := m.scheduleLocation()
	if m.missionPlan.plannedStart != nil && m.missionPlan.plannedEnd != nil {
		m.schedule.start.SetValue(m.missionPlan.plannedStart.In(location).Format("2006-01-02 15:04"))
		m.schedule.end.SetValue(m.missionPlan.plannedEnd.In(location).Format("2006-01-02 15:04"))
	} else {
		m.schedule.start.SetValue("")
		m.schedule.end.SetValue("")
	}
	m.schedule.focus = 0
	m.schedule.error = ""
	m.schedule.end.Blur()
	m.schedule.start.Focus()
	m.previousRoute, m.route = RouteMissionPlanning, RouteMissionSchedule
	return textinput.Blink
}

func (m *Model) updateMissionPlan(key string) tea.Cmd {
	actions := m.missionActions()
	switch key {
	case "up", "k":
		if m.missionPlan.selected > 0 {
			m.missionPlan.selected--
		}
	case "down", "j":
		if m.missionPlan.selected < len(actions)-1 {
			m.missionPlan.selected++
		}
	case "enter":
		switch m.missionPlan.selected {
		case 0:
			m.previousRoute, m.route = RouteMissionPlanning, RouteMissionReview
			m.missionReview.selected = 0
			return nil
		case 1:
			m.previousRoute, m.route = RouteMissionPlanning, RouteLaunch
		case 2:
			if m.missionPlan.origin.Kind != "Home Base" && m.options.SaveHomeBase == nil {
				m.error = ErrorModel{title: "HOME BASE UNAVAILABLE", detail: "Home Base persistence is not initialized.", context: "HOME_BASE"}
				m.route = RouteError
				return nil
			}
			if m.missionPlan.origin.Kind != "Home Base" {
				name := m.missionPlan.origin.Label
				if m.missionPlan.origin.Kind == "ZIP Code" && name == "ZIP "+m.missionPlan.origin.ZIP {
					name = "Home Base"
				}
				if err := m.options.SaveHomeBase(HomeBase{Name: name, ZIP: m.missionPlan.origin.ZIP}); err != nil {
					m.error = ErrorModel{title: "HOME BASE SAVE FAILED", detail: err.Error(), context: "HOME_BASE"}
					m.route = RouteError
					return nil
				}
				m.missionPlan.origin.Label = name
			}
			m.options.HomeBaseConfigured = true
			m.options.HomeBaseName = m.missionPlan.origin.Label
			m.options.HomeBaseZIP = m.missionPlan.origin.ZIP
			m.missionPlan.status = "Origin saved as Home Base."
		case 3:
			m.previousRoute, m.route = RouteMissionPlanning, RouteLaunch
		default:
			if m.missionPlan.selected == m.targetActionIndex() {
				m.previousRoute, m.route = RouteMissionPlanning, RouteTargetBrowser
				m.target.selected = 0
			} else if m.missionPlan.selected == m.forecastActionIndex() {
				m.previousRoute, m.route = RouteMissionPlanning, RouteForecastBrowser
				m.missionPlan.forecastSelected = 0
			} else if m.missionPlan.selected == m.routeActionIndex() {
				m.previousRoute, m.route = RouteMissionPlanning, RouteRoutePlan
			} else if m.missionPlan.selected == m.equipmentActionIndex() {
				m.previousRoute, m.route = RouteMissionPlanning, RouteEquipmentBrowser
				m.equipment.selected = 0
			} else if m.missionPlan.selected == m.readinessActionIndex() {
				m.equipmentReadiness.profileID = m.missionPlan.equipmentID
				m.previousRoute, m.route = RouteMissionPlanning, RouteEquipmentReadiness
			} else if m.missionPlan.selected == m.scheduleActionIndex() {
				return m.openMissionSchedule()
			} else if m.missionPlan.selected == m.telescopeActionIndex() {
				return m.slewToSelectedTarget()
			} else if m.missionPlan.selected == m.aiBriefActionIndex() {
				return m.requestMissionBrief()
			}
		}
	case "esc":
		m.route = m.previousRoute
	}
	return nil
}

func (m *Model) updateMissionReview(key string) tea.Cmd {
	const actionCount = 3
	switch key {
	case "up", "k":
		if m.missionReview.selected > 0 {
			m.missionReview.selected--
		}
	case "down", "j":
		if m.missionReview.selected < actionCount-1 {
			m.missionReview.selected++
		}
	case "enter":
		switch m.missionReview.selected {
		case 0:
			return m.launchReviewedMission(true)
		case 1:
			return m.launchReviewedMission(false)
		default:
			m.route = RouteMissionPlanning
		}
	case "esc", "b":
		m.route = RouteMissionPlanning
	}
	return nil
}

func (m *Model) launchReviewedMission(openVault bool) tea.Cmd {
	if m.options.CreateMission == nil {
		m.error = ErrorModel{title: "MISSION CREATION UNAVAILABLE", detail: "Mission persistence is not initialized.", context: "MISSION"}
		m.route = RouteError
		return nil
	}
	missionID, err := m.options.CreateMission(m.missionPlan.origin, m.missionPlan.equipmentID, m.missionPlan.plannedStart, m.missionPlan.plannedEnd)
	if err != nil {
		m.error = ErrorModel{title: "MISSION SAVE FAILED", detail: err.Error(), context: "MISSION"}
		m.route = RouteError
		return nil
	}
	if len(m.missionPlan.targets) > 0 {
		if m.options.SaveMissionTargets == nil {
			m.error = ErrorModel{title: "TARGET PLAN UNAVAILABLE", detail: "The selected target sequence could not be persisted.", context: "MISSION"}
			m.route = RouteError
			return nil
		}
		if err := m.options.SaveMissionTargets(missionID, m.missionPlan.targets); err != nil {
			m.error = ErrorModel{title: "TARGET PLAN SAVE FAILED", detail: err.Error(), context: "MISSION"}
			m.route = RouteError
			return nil
		}
	}
	if openVault && m.options.OpenObsidianVault != nil {
		if err := m.options.OpenObsidianVault(); err != nil {
			m.missionPlan.status = "Mission saved; Obsidian could not be opened: " + err.Error()
		}
	}
	m.operation = OperationModel{missionID: missionID, status: "PLANNED"}
	m.blastOff = 10
	m.previousRoute, m.route = RouteMissionReview, RouteBlastOff
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return blastOffTickMsg{} })
}

func (m *Model) updateOperation(key string) tea.Cmd {
	switch key {
	case "enter":
		if m.operation.status == "PLANNED" {
			if m.options.StartMission == nil {
				m.error = ErrorModel{title: "OPERATION UNAVAILABLE", detail: "Mission lifecycle service is not initialized.", context: "MISSION"}
				m.route = RouteError
				return nil
			}
			if err := m.options.StartMission(m.operation.missionID); err != nil {
				m.error = ErrorModel{title: "OPERATION START FAILED", detail: err.Error(), context: "MISSION"}
				m.route = RouteError
				return nil
			}
			m.operation.status = "LAUNCHED"
		} else if m.operation.status == "LAUNCHED" {
			if m.options.ActivateMission == nil {
				m.error = ErrorModel{title: "OBSERVING UNAVAILABLE", detail: "Mission lifecycle service is not initialized.", context: "MISSION"}
				m.route = RouteError
				return nil
			}
			if err := m.options.ActivateMission(m.operation.missionID); err != nil {
				m.error = ErrorModel{title: "OBSERVING START FAILED", detail: err.Error(), context: "MISSION"}
				m.route = RouteError
				return nil
			}
			m.operation.status = "ACTIVE"
		}
	case "a":
		if m.operation.status == "LAUNCHED" || m.operation.status == "ACTIVE" {
			m.observation = newObservationEntry()
			m.observation.target.Focus()
			m.route = RouteObservationEntry
			return textinput.Blink
		}
	case "c":
		if m.operation.status == "ACTIVE" {
			if m.options.CompleteMission == nil {
				m.error = ErrorModel{title: "DEBRIEF UNAVAILABLE", detail: "Mission lifecycle service is not initialized.", context: "MISSION"}
				m.route = RouteError
				return nil
			}
			if err := m.options.CompleteMission(m.operation.missionID); err != nil {
				m.error = ErrorModel{title: "MISSION COMPLETION FAILED", detail: err.Error(), context: "MISSION"}
				m.route = RouteError
				return nil
			}
			m.debrief = newDebrief()
			m.debrief.missionID = m.operation.missionID
			m.debrief.count = len(m.operation.observations)
			m.debrief.summary.Focus()
			m.route = RouteDebrief
			return textinput.Blink
		}
	case "esc":
		m.route = m.previousRoute
	}
	return nil
}

func (m *Model) updateObservation(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" {
		m.route = RouteOperation
		return nil
	}
	if key == "tab" || key == "down" || key == "up" {
		m.observation.focus = 1 - m.observation.focus
		if m.observation.focus == 0 {
			m.observation.notes.Blur()
			m.observation.target.Focus()
		} else {
			m.observation.target.Blur()
			m.observation.notes.Focus()
		}
		return nil
	}
	if key == "enter" && m.observation.focus == 1 {
		target, notes := strings.TrimSpace(m.observation.target.Value()), strings.TrimSpace(m.observation.notes.Value())
		if target == "" {
			m.observation.error = "A target name is required."
			return nil
		}
		if m.options.RecordObservation == nil {
			m.observation.error = "Observation storage is not initialized."
			return nil
		}
		if err := m.options.RecordObservation(m.operation.missionID, target, notes); err != nil {
			m.observation.error = err.Error()
			return nil
		}
		m.operation.observations = append(m.operation.observations, target)
		m.route = RouteOperation
		return nil
	}
	var command tea.Cmd
	if m.observation.focus == 0 {
		m.observation.target, command = m.observation.target.Update(message)
	} else {
		m.observation.notes, command = m.observation.notes.Update(message)
	}
	return command
}

func (m *Model) updateDebrief(message tea.KeyMsg) tea.Cmd {
	key := message.String()
	if key == "esc" || key == "b" {
		m.debrief.summary.Blur()
		m.route = RouteLaunch
		return nil
	}
	if key == "enter" {
		if m.debrief.saved {
			m.route = RouteLaunch
			return nil
		}
		summary := strings.TrimSpace(m.debrief.summary.Value())
		if summary == "" {
			m.debrief.error = "A debrief summary is required."
			return nil
		}
		if m.options.RecordDebrief == nil {
			m.debrief.error = "Debrief storage is not initialized."
			return nil
		}
		if err := m.options.RecordDebrief(m.debrief.missionID, summary); err != nil {
			m.debrief.error = err.Error()
			return nil
		}
		m.debrief.summary.Blur()
		m.debrief.error = ""
		m.debrief.saved = true
		return nil
	}
	var command tea.Cmd
	m.debrief.summary, command = m.debrief.summary.Update(message)
	return command
}

func (m *Model) setMissionOrigin(origin Origin) {
	m.missionPlan.origin = origin
	if origin.Timezone != "" {
		m.options.ForecastTimezone = origin.Timezone
	}
	// Target visibility and weather ranking are location-specific. A changed
	// origin must not leave a stale target recommendation on screen.
	m.missionPlan.target = ""
	m.missionPlan.targetInfo = ""
	m.missionPlan.targetForecast = ""
	m.missionPlan.missionWindow = ""
	m.missionPlan.plannedStart = nil
	m.missionPlan.plannedEnd = nil
	m.missionPlan.forecastPoints = nil
	if m.options.AstronomySummary != nil {
		m.missionPlan.astronomy = m.options.AstronomySummary(origin)
	} else if origin.Latitude == nil || origin.Longitude == nil {
		m.missionPlan.astronomy = "unavailable until coordinates are known"
	}
	if m.options.WeatherSummary != nil {
		m.missionPlan.weather = m.options.WeatherSummary(origin)
	} else {
		m.missionPlan.weather = "unavailable"
	}
	if m.options.ForecastSummary != nil {
		m.missionPlan.forecast = m.options.ForecastSummary(origin)
	} else {
		m.missionPlan.forecast = "unavailable"
	}
	if m.options.ForecastPoints != nil {
		m.missionPlan.forecastPoints = m.options.ForecastPoints(origin)
		m.missionPlan.forecastSelected = 0
	}
	if m.options.RouteSummary != nil {
		m.missionPlan.route = m.options.RouteSummary(origin)
	} else {
		m.missionPlan.route = "unavailable"
	}
	m.refreshMissionAutomation()
}

func (m *Model) refreshMissionAutomation() {
	if m.options.MissionWindow != nil {
		start, end, summary := m.options.MissionWindow(m.missionPlan.origin, m.missionPlan.targets)
		m.missionPlan.plannedStart, m.missionPlan.plannedEnd = start, end
		m.missionPlan.missionWindow = summary
	}
	if m.options.WeatherAssessment != nil {
		m.missionPlan.weatherDecision = m.options.WeatherAssessment(m.missionPlan.origin, m.missionPlan.targets)
	}
}

func (m *Model) updateError(key string) tea.Cmd {
	switch key {
	case "r":
		if m.error.context == "GPS" {
			m.route = RouteGPSAcquisition
			m.gps.status = "ACQUIRING CURRENT LOCATION"
			if m.options.LocationProvider == nil {
				m.error.detail = "No GPS adapter is configured for this installation."
				m.route = RouteError
			}
		}
	case "z":
		if m.error.context != "GPS" {
			return nil
		}
		m.previousRoute, m.route = RouteError, RouteZIPEntry
		m.zipEntry.error = ""
		m.zipEntry.input.Focus()
		return textinput.Blink
	case "esc", "b":
		m.route = m.previousRoute
	}
	return nil
}

func (m Model) actions() []launchAction {
	actions := []launchAction{
		{key: "h", label: "LAUNCH FROM HOME BASE", description: m.homeBaseLine()},
		{key: "g", label: "ACQUIRE CURRENT LOCATION", description: "Use a configured GPS adapter"},
		{key: "z", label: "ENTER ZIP CODE", description: "Use a postal code as mission origin"},
	}
	if m.options.AtlasEnabled && m.options.AtlasStatus == "READY" && len(m.options.AtlasLocations) > 0 {
		actions = append(actions, launchAction{key: "a", label: "BROWSE SKYBASE ATLAS", description: m.options.AtlasStatus})
	}
	if m.options.LoadMissions != nil {
		actions = append(actions, launchAction{key: "archive", label: "OPEN MISSION ARCHIVE", description: "Review locally recorded missions"})
	}
	actions = append(actions, launchAction{key: "launch", label: "LAUNCH MISSION", description: "Begin mission planning with the selected origin"})
	return actions
}

func (m Model) homeBaseLine() string {
	name := m.options.HomeBaseName
	if name == "" {
		name = "Not configured"
	}
	location := m.options.HomeBaseLocation
	if location == "" {
		location = "Not configured"
	}
	return name + " · " + location
}

// View renders the active console.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		m.width = 100
	}
	if m.height == 0 {
		m.height = 30
	}
	switch m.route {
	case RouteBoot:
		return m.renderBoot()
	case RouteLaunch:
		return m.renderLaunch()
	case RouteHomeBaseSetup:
		return m.renderHomeBaseSetup()
	case RouteGPSAcquisition:
		return m.renderGPS()
	case RouteZIPEntry:
		return m.renderZIP()
	case RouteAtlasBrowser:
		return m.renderAtlas()
	case RouteAtlasImport:
		return m.renderAtlasImport()
	case RouteAtlasExport:
		return m.renderAtlasExport()
	case RouteTargetBrowser:
		return m.renderTarget()
	case RouteForecastBrowser:
		return m.renderForecast()
	case RouteRoutePlan:
		return m.renderRoutePlan()
	case RouteEquipmentSetup:
		return m.renderEquipmentSetup()
	case RouteEquipmentBrowser:
		return m.renderEquipmentBrowser()
	case RouteEquipmentInventory:
		return m.renderEquipmentInventory()
	case RouteEquipmentItemSetup:
		return m.renderEquipmentItemSetup()
	case RouteEquipmentReadiness:
		return m.renderEquipmentReadiness()
	case RouteMissionSchedule:
		return m.renderMissionSchedule()
	case RouteBackup:
		return m.renderBackup()
	case RoutePlugins:
		return m.renderPlugins()
	case RouteCommandPalette:
		return m.renderCommandPalette()
	case RouteMissionArchive:
		return m.renderMissionArchive()
	case RouteMissionDetail:
		return m.renderMissionDetail()
	case RouteSyncExport:
		return m.renderSyncExport()
	case RouteSyncImport:
		return m.renderSyncImport()
	case RouteMissionPlanning:
		return m.renderMissionPlan()
	case RouteMissionReview:
		return m.renderMissionReview()
	case RouteBlastOff:
		return m.renderBlastOff()
	case RouteDeepSpace:
		return m.renderDeepSpace()
	case RouteOperation:
		return m.renderOperation()
	case RouteObservationEntry:
		return m.renderObservationEntry()
	case RouteDebrief:
		return m.renderDebrief()
	case RouteSettings:
		return m.renderSettings()
	case RouteHelp:
		return m.renderHelp()
	case RouteError:
		return m.renderError()
	default:
		return ""
	}
}

func (m Model) renderBoot() string {
	checks := make([]string, 0, len(bootStages))
	for index, stage := range bootStages {
		glyph, style, suffix := "·", m.theme.MutedStyle(), "STANDBY"
		if index < m.bootStage {
			glyph, style, suffix = "✓", m.theme.GoodStyle(), "READY"
		} else if index == m.bootStage && index < len(bootStages)-1 {
			glyph, style, suffix = m.spinner.View(), m.theme.AccentStyle(), "ACTIVE"
		}
		checks = append(checks, style.Render(fmt.Sprintf("%s  %-39s %s", glyph, stage, suffix)))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.Subtitle.Render("Powered by SkyBase"), "", m.theme.PanelTitle.Render("NIGHTOPS SYSTEM INITIALIZATION"), "", lipgloss.JoinVertical(lipgloss.Left, checks...), "", m.theme.AccentStyle().Render("  "+bootStageLabel(m.bootStage)), "", m.theme.MutedStyle().Render("  Press Enter or Space to skip boot"))
	return m.center(content)
}

func bootStageLabel(stage int) string {
	if stage >= len(bootStages) {
		return bootStages[len(bootStages)-1]
	}
	return bootStages[stage] + "..."
}

func (m Model) renderLaunch() string {
	launch := m.theme.SelectedLaunchAction.Width(34).Render("▸  LAUNCH MISSION\n\n   Enter to begin mission prep")
	profile := nonEmpty(m.options.HomeBaseName, "HOME BASE NOT CONFIGURED")
	status := m.theme.MutedStyle().Render("ORIGIN  " + profile + "\n" + m.statusLine("DATABASE", m.options.DatabaseStatus) + "  " + m.statusLine("OBSIDIAN", m.options.ObsidianStatus) + "\n" + m.statusLine("WEATHER", m.options.WeatherStatus) + "  " + m.statusLine("ATLAS", m.options.AtlasStatus) + "\n" + m.statusLine("GPS", m.options.GPSStatus))
	content := lipgloss.JoinVertical(lipgloss.Center,
		m.theme.AccentStyle().Render(launchBackdrop(m.spinner.View())),
		m.theme.Wordmark.Render("N I G H T O P S"),
		m.theme.Subtitle.Render("Powered by SkyBase"),
		m.theme.Motto.Render("Every night under the stars is a mission worth remembering."),
		"", launch, "", status,
		"", m.theme.MutedStyle().Render("Enter Launch   s Settings   ? Help   q Quit"))
	return m.center(content)
}

func launchBackdrop(glyph string) string {
	if glyph == "" {
		glyph = "·"
	}
	return "              .        *             .\n        *             .       .             *\n   .         · · ·   NIGHT SKY   · · ·        .\n        .        *             .        *\n              .        " + glyph + "        ."
}

func (m Model) renderAction(action launchAction, selected bool) string {
	style := m.theme.Action
	if action.key == "launch" {
		style = m.theme.LaunchAction
	}
	if selected {
		if action.key == "launch" {
			style = m.theme.SelectedLaunchAction
		} else {
			style = m.theme.SelectedAction
		}
	}
	prefix := "  "
	if selected {
		prefix = "▸ "
	}
	return style.Width(m.panelWidth() - 4).Render(lipgloss.JoinVertical(lipgloss.Left, prefix+action.label, m.theme.MutedStyle().Render("  "+action.description)))
}

func (m Model) renderStatus() string {
	lines := []string{m.theme.PanelTitle.Render("SYSTEM STATUS"), "", m.statusLine("LOCAL DATABASE", m.options.DatabaseStatus), m.statusLine("OBSIDIAN EXPORT", m.options.ObsidianStatus), m.statusLine("SKYBASE ATLAS", m.options.AtlasStatus), m.statusLine("GPS", m.options.GPSStatus), m.statusLine("GEOCODING", m.options.GeocodingStatus), m.statusLine("WEATHER", m.options.WeatherStatus), m.statusLine("TELESCOPE", m.options.TelescopeStatus), m.statusLine("AI MISSION BRIEF", m.options.AIStatus), m.statusLine("LOCAL API", m.options.APIStatus), "", m.theme.MutedStyle().Render("PROFILE"), "  " + nonEmpty(m.options.HomeBaseName, "Not configured"), "", m.theme.MutedStyle().Render("VERSION"), "  " + nonEmpty(m.options.Version, "development")}
	return m.theme.Status.Width(m.statusWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) statusLine(label, status string) string {
	status = nonEmpty(status, "NOT CONFIGURED")
	style := m.theme.MutedStyle()
	glyph := "·"
	switch status {
	case "READY":
		style, glyph = m.theme.GoodStyle(), "✓"
	case "ERROR", "UNAVAILABLE":
		style, glyph = m.theme.ErrorStyle(), "!"
	case "STANDBY", "NOT CONFIGURED":
		style, glyph = m.theme.WarningStyle(), "◐"
	case "DISABLED":
		style, glyph = m.theme.MutedStyle(), "–"
	}
	return style.Render(glyph) + " " + label + "  " + style.Render(status)
}

func (m Model) renderHomeBaseSetup() string {
	form := lipgloss.JoinVertical(lipgloss.Left, m.theme.PanelTitle.Render("HOME BASE SETUP"), "", "Configure a reusable mission origin.", "", m.homeBase.name.View(), m.homeBase.zip.View(), "", m.theme.MutedStyle().Render("Tab/↑↓ switch field   Enter save   Esc cancel"))
	if m.homeBase.error != "" {
		form = lipgloss.JoinVertical(lipgloss.Left, form, "", m.theme.ErrorStyle().Render(m.homeBase.error))
	}
	return m.center(lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), form))
}

func (m Model) renderGPS() string {
	status := m.theme.AccentStyle().Render(m.spinner.View() + "  " + nonEmpty(m.gps.status, "ACQUIRING CURRENT LOCATION"))
	controls := "Esc cancel"
	if m.gps.status == "UNAVAILABLE" {
		status = m.theme.ErrorStyle().Render("!  GPS UNAVAILABLE")
		controls = "r Retry   z Enter ZIP Code Instead   Esc Back"
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("GPS ACQUISITION"), "", status, "", "No coordinates are shown until a real GPS adapter responds.", "", m.theme.MutedStyle().Render(controls))
	return m.center(content)
}

func (m Model) renderZIP() string {
	detail := "Coordinates will remain unknown until geocoding is configured."
	if m.options.ResolveZIP != nil {
		detail = "Coordinates are resolved from the configured provider when available; cached results work offline."
	}
	if m.zipEntry.pending {
		detail = m.theme.AccentStyle().Render("Resolving ZIP coordinates…")
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("ENTER MISSION ORIGIN"), "", detail, "", m.zipEntry.input.View(), "", m.theme.MutedStyle().Render("Enter continue   Esc back"))
	if m.zipEntry.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.zipEntry.error))
	}
	return m.center(content)
}

func (m Model) renderAtlas() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("SKYBASE ATLAS // LOCAL SITES"), "", "Select a catalog location for this mission."}
	for index, site := range m.options.AtlasLocations {
		style := m.theme.Action
		if index == m.atlas.selected {
			style = m.theme.SelectedAction
		}
		detail := site.Bortle
		if detail == "" {
			detail = "Bortle unknown"
		} else {
			detail = "Bortle " + detail
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(site.Name+" · "+detail))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Enter Select   Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderAtlasImport() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("SKYBASE ATLAS // IMPORT CATALOG"), "", "Import a validated CSV catalog from the local filesystem.", "No network access is used.", "", m.atlasImport.path.View(), m.atlasImport.version.View(), "", m.theme.MutedStyle().Render("Tab/↑↓ switch field   Enter import   Esc cancel"))
	if m.atlasImport.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.atlasImport.error))
	}
	return m.center(content)
}

func (m Model) renderAtlasExport() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("SKYBASE ATLAS // EXPORT CONTRIBUTION"), "", "Export the validated local catalog for review or community contribution.", "Source attribution is preserved. No network access is used.", "", m.atlasExport.path.View(), "", m.theme.MutedStyle().Render("Enter export CSV   Esc cancel"))
	if m.atlasExport.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.atlasExport.error))
	}
	return m.center(content)
}

func (m Model) renderTarget() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("TONIGHT'S TARGETS"), "", "Choose what you want to capture. NightOps will calculate each target's dark-sky window and rank the weather automatically.", m.theme.MutedStyle().Render("Space/Enter select   c build mission   Esc back"), ""}
	for index, target := range m.options.Targets {
		style := m.theme.Action
		if index == m.target.selected {
			style = m.theme.SelectedAction
		}
		check := "[ ]"
		if m.target.chosen[target.ID] {
			check = "[✓]"
		}
		detail := fmt.Sprintf("%s · RA %.3fh · Dec %.2f°", target.Kind, target.RightAscension, target.Declination)
		if m.options.TargetSummary != nil && m.missionPlan.origin.Latitude != nil && m.missionPlan.origin.Longitude != nil {
			detail += " · " + m.options.TargetSummary(m.missionPlan.origin, target)
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(check+" "+target.Name+"\n  "+m.theme.MutedStyle().Render(detail)))
	}
	lines = append(lines, "", m.theme.AccentStyle().Render(fmt.Sprintf("%d target(s) selected", len(m.selectedTargets()))), m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Space/Enter Toggle   c Build mission   Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderForecast() string {
	points := m.filteredForecastPoints()
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("WEATHER // HOURLY FORECAST"), "", "Select a forecast hour for this mission.", m.theme.MutedStyle().Render(m.forecastFilterLine())}
	for index, point := range points {
		style := m.theme.Action
		if index == m.missionPlan.forecastSelected {
			style = m.theme.SelectedAction
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(formatForecastPoint(point, m.options.ForecastTimezone)))
	}
	if len(points) == 0 {
		lines = append(lines, "", m.theme.WarningStyle().Render("No forecast hours match the active filters."))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render(fmt.Sprintf("d Darkness   c Cloud ≤%d%%   p Precip ≤%d%%   ↑/k ↓/j Navigate   Enter Select   Esc Back", m.forecastCloudMax(), m.forecastPrecipMax())))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderRoutePlan() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("ROUTE PLANNING"), "", "ROUTE FACTS", "", nonEmpty(m.missionPlan.route, "unavailable"), "", "Route facts may be cached locally for offline use.", "", m.theme.MutedStyle().Render("Enter Accept Route   Esc Back"))
	return m.center(content)
}

func (m Model) renderEquipmentSetup() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("EQUIPMENT // NEW PROFILE"), "", "Create a reusable observing setup.", "", m.equipmentSetup.name.View(), m.equipmentSetup.description.View(), "", m.theme.MutedStyle().Render("Tab/↑↓ switch field   Enter save   Esc cancel"))
	if m.equipmentSetup.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.equipmentSetup.error))
	}
	return m.center(content)
}

func (m Model) renderEquipmentBrowser() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("EQUIPMENT // SELECT PROFILE"), "", "Select the setup for this mission."}
	for index, profile := range m.options.EquipmentProfiles {
		style := m.theme.Action
		if index == m.equipment.selected {
			style = m.theme.SelectedAction
		}
		detail := profile.Name
		if profile.Description != "" {
			detail += " · " + profile.Description
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(detail))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Enter Select   Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderEquipmentInventory() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("EQUIPMENT // INVENTORY"), "", "Select a profile to view its recorded inventory."}
	if len(m.options.EquipmentProfiles) == 0 {
		lines = append(lines, "", m.theme.WarningStyle().Render("No equipment profiles configured. Create one in Settings first."))
	} else {
		profile := m.options.EquipmentProfiles[m.equipmentInventory.selected]
		lines = append(lines, "")
		for index, candidate := range m.options.EquipmentProfiles {
			style := m.theme.Action
			if index == m.equipmentInventory.selected {
				style = m.theme.SelectedAction
			}
			lines = append(lines, style.Width(m.panelWidth()-4).Render(candidate.Name))
		}
		lines = append(lines, "", m.theme.MutedStyle().Render("INVENTORY FOR "+profile.Name))
		items := m.equipmentItemsFor(profile.ID)
		if len(items) == 0 {
			lines = append(lines, m.theme.WarningStyle().Render("No inventory items recorded."))
		} else {
			for _, item := range items {
				kind := "OPTIONAL"
				if item.Required {
					kind = "REQUIRED"
				}
				lines = append(lines, "  "+item.Category+" · "+item.Name+" ["+kind+"]")
			}
		}
	}
	controls := "Esc Back"
	if len(m.options.EquipmentProfiles) > 0 {
		controls = "↑/k ↓/j Profile   n New Item   Enter Readiness   Esc Back"
	}
	lines = append(lines, "", m.theme.MutedStyle().Render(controls))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderEquipmentItemSetup() string {
	required := "REQUIRED"
	if !m.equipmentItemSetup.required {
		required = "OPTIONAL"
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("EQUIPMENT // NEW INVENTORY ITEM"), "", "Record an item already owned by this setup.", "", m.equipmentItemSetup.name.View(), m.equipmentItemSetup.category.View(), "  PRESENCE    RECORDED", "  IMPORTANCE  "+required, "", m.theme.MutedStyle().Render("Ctrl+r Toggle required/optional   Tab/↑↓ switch field   Enter save   Esc cancel"))
	if m.equipmentItemSetup.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.equipmentItemSetup.error))
	}
	return m.center(content)
}

func (m Model) renderEquipmentReadiness() string {
	profile := m.equipmentProfile(m.equipmentReadiness.profileID)
	items := m.equipmentItemsFor(m.equipmentReadiness.profileID)
	required := 0
	for _, item := range items {
		if item.Required {
			required++
		}
	}
	status := m.theme.WarningStyle().Render("NOT CONFIGURED")
	if required > 0 {
		status = m.theme.GoodStyle().Render("READY")
	}
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("EQUIPMENT // SESSION READINESS"), "", "PROFILE        " + nonEmpty(profile.Name, "Unknown profile"), "STATUS         " + status, fmt.Sprintf("REQUIRED ITEMS  %d recorded", required), "", "INVENTORY"}
	if len(items) == 0 {
		lines = append(lines, m.theme.WarningStyle().Render("No inventory items recorded; readiness cannot be confirmed."))
	} else {
		for _, item := range items {
			glyph := "·"
			if item.Required {
				glyph = "✓"
			}
			lines = append(lines, "  "+glyph+" "+item.Category+" · "+item.Name)
		}
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) equipmentProfile(id string) EquipmentProfile {
	for _, profile := range m.options.EquipmentProfiles {
		if profile.ID == id {
			return profile
		}
	}
	return EquipmentProfile{}
}

func (m Model) equipmentItemsFor(profileID string) []EquipmentItem {
	items := make([]EquipmentItem, 0)
	for _, item := range m.options.EquipmentItems {
		if item.ProfileID == profileID {
			items = append(items, item)
		}
	}
	return items
}

func (m Model) filteredForecastPoints() []ForecastPoint {
	cloudMax, precipMax := m.forecastCloudMax(), m.forecastPrecipMax()
	points := make([]ForecastPoint, 0, len(m.missionPlan.forecastPoints))
	for _, point := range m.missionPlan.forecastPoints {
		if m.missionPlan.darkOnly && !point.Dark {
			continue
		}
		if m.missionPlan.cloudOnly && (point.CloudCoverPercent == nil || *point.CloudCoverPercent > float64(cloudMax)) {
			continue
		}
		if m.missionPlan.precipOnly && (point.PrecipitationProbability == nil || *point.PrecipitationProbability > float64(precipMax)) {
			continue
		}
		points = append(points, point)
	}
	return points
}

func (m Model) forecastFilterLine() string {
	filters := make([]string, 0, 3)
	if m.missionPlan.darkOnly {
		filters = append(filters, "DARKNESS")
	}
	if m.missionPlan.cloudOnly {
		filters = append(filters, fmt.Sprintf("CLOUD ≤%d%%", m.forecastCloudMax()))
	}
	if m.missionPlan.precipOnly {
		filters = append(filters, fmt.Sprintf("PRECIP ≤%d%%", m.forecastPrecipMax()))
	}
	if len(filters) == 0 {
		return "FILTERS  ALL HOURS"
	}
	return "FILTERS  " + strings.Join(filters, " · ")
}

func (m Model) forecastCloudMax() int {
	if m.options.ForecastThresholdsConfigured && m.options.ForecastCloudMax >= 0 && m.options.ForecastCloudMax <= 100 {
		return m.options.ForecastCloudMax
	}
	return 50
}

func (m Model) forecastPrecipMax() int {
	if m.options.ForecastThresholdsConfigured && m.options.ForecastPrecipMax >= 0 && m.options.ForecastPrecipMax <= 100 {
		return m.options.ForecastPrecipMax
	}
	return 20
}

func (m Model) renderMissionPlan() string {
	lines := []string{m.theme.PanelTitle.Render("MISSION PLANNING // TONIGHT"), "", "NightOps fills the date, target windows, and weather from this run.", ""}
	origin := nonEmpty(m.missionPlan.origin.Label, "origin not selected")
	if m.missionPlan.origin.ZIP != "" {
		origin += " · ZIP " + m.missionPlan.origin.ZIP
	}
	if m.missionPlan.origin.Latitude != nil && m.missionPlan.origin.Longitude != nil {
		origin += fmt.Sprintf(" · %.4f, %.4f", *m.missionPlan.origin.Latitude, *m.missionPlan.origin.Longitude)
		lines = append(lines, "ORIGIN       "+m.theme.GoodStyle().Render("READY")+" · "+compactText(origin, 58))
	} else {
		lines = append(lines, "ORIGIN       "+m.theme.WarningStyle().Render("NEEDS COORDINATES")+" · "+compactText(origin, 44))
	}

	if m.missionPlan.plannedStart != nil && m.missionPlan.plannedEnd != nil {
		location := m.scheduleLocation()
		window := m.missionPlan.plannedStart.In(location).Format("15:04 MST") + " → " + m.missionPlan.plannedEnd.In(location).Format("15:04 MST")
		lines = append(lines, "MISSION WINDOW "+m.theme.GoodStyle().Render("TONIGHT")+" · "+window)
		if m.missionPlan.missionWindow != "" {
			lines = append(lines, "               "+compactText(m.missionPlan.missionWindow, 56))
		}
	} else {
		lines = append(lines, "MISSION WINDOW "+m.theme.WarningStyle().Render("WAITING")+" · select targets and a usable location")
	}
	weatherCheck := nonEmpty(m.missionPlan.weatherDecision, nonEmpty(m.missionPlan.weather, "not available"))
	lines = append(lines, "WEATHER      "+compactText(weatherCheck, 58))
	if m.missionPlan.origin.Latitude == nil || m.missionPlan.origin.Longitude == nil {
		lines = append(lines, m.theme.WarningStyle().Render("COORDINATES   unavailable until coordinates are known"))
	}
	if m.missionPlan.astronomy != "" {
		lines = append(lines, "ASTRONOMY    "+compactText(m.missionPlan.astronomy, 58))
	}
	if m.missionPlan.forecast != "" {
		lines = append(lines, "FORECAST     "+compactText(m.missionPlan.forecast, 58))
	}
	if m.missionPlan.route != "" && m.missionPlan.route != "unavailable" {
		lines = append(lines, "ROUTE        "+compactText(m.missionPlan.route, 58))
	}

	if len(m.missionPlan.targets) == 0 {
		lines = append(lines, "TARGETS      "+m.theme.WarningStyle().Render("SELECT TARGETS"))
	} else {
		lines = append(lines, fmt.Sprintf("TARGETS      %s · %d selected", m.theme.GoodStyle().Render("READY"), len(m.missionPlan.targets)))
		for position, target := range m.missionPlan.targets {
			detail := target.Kind
			if m.options.TargetSummary != nil {
				detail = compactText(m.options.TargetSummary(m.missionPlan.origin, target), 44)
			}
			lines = append(lines, fmt.Sprintf("             %d. %-24s %s", position+1, compactText(target.Name, 24), target.Kind))
			if m.options.TargetSummary != nil {
				lines = append(lines, "                  WINDOW "+compactText(detail, 54))
			}
			if m.options.TargetForecastSummary != nil {
				lines = append(lines, "                  WEATHER "+compactText(m.options.TargetForecastSummary(m.missionPlan.origin, target, m.missionPlan.forecastPoints), 50))
			}
		}
	}
	equipment := nonEmpty(m.missionPlan.equipment, "not selected")
	lines = append(lines, "EQUIPMENT    "+compactText(equipment, 58))
	if m.missionPlan.briefPending {
		lines = append(lines, m.theme.WarningStyle().Render("BRIEF        generating…"))
	}
	lines = append(lines, "", m.theme.PanelTitle.Render("NEXT STEP"))
	actions := m.missionActions()
	for index, action := range actions {
		style := m.theme.Action
		if index == m.missionPlan.selected {
			style = m.theme.SelectedAction
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(action))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render(compactText(m.missionPlan.status, 70)), "", m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Enter Select   Esc Back"))
	wordmark := m.theme.Wordmark.Width(m.panelWidth()).Render("N I G H T O P S")
	return m.center(lipgloss.JoinVertical(lipgloss.Left, wordmark, m.theme.Panel.Width(m.panelWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))))
}

func (m Model) renderMissionReview() string {
	status := func(ready bool, value, fallback string) string {
		if ready {
			return m.theme.GoodStyle().Render("READY") + " · " + value
		}
		return m.theme.MutedStyle().Render("NOT SET") + " · " + fallback
	}
	window := "No observing window; launch can still be recorded offline."
	if m.missionPlan.plannedStart != nil && m.missionPlan.plannedEnd != nil {
		location := m.scheduleLocation()
		window = m.missionPlan.plannedStart.In(location).Format("2006-01-02 15:04 MST") + " → " + m.missionPlan.plannedEnd.In(location).Format("2006-01-02 15:04 MST")
	}
	targets := "No targets selected; observations can be entered manually."
	if len(m.missionPlan.targets) > 0 {
		ordered := make([]string, 0, len(m.missionPlan.targets))
		for index, target := range m.missionPlan.targets {
			ordered = append(ordered, fmt.Sprintf("%d. %s", index+1, target.Name))
		}
		targets = strings.Join(ordered, "   ")
	}
	equipment := "No equipment profile selected."
	if m.missionPlan.equipment != "" {
		equipment = m.missionPlan.equipment + " · " + m.equipmentReadinessLine(m.missionPlan.equipmentID)
	}
	lines := []string{
		m.wordmark(),
		m.theme.PanelTitle.Render("MISSION // FINAL REVIEW"),
		"",
		"Confirm the recorded inputs before creating the local mission.",
		"",
		"ORIGIN          " + status(m.missionPlan.origin.Label != "", m.missionPlan.origin.Label, "origin required"),
		"TARGET SEQUENCE " + status(len(m.missionPlan.targets) > 0, targets, targets),
		"OBSERVING WINDOW (TONIGHT) " + status(m.missionPlan.plannedStart != nil && m.missionPlan.plannedEnd != nil, window, window),
		"EQUIPMENT       " + status(m.missionPlan.equipmentID != "", equipment, equipment),
		"ASTRONOMY       " + nonEmpty(m.missionPlan.astronomy, "unknown until coordinates are available"),
		"WEATHER         " + nonEmpty(m.missionPlan.weatherDecision, nonEmpty(m.missionPlan.weather, "unavailable")),
		"",
	}
	for index, action := range []string{"LAUNCH + OPEN OBSIDIAN", "LAUNCH + CONTINUE IN NIGHTOPS", "RETURN TO MISSION PLANNING"} {
		style := m.theme.Action
		if index == m.missionReview.selected {
			style = m.theme.SelectedAction
		}
		lines = append(lines, style.Width(m.panelWidth()-4).Render(action))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Enter Select   Esc Back"))
	return m.center(m.theme.Panel.Width(m.panelWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (m *Model) updateDeepSpace(key string) tea.Cmd {
	if key == "esc" {
		m.route = RouteOperation
		return nil
	}
	if key == "o" {
		m.route = RouteOperation
		return nil
	}
	if len(m.missionPlan.targets) == 0 {
		if key == "a" {
			m.previousRoute, m.route = RouteDeepSpace, RouteObservationEntry
			m.observation.target.Focus()
			return textinput.Blink
		}
		return nil
	}
	switch key {
	case "up", "k":
		if m.deepSpaceSelected > 0 {
			m.deepSpaceSelected--
		}
	case "down", "j":
		if m.deepSpaceSelected < len(m.missionPlan.targets)-1 {
			m.deepSpaceSelected++
		}
	case "enter":
		if m.options.SlewToTarget == nil && m.options.SlewToTargetAt == nil {
			m.missionPlan.status = "Telescope control is not configured; record an observation with a."
			return nil
		}
		target := m.missionPlan.targets[m.deepSpaceSelected]
		if m.options.SlewToTargetAt != nil && m.missionPlan.origin.Latitude != nil && m.missionPlan.origin.Longitude != nil {
			if err := m.options.SlewToTargetAt(m.missionPlan.origin, target); err != nil {
				m.error = ErrorModel{title: "TELESCOPE SLEW FAILED", detail: err.Error(), context: "TELESCOPE"}
				m.route = RouteError
				return nil
			}
			m.missionPlan.status = "Slew command accepted for " + target.Name + "."
		} else if m.options.SlewToTarget != nil {
			if err := m.options.SlewToTarget(target); err != nil {
				m.error = ErrorModel{title: "TELESCOPE SLEW FAILED", detail: err.Error(), context: "TELESCOPE"}
				m.route = RouteError
				return nil
			}
			m.missionPlan.status = "Slew command accepted for " + target.Name + "."
		}
	case "a":
		m.previousRoute, m.route = RouteDeepSpace, RouteObservationEntry
		m.observation.target.Focus()
		return textinput.Blink
	case "o":
		m.route = RouteOperation
	}
	return nil
}

func (m Model) renderBlastOff() string {
	count := fmt.Sprintf("%02d", m.blastOff)
	lines := []string{m.wordmark(), m.theme.GoodStyle().Render("MISSION LAUNCH SEQUENCE"), "", m.theme.GoodStyle().Render("╔══════════════════════════════╗"), m.theme.GoodStyle().Render("║          T - " + count + "             ║"), m.theme.GoodStyle().Render("╚══════════════════════════════╝"), "", "ORIGIN          " + m.missionPlan.origin.Label, fmt.Sprintf("TARGETS         %d queued", len(m.missionPlan.targets)), "", m.theme.MutedStyle().Render("Press Esc to abort launch")}
	return m.center(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

func (m Model) renderDeepSpace() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("DEEP SPACE // ACTIVE MISSION"), "", "ORIGIN  " + m.missionPlan.origin.Label, "WEATHER " + nonEmpty(m.missionPlan.weather, "unknown"), "", m.theme.GoodStyle().Render("MISSION TELEMETRY ONLINE"), ""}
	if m.missionPlan.origin.Latitude == nil || m.missionPlan.origin.Longitude == nil {
		lines = append(lines, m.theme.WarningStyle().Render("POSITION SOURCE  ZIP / coordinates unknown"))
	} else {
		lines = append(lines, fmt.Sprintf("POSITION SOURCE  %.4f, %.4f", *m.missionPlan.origin.Latitude, *m.missionPlan.origin.Longitude))
	}
	for index, target := range m.missionPlan.targets {
		marker := "  "
		if index == m.deepSpaceSelected {
			marker = "▸ "
		}
		lines = append(lines, "", marker+target.Name, targetArt(target.Kind), m.theme.MutedStyle().Render(fmt.Sprintf("RA %.3fh  DEC %.2f°  ·  %s", target.RightAscension, target.Declination, target.Source)))
	}
	if len(m.missionPlan.targets) == 0 {
		lines = append(lines, m.theme.WarningStyle().Render("No catalog targets were scheduled for this mission."))
	}
	slewAction := "Enter Slew"
	if m.options.SlewToTarget == nil && m.options.SlewToTargetAt == nil {
		slewAction = "Telescope control unavailable"
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Target   "+slewAction+"   a Record Observation   o Operation   Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func targetArt(kind string) string {
	switch strings.ToLower(kind) {
	case "galaxy":
		return "        .  *      .     (  @  )     .      *  .\n             .      .---===---.      ."
	case "nebula":
		return "             .  ~~~  *  ~~~  .\n          ~~~      < * >      ~~~"
	case "cluster":
		return "          *  .  *   .  *  .  *\n             .  *  .  *  ."
	default:
		return "              .      *      .\n                 ·  ·  ·"
	}
}

func (m Model) renderMissionSchedule() string {
	timezone := m.missionPlan.origin.Timezone
	if timezone == "" {
		timezone = m.options.ScheduleTimezone
	}
	if timezone == "" {
		timezone = "UTC"
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("MISSION // OBSERVING WINDOW"), "", "Enter local times for "+timezone+".", "No forecast values are invented from this window.", "", m.schedule.start.View(), m.schedule.end.View(), "", m.theme.MutedStyle().Render("Format YYYY-MM-DD HH:MM   Tab/↑↓ switch field   Enter save   Esc cancel"))
	if m.schedule.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.schedule.error))
	}
	return m.center(content)
}

func (m Model) renderBackup() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("DATABASE // BACKUP"), "", "Create a consistent local SQLite backup.", "The active database remains untouched.", "", m.backup.path.View(), "", m.theme.MutedStyle().Render("Enter create backup   Esc cancel"))
	if m.backup.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.backup.error))
	}
	return m.center(content)
}

func (m Model) renderPlugins() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("NIGHTOPS // PLUGIN REGISTRY"), "", "Local manifest metadata only. Plugin code is not executed.", "DIRECTORY      " + nonEmpty(m.options.PluginsDir, "not configured"), "STATUS         " + nonEmpty(m.options.PluginsStatus, "NOT CONFIGURED"), ""}
	if len(m.options.Plugins) == 0 {
		lines = append(lines, m.theme.WarningStyle().Render("No validated plugin manifests discovered."))
	} else {
		for index, plugin := range m.options.Plugins {
			style := m.theme.Action
			if index == m.plugins.selected {
				style = m.theme.SelectedAction
			}
			lines = append(lines, style.Width(m.panelWidth()-4).Render(plugin.Name+" · "+plugin.Version))
		}
		plugin := m.options.Plugins[m.plugins.selected]
		lines = append(lines, "", "ID             "+plugin.ID, "ENTRYPOINT     "+plugin.Entrypoint, "DESCRIPTION    "+nonEmpty(plugin.Description, "none declared"), "CAPABILITIES   "+nonEmpty(strings.Join(plugin.Capabilities, ", "), "none declared"), "", m.theme.MutedStyle().Render("↑/k ↓/j Select   Esc Back"))
	}
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) slewToSelectedTarget() tea.Cmd {
	if m.options.SlewToTarget == nil {
		m.error = ErrorModel{title: "TELESCOPE UNAVAILABLE", detail: "No configured telescope adapter is available for this installation.", context: "TELESCOPE"}
		m.route = RouteError
		return nil
	}
	var target TargetSite
	for _, candidate := range m.options.Targets {
		if candidate.Name == m.missionPlan.target {
			target = candidate
			break
		}
	}
	if target.Name == "" {
		m.error = ErrorModel{title: "TARGET UNAVAILABLE", detail: "Select a celestial target before sending a telescope command.", context: "TELESCOPE"}
		m.route = RouteError
		return nil
	}
	var err error
	if m.options.SlewToTargetAt != nil {
		err = m.options.SlewToTargetAt(m.missionPlan.origin, target)
	} else {
		err = m.options.SlewToTarget(target)
	}
	if err != nil {
		m.error = ErrorModel{title: "TELESCOPE SLEW FAILED", detail: err.Error(), context: "TELESCOPE"}
		m.route = RouteError
		return nil
	}
	m.missionPlan.status = "Slew command accepted by " + nonEmpty(m.options.TelescopeName, "the configured telescope") + "."
	return nil
}

func (m *Model) requestMissionBrief() tea.Cmd {
	if m.options.GenerateMissionBrief == nil {
		m.error = ErrorModel{title: "AI BRIEF UNAVAILABLE", detail: "No local AI provider is configured for this installation.", context: "AI"}
		m.route = RouteError
		return nil
	}
	input := MissionBriefInput{Origin: m.missionPlan.origin.Kind + " · " + m.missionPlan.origin.Label, Target: m.missionPlan.target, TargetWindow: m.missionPlan.targetInfo, Conditions: m.missionPlan.weather, Forecast: m.missionPlan.forecast, Astronomy: m.missionPlan.astronomy, Route: m.missionPlan.route, Equipment: m.missionPlan.equipment}
	m.missionPlan.briefPending = true
	m.missionPlan.status = "Generating mission brief from the configured local provider..."
	return func() tea.Msg {
		brief, err := m.options.GenerateMissionBrief(input)
		return missionBriefResultMsg{brief: brief, err: err}
	}
}

func (m Model) missionActions() []string {
	actions := []string{"REVIEW & CREATE OBSIDIAN MISSION", "CHANGE ORIGIN", "SAVE AS HOME BASE", "BACK TO LAUNCH"}
	if m.options.TargetsEnabled && len(m.options.Targets) > 0 {
		if len(m.missionPlan.targets) > 0 {
			actions = append(actions, "CHANGE TARGETS")
		} else {
			actions = append(actions, "SELECT TARGETS")
		}
	}
	if m.options.ForecastPoints != nil && len(m.missionPlan.forecastPoints) > 0 {
		actions = append(actions, "BROWSE HOURLY FORECAST")
	}
	if m.options.RouteEnabled != nil && m.options.RouteEnabled(m.missionPlan.origin) {
		actions = append(actions, "VIEW ROUTE PLAN")
	}
	if len(m.options.EquipmentProfiles) > 0 {
		actions = append(actions, "SELECT EQUIPMENT")
	}
	if m.missionPlan.equipmentID != "" {
		actions = append(actions, "CHECK EQUIPMENT READINESS")
	}
	if m.missionPlan.plannedStart == nil || m.missionPlan.plannedEnd == nil {
		actions = append(actions, "OVERRIDE AUTO WINDOW (ADVANCED)")
	}
	if (m.options.SlewToTarget != nil || m.options.SlewToTargetAt != nil) && m.missionPlan.target != "" {
		actions = append(actions, "SLEW TELESCOPE TO TARGET")
	}
	if m.options.GenerateMissionBrief != nil {
		if m.missionPlan.brief != "" {
			actions = append(actions, "REFRESH MISSION BRIEF")
		} else {
			actions = append(actions, "GENERATE MISSION BRIEF")
		}
	}
	return actions
}

func (m Model) targetActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "BUILD TARGET SEQUENCE" || action == "SELECT TARGETS" || action == "CHANGE TARGETS" {
			return index
		}
	}
	return -1
}

func (m Model) forecastActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "BROWSE HOURLY FORECAST" {
			return index
		}
	}
	return -1
}

func (m Model) routeActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "VIEW ROUTE PLAN" {
			return index
		}
	}
	return -1
}

func (m Model) equipmentActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "SELECT EQUIPMENT" {
			return index
		}
	}
	return -1
}

func (m Model) readinessActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "CHECK EQUIPMENT READINESS" {
			return index
		}
	}
	return -1
}

func (m Model) scheduleActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "SET MISSION WINDOW" || action == "EDIT MISSION WINDOW" || action == "OVERRIDE AUTO WINDOW (ADVANCED)" {
			return index
		}
	}
	return -1
}

func (m Model) telescopeActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "SLEW TELESCOPE TO TARGET" {
			return index
		}
	}
	return -1
}

func (m Model) aiBriefActionIndex() int {
	for index, action := range m.missionActions() {
		if action == "GENERATE MISSION BRIEF" || action == "REFRESH MISSION BRIEF" {
			return index
		}
	}
	return -1
}

func formatForecastPoint(point ForecastPoint, timezone string) string {
	at := point.At
	if timezone != "" {
		if location, err := time.LoadLocation(timezone); err == nil {
			at = at.In(location)
		}
	}
	value := at.Format("15:04 MST")
	if point.TemperatureC != nil {
		value += fmt.Sprintf(" %.1f°C", *point.TemperatureC)
	}
	if point.CloudCoverPercent != nil {
		value += fmt.Sprintf(" clouds %.0f%%", *point.CloudCoverPercent)
	}
	if point.PrecipitationProbability != nil {
		value += fmt.Sprintf(" precip %.0f%%", *point.PrecipitationProbability)
	}
	return value
}

func (m Model) renderOperation() string {
	actions := "Enter launch operation"
	if m.operation.status == "LAUNCHED" {
		actions = "Enter begin observing   a log observation"
	}
	if m.operation.status == "ACTIVE" {
		actions = "a log observation   c complete operation"
	}
	lines := []string{m.theme.PanelTitle.Render("OPERATION // " + shortID(m.operation.missionID)), "", "STATUS          " + m.operation.status, "OBSERVATIONS    " + fmt.Sprint(len(m.operation.observations)), ""}
	if len(m.operation.observations) > 0 {
		lines = append(lines, "RECORDED TARGETS")
		for _, target := range m.operation.observations {
			lines = append(lines, "  "+target)
		}
		lines = append(lines, "")
	}
	lines = append(lines, m.theme.MutedStyle().Render(actions), m.theme.MutedStyle().Render("Esc Back   q Quit"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.Panel.Width(m.panelWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))))
}

func (m Model) renderObservationEntry() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("FLIGHT RECORDER // OBSERVATION"), "", m.observation.target.View(), m.observation.notes.View(), "", m.theme.MutedStyle().Render("Tab/↑↓ switch field   Enter record   Esc cancel")}
	if m.observation.error != "" {
		lines = append(lines, "", m.theme.ErrorStyle().Render(m.observation.error))
	}
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderDebrief() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("MISSION DEBRIEF"), "", "MISSION         " + shortID(m.debrief.missionID), "STATUS          COMPLETED", "OBSERVATIONS    " + fmt.Sprint(m.debrief.count), ""}
	if m.debrief.saved {
		lines = append(lines, m.theme.GoodStyle().Render("Debrief saved locally and projected to Obsidian when configured."), "", m.theme.MutedStyle().Render("Enter or b Back to Launch"))
	} else {
		lines = append(lines, "Record what you learned from this operation.", "", m.debrief.summary.View(), "", m.theme.MutedStyle().Render("Enter save debrief   b/Esc Back to Launch"))
	}
	if m.debrief.error != "" {
		lines = append(lines, "", m.theme.ErrorStyle().Render(m.debrief.error))
	}
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func (m Model) renderSettings() string {
	controls := "h Home Base Setup   e New Equipment Profile"
	if m.options.OpenObsidianVault != nil {
		controls += "   o Open Obsidian Vault"
	}
	if m.options.CreateBackup != nil {
		controls += "   b Create Database Backup"
	}
	if m.options.SaveEquipmentItem != nil {
		controls += "   v Equipment Inventory"
	}
	if m.options.SaveAtlas != nil && m.options.AtlasEnabled {
		controls += "   i Import Atlas CSV"
	}
	if m.options.ExportAtlas != nil && m.options.AtlasEnabled && m.options.AtlasStatus == "READY" {
		controls += "   x Export Atlas CSV"
	}
	if m.options.ExportSync != nil {
		controls += "   y Export Sync Bundle"
	}
	if m.options.ImportSync != nil {
		controls += "   u Import Sync Bundle"
	}
	if m.options.PluginsStatus != "DISABLED" {
		controls += "   p Plugin Registry"
	}
	atlasExportState := "NOT CONFIGURED"
	if m.options.ExportAtlas != nil && m.options.AtlasEnabled && m.options.AtlasStatus == "READY" {
		atlasExportState = "READY"
	}
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("SETTINGS // SYSTEM CONFIGURATION"), "", "HOME BASE        " + m.homeBaseLine(), "OBSIDIAN VAULT   " + nonEmpty(m.options.ObsidianStatus, "NOT CONFIGURED"), "EQUIPMENT       " + fmt.Sprintf("%d profile(s)", len(m.options.EquipmentProfiles)), "THEME            " + nonEmpty(m.options.ThemeName, "MISSION CONTROL"), "ATLAS            " + m.options.AtlasStatus, "GEOCODING       " + nonEmpty(m.options.GeocodingStatus, "NOT CONFIGURED"), "PLUGINS         " + nonEmpty(m.options.PluginsStatus, "NOT CONFIGURED"), "TELESCOPE       " + nonEmpty(m.options.TelescopeStatus, "NOT CONFIGURED"), "AI BRIEF        " + nonEmpty(m.options.AIStatus, "NOT CONFIGURED"), "LOCAL API       " + nonEmpty(m.options.APIStatus, "NOT CONFIGURED"), "BACKUP          " + nonEmpty(m.options.BackupStatus, "NOT CONFIGURED"), "ATLAS EXPORT    " + atlasExportState, "SYNC            " + syncStatus(m.options.ExportSync, m.options.ImportSync)}
	if m.settings.notice != "" {
		lines = append(lines, "", m.theme.GoodStyle().Render(m.settings.notice))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render(controls+"   ? Help   Esc Back   q Quit"))
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.center(content)
}

func (m Model) equipmentReadinessLine(profileID string) string {
	items := m.equipmentItemsFor(profileID)
	required := 0
	for _, item := range items {
		if item.Required {
			required++
		}
	}
	if required == 0 {
		return "NOT CONFIGURED"
	}
	return fmt.Sprintf("READY · %d required item(s)", required)
}

func (m Model) renderHelp() string {
	palette := ""
	if m.options.CommandPaletteEnabled {
		palette = "Ctrl+K          Open command palette"
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("NIGHTOPS // FLIGHT CONTROLS"), "", "↑/k and ↓/j   Move focus", "Enter          Select or launch", "s               Open settings", palette, "?               Open or close help", "Esc             Return", "q               Quit", "", "The selected launch action becomes the origin for Mission Planning.", "", m.theme.MutedStyle().Render("Screen: "+m.previousRoute.String()))
	return m.center(content)
}

func (m Model) renderCommandPalette() string {
	commands := m.filteredPaletteCommands()
	rows := make([]string, 0, len(commands))
	for index, command := range commands {
		style := m.theme.Action
		prefix := "  "
		if index == m.palette.selected {
			style = m.theme.SelectedAction
			prefix = "▸ "
		}
		rows = append(rows, style.Width(m.panelWidth()-4).Render(prefix+command.Label+"  "+m.theme.MutedStyle().Render(command.Key)))
	}
	if len(rows) == 0 {
		rows = append(rows, m.theme.MutedStyle().Render("No matching local commands."))
	}
	selected := ""
	if len(commands) > 0 {
		selected = commands[m.palette.selected].Description
	}
	lines := []string{m.theme.PanelTitle.Render("NIGHTOPS // COMMAND PALETTE"), "", m.theme.MutedStyle().Render("Type to filter commands available in this console."), "", m.palette.input.View(), "", strings.Join(rows, "\n")}
	if selected != "" {
		lines = append(lines, "", m.theme.MutedStyle().Render(selected))
	}
	if m.palette.error != "" {
		lines = append(lines, "", m.theme.ErrorStyle().Render(m.palette.error))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Navigate   Enter Run   Esc Close"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.Panel.Width(m.panelWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))))
}

func (m Model) renderMissionArchive() string {
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("MISSIONS // OFFLINE ARCHIVE"), "", "Locally persisted missions remain available without network access."}
	if len(m.missionArchive.missions) == 0 {
		lines = append(lines, "", m.theme.WarningStyle().Render("No missions have been recorded yet."))
	} else {
		for index, mission := range m.missionArchive.missions {
			style := m.theme.Action
			if index == m.missionArchive.selected {
				style = m.theme.SelectedAction
			}
			detail := strings.ToUpper(nonEmpty(mission.Status, "unknown")) + " · " + nonEmpty(mission.LaunchSiteName, "site unknown")
			if mission.PlannedStart != nil {
				detail += " · " + formatArchiveTime(*mission.PlannedStart, mission.Timezone)
			}
			lines = append(lines, style.Width(m.panelWidth()-4).Render(mission.Name+"\n"+m.theme.MutedStyle().Render(detail)))
		}
	}
	if m.missionArchive.error != "" {
		lines = append(lines, "", m.theme.ErrorStyle().Render(m.missionArchive.error))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("↑/k ↓/j Select   Enter Details   Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderMissionDetail() string {
	mission := m.missionDetail
	lines := []string{m.wordmark(), m.theme.PanelTitle.Render("MISSION // ARCHIVE DETAIL"), "", "NAME            " + mission.Name, "STATUS          " + strings.ToUpper(nonEmpty(mission.Status, "unknown")), "MISSION ID      " + mission.ID, "LAUNCH SITE     " + nonEmpty(mission.LaunchSiteName, "unknown"), "EQUIPMENT       " + nonEmpty(mission.EquipmentProfileID, "none selected"), "CREATED         " + formatArchiveTime(mission.CreatedAt, mission.Timezone)}
	if mission.PlannedStart != nil && mission.PlannedEnd != nil {
		lines = append(lines, "MISSION WINDOW  "+formatArchiveTime(*mission.PlannedStart, mission.Timezone)+" → "+formatArchiveTime(*mission.PlannedEnd, mission.Timezone))
	} else {
		lines = append(lines, m.theme.WarningStyle().Render("MISSION WINDOW  not scheduled"))
	}
	lines = append(lines, "", m.theme.MutedStyle().Render("Mission records are owned by SQLite and projected to Obsidian when configured."), "", m.theme.MutedStyle().Render("b/Esc Back"))
	return m.center(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderSyncExport() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("SYNC // EXPORT BUNDLE"), "", "Create a versioned offline bundle for another NightOps client.", "Stable IDs and update timestamps are preserved.", "", m.syncExport.path.View(), "", m.theme.MutedStyle().Render("Enter export bundle   Esc cancel"))
	if m.syncExport.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.syncExport.error))
	}
	return m.center(content)
}

func (m Model) renderSyncImport() string {
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.PanelTitle.Render("SYNC // IMPORT BUNDLE"), "", "Merge a local bundle using stable IDs and newer-record-wins semantics.", "Older incoming records are skipped; nothing is deleted.", "", m.syncImport.path.View(), "", m.theme.MutedStyle().Render("Enter merge bundle   Esc cancel"))
	if m.syncImport.error != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", m.theme.ErrorStyle().Render(m.syncImport.error))
	}
	return m.center(content)
}

func formatArchiveTime(value time.Time, timezone string) string {
	if value.IsZero() {
		return "unknown"
	}
	if timezone != "" {
		if location, err := time.LoadLocation(timezone); err == nil {
			value = value.In(location)
		}
	}
	return value.Format("2006-01-02 15:04 MST")
}

func (m Model) renderError() string {
	controls := "Esc Back"
	if m.error.context == "GPS" {
		controls = "r Retry   z Enter ZIP Code Instead   Esc Back"
	}
	content := lipgloss.JoinVertical(lipgloss.Left, m.wordmark(), m.theme.ErrorStyle().Render(m.error.title), "", m.error.detail, "", controls)
	return m.center(content)
}

func (m Model) renderFooter() string {
	footer := "↑/k ↓/j Navigate   Enter Launch   s Settings   ? Help"
	if m.options.CommandPaletteEnabled {
		footer += "   Ctrl+K Palette"
	}
	return m.theme.Footer.Render(footer + "   q Quit")
}

func (m Model) wordmark() string {
	return m.theme.Wordmark.Render("N I G H T O P S")
}

func (m Model) center(content string) string {
	if m.height < 24 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) panelWidth() int {
	if m.width < 90 {
		return max(46, m.width-4)
	}
	return 58
}

func (m Model) statusWidth() int {
	if m.width >= 110 {
		return 31
	}
	return m.panelWidth()
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func compactText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit < 4 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "…"
}

func exportStatus(exporter func(string) error) string {
	if exporter == nil {
		return "NOT CONFIGURED"
	}
	return "READY"
}

func syncStatus(exporter func(string) error, importer func(string) (string, error)) string {
	if exporter == nil || importer == nil {
		return "NOT CONFIGURED"
	}
	return "READY"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

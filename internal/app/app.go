// Package app composes the NightOps runtime.
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/jordanistan/nightops/internal/ai"
	"github.com/jordanistan/nightops/internal/api"
	"github.com/jordanistan/nightops/internal/application"
	"github.com/jordanistan/nightops/internal/astronomy"
	"github.com/jordanistan/nightops/internal/atlas"
	"github.com/jordanistan/nightops/internal/config"
	"github.com/jordanistan/nightops/internal/console"
	"github.com/jordanistan/nightops/internal/domain"
	"github.com/jordanistan/nightops/internal/export/obsidian"
	"github.com/jordanistan/nightops/internal/geocoding"
	"github.com/jordanistan/nightops/internal/plugins"
	"github.com/jordanistan/nightops/internal/providers"
	"github.com/jordanistan/nightops/internal/routing"
	"github.com/jordanistan/nightops/internal/storage/sqlite"
	syncbundle "github.com/jordanistan/nightops/internal/sync"
	"github.com/jordanistan/nightops/internal/targets"
	"github.com/jordanistan/nightops/internal/telescope"
	"github.com/jordanistan/nightops/internal/weather"
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
	deviceID, err := syncbundle.LoadOrCreateDeviceID(filepath.Join(dataDir, "sync", "device-id"))
	if err != nil {
		return fmt.Errorf("initialize sync device identity: %w", err)
	}
	log.Info("local database ready", "path", filepath.Join(dataDir, "nightops.db"))
	var exporter application.MissionExporter
	obsidianRoot := ""
	obsidianNotes := ""
	if cfg.Obsidian.Enabled && cfg.Obsidian.VaultDir != "" {
		var obsidianErr error
		obsidianRoot, obsidianNotes, obsidianErr = prepareObsidianWorkspace(cfg)
		if obsidianErr != nil {
			log.Warn("prepare Obsidian workspace", "error", obsidianErr)
		} else {
			exporter = obsidian.New(obsidianRoot, cfg.Obsidian.NotesDir)
		}
	}
	var knowledgeProvider application.TargetKnowledgeProvider
	if cfg.Features.TargetKnowledge {
		knowledgeProvider = providers.NewWikipediaProvider(cfg.TargetKnowledge.Endpoint, time.Duration(cfg.TargetKnowledge.TimeoutSeconds)*time.Second)
	}
	planner := application.MissionPlanner{Sites: store, Missions: store, MissionTargets: store, TargetKnowledge: store, KnowledgeProvider: knowledgeProvider, Weather: store, Observations: store, Debriefs: store, Equipment: store, Exporter: exporter, Clock: application.SystemClock{}}
	equipmentProfiles, equipmentErr := store.ListEquipmentProfiles(ctx)
	if equipmentErr != nil {
		log.Error("load equipment profiles", "error", equipmentErr)
		equipmentProfiles = nil
	}
	equipmentItems := make([]domain.EquipmentItem, 0)
	for _, profile := range equipmentProfiles {
		items, itemErr := store.ListEquipmentItems(ctx, profile.ID)
		if itemErr != nil {
			log.Error("load equipment items", "profile", profile.ID, "error", itemErr)
			continue
		}
		equipmentItems = append(equipmentItems, items...)
	}
	atlasLocations := make([]console.AtlasSite, 0)
	var activeAtlas atlas.Catalog
	atlasState := "DISABLED"
	if cfg.Features.Atlas {
		catalog, catalogErr := store.LoadActiveAtlasCatalog(ctx)
		if catalogErr != nil {
			catalog, catalogErr = atlas.LoadEmbeddedAustin()
		}
		if catalogErr != nil {
			atlasState = "ERROR"
			log.Error("load SkyBase Atlas", "error", catalogErr)
		} else {
			atlasState = "READY"
			activeAtlas = catalog
			atlasLocations = projectAtlas(catalog)
		}
	}
	targetSites := make([]console.TargetSite, 0)
	targetsEnabled := false
	if catalog, catalogErr := targets.LoadEmbedded(); catalogErr != nil {
		log.Error("load celestial target catalog", "error", catalogErr)
	} else {
		targetsEnabled = true
		for _, target := range catalog.Targets {
			targetSites = append(targetSites, console.TargetSite{ID: target.ID, Name: target.Name, Kind: target.Kind, RightAscension: target.RightAscension, Declination: target.Declination, Source: target.Source})
		}
	}
	warmTargetKnowledge(ctx, store, knowledgeProvider, targetSites)
	pluginManifests := make([]console.PluginManifest, 0)
	pluginStatus := "DISABLED"
	pluginDir := cfg.Plugins.Dir
	if pluginDir == "" {
		pluginDir = "~/.local/share/nightops/plugins"
	}
	if cfg.Features.Plugins {
		loaded, pluginErr := plugins.LoadDirectory(config.ExpandPath(pluginDir))
		if pluginErr != nil {
			if os.IsNotExist(pluginErr) {
				pluginStatus = "NOT CONFIGURED"
			} else {
				pluginStatus = "ERROR"
				log.Error("load plugin manifests", "error", pluginErr)
			}
		} else {
			pluginStatus = "READY"
			pluginManifests = projectPlugins(loaded)
		}
	}
	location := "Not configured"
	if cfg.Origin.Latitude != 0 || cfg.Origin.Longitude != 0 {
		location = fmt.Sprintf("%.4f, %.4f", cfg.Origin.Latitude, cfg.Origin.Longitude)
	}
	var weatherProvider weather.Provider
	if cfg.Features.Weather && cfg.Weather.Provider == "open-meteo" && cfg.Weather.Endpoint != "" {
		weatherProvider = providers.NewOpenMeteoProvider(cfg.Weather.Endpoint, cfg.Weather.CacheMinutes)
	}
	var routeProvider routing.Provider
	if cfg.Features.Routing && cfg.Routing.Provider == "osrm" && cfg.Routing.Endpoint != "" {
		routeProvider = providers.NewOSRMProviderWithPolicy(cfg.Routing.Endpoint, time.Duration(cfg.Routing.TimeoutSeconds)*time.Second, cfg.Routing.MaxRetries, time.Duration(cfg.Routing.RetryBackoffMillis)*time.Millisecond)
	}
	var resolveZIP func(string) (console.Origin, error)
	geocodingStatus := "DISABLED"
	if cfg.Features.Geocoding && cfg.Geocoding.Provider == "nominatim" && cfg.Geocoding.Endpoint != "" {
		geocoder := providers.NewNominatimProvider(cfg.Geocoding.Endpoint, time.Duration(cfg.Geocoding.TimeoutSeconds)*time.Second)
		cache := geocoding.NewFileCache(filepath.Join(dataDir, "geocoding", "cache.json"))
		cacheAge := time.Duration(cfg.Geocoding.CacheMinutes) * time.Minute
		geocodingStatus = "STANDBY"
		resolveZIP = func(zip string) (console.Origin, error) {
			result, cacheErr := cache.Find(zip)
			if cacheErr == nil && result.Fresh(time.Now().UTC(), cacheAge) {
				return zipOrigin(result, cfg.Origin.Timezone), nil
			}
			result, err := geocoder.Resolve(ctx, zip)
			if err != nil {
				if cacheErr == nil {
					return zipOrigin(result, cfg.Origin.Timezone), nil
				}
				return console.Origin{}, err
			}
			if saveErr := cache.Save(result); saveErr != nil {
				log.Warn("save geocoding cache", "error", saveErr)
			}
			return zipOrigin(result, cfg.Origin.Timezone), nil
		}
	}
	if resolveZIP != nil && cfg.Origin.HomeBaseZIP != "" && cfg.Origin.Latitude == 0 && cfg.Origin.Longitude == 0 {
		if resolved, resolveErr := resolveZIP(cfg.Origin.HomeBaseZIP); resolveErr == nil && resolved.Latitude != nil && resolved.Longitude != nil {
			cfg.Origin.Latitude = *resolved.Latitude
			cfg.Origin.Longitude = *resolved.Longitude
			if resolved.Timezone != "" {
				cfg.Origin.Timezone = resolved.Timezone
			}
			if cfg.App.ConfigPath != "" {
				if saveErr := config.Save(cfg.App.ConfigPath, cfg); saveErr != nil {
					log.Warn("save resolved Home Base coordinates", "error", saveErr)
				}
			}
			log.Info("resolved configured Home Base coordinates", "zip", cfg.Origin.HomeBaseZIP)
		}
	}
	location = "Not configured"
	if cfg.Origin.Latitude != 0 || cfg.Origin.Longitude != 0 {
		location = fmt.Sprintf("%.4f, %.4f", cfg.Origin.Latitude, cfg.Origin.Longitude)
	}
	var telescopeProvider telescope.Provider
	telescopeStatus := "DISABLED"
	telescopeName := ""
	if cfg.Features.Telescope {
		if cfg.Telescope.Provider == "alpaca" && cfg.Telescope.Endpoint != "" {
			telescopeProvider = telescope.NewAlpacaProvider(cfg.Telescope.Endpoint, cfg.Telescope.DeviceNumber, time.Duration(cfg.Telescope.TimeoutSeconds)*time.Second)
			telescopeStatus = "STANDBY"
		} else if cfg.Telescope.Provider == "dwarf" && cfg.Telescope.Endpoint != "" {
			telescopeProvider = telescope.NewDwarfProvider(cfg.Telescope.Endpoint, time.Duration(cfg.Telescope.TimeoutSeconds)*time.Second)
			telescopeStatus = "STANDBY"
		} else {
			telescopeStatus = "NOT CONFIGURED"
		}
	}
	var exportAtlas func(string) error
	if activeAtlas.Version != "" {
		exportAtlas = func(path string) error {
			return atlas.SaveCSV(config.ExpandPath(path), activeAtlas)
		}
	}
	var aiProvider ai.Provider
	aiStatus := "DISABLED"
	if cfg.Features.AI {
		if cfg.AI.Provider == "ollama" && cfg.AI.Endpoint != "" && cfg.AI.Model != "" {
			aiProvider = ai.NewOllamaProvider(cfg.AI.Endpoint, cfg.AI.Model, time.Duration(cfg.AI.TimeoutSeconds)*time.Second)
			aiStatus = "STANDBY"
		} else {
			aiStatus = "NOT CONFIGURED"
		}
	}
	var generateMissionBrief func(console.MissionBriefInput) (string, error)
	if aiProvider != nil {
		generateMissionBrief = func(input console.MissionBriefInput) (string, error) {
			return aiProvider.GenerateBrief(ctx, ai.BriefInput{Origin: input.Origin, Target: input.Target, TargetWindow: input.TargetWindow, Conditions: input.Conditions, Forecast: input.Forecast, Astronomy: input.Astronomy, Route: input.Route, Equipment: input.Equipment})
		}
	}
	var slewToTarget func(console.TargetSite) error
	var slewToTargetAt func(console.Origin, console.TargetSite) error
	if telescopeProvider != nil {
		slewToTarget = func(target console.TargetSite) error {
			return telescopeProvider.SlewTo(ctx, target.RightAscension, target.Declination)
		}
		if coordinateProvider, ok := telescopeProvider.(telescope.CoordinateSlewProvider); ok {
			slewToTargetAt = func(origin console.Origin, target console.TargetSite) error {
				if origin.Latitude == nil || origin.Longitude == nil {
					return fmt.Errorf("telescope slew requires mission-origin coordinates")
				}
				return coordinateProvider.SlewToAt(ctx, target.RightAscension, target.Declination, *origin.Latitude, *origin.Longitude)
			}
		}
	}
	loadMissions := func() ([]console.MissionSummary, error) {
		missions, err := store.ListMissions(ctx)
		if err != nil {
			return nil, err
		}
		return projectMissions(ctx, store, missions), nil
	}
	exportSync := func(path string) error {
		bundle, err := buildSyncBundle(ctx, store, deviceID)
		if err != nil {
			return err
		}
		return syncbundle.Save(config.ExpandPath(path), bundle)
	}
	importSync := func(path string) (string, error) {
		bundle, err := syncbundle.Load(config.ExpandPath(path))
		if err != nil {
			return "", err
		}
		report, err := bundle.Merge(ctx, store)
		if err != nil {
			return "", err
		}
		return formatSyncMergeReport(report), nil
	}
	apiStatus := "DISABLED"
	if cfg.Features.API {
		apiStatus = "STANDBY"
	}
	var openObsidianVault func() error
	if cfg.Obsidian.Enabled && obsidianNotes != "" {
		if info, statErr := os.Stat(obsidianNotes); statErr == nil && info.IsDir() {
			openObsidianVault = func() error { return exec.Command("xdg-open", obsidianNotes).Start() }
		}
	}
	weatherState := initializeWeather(ctx, cfg, store, weatherProvider, cfg.Origin.Latitude, cfg.Origin.Longitude)
	model := console.New(console.ThemeForName(cfg.App.Theme), console.Options{
		ThemeName:              cfg.App.Theme,
		HomeBaseName:           cfg.Origin.HomeBaseName,
		HomeBaseLocation:       location,
		HomeBaseZIP:            cfg.Origin.HomeBaseZIP,
		HomeBaseTimezone:       cfg.Origin.Timezone,
		HomeBaseLatitude:       coordinatePointer(cfg.Origin.Latitude, cfg.Origin.Longitude, true),
		HomeBaseLongitude:      coordinatePointer(cfg.Origin.Latitude, cfg.Origin.Longitude, false),
		HomeBaseConfigured:     cfg.Origin.HomeBaseName != "" && (cfg.Origin.HomeBaseZIP != "" || location != "Not configured"),
		DatabaseStatus:         "READY",
		ObsidianStatus:         obsidianStatus(cfg),
		OpenObsidianVault:      openObsidianVault,
		AtlasStatus:            atlasState,
		AtlasLocations:         atlasLocations,
		GPSStatus:              "UNAVAILABLE",
		GeocodingStatus:        geocodingStatus,
		ResolveZIP:             resolveZIP,
		WeatherStatus:          weatherState,
		AtlasEnabled:           cfg.Features.Atlas,
		Version:                "v0.1.0-dev",
		PluginsStatus:          pluginStatus,
		PluginsDir:             config.ExpandPath(pluginDir),
		Plugins:                pluginManifests,
		CommandPaletteEnabled:  cfg.Features.CommandPalette,
		AtlasExportDefaultPath: filepath.Join(dataDir, "atlas", "contribution-"+time.Now().UTC().Format("20060102-150405")+".csv"),
		TelescopeStatus:        telescopeStatus,
		TelescopeName:          telescopeName,
		AIStatus:               aiStatus,
		AIModel:                cfg.AI.Model,
		APIStatus:              apiStatus,
		BackupStatus:           "READY",
		BackupDefaultPath:      filepath.Join(dataDir, "backups", "nightops-"+time.Now().UTC().Format("20060102-150405")+".db"),
		SyncExportDefaultPath:  filepath.Join(dataDir, "sync", "nightops-"+time.Now().UTC().Format("20060102-150405")+".json"),
		SaveHomeBase: func(homeBase console.HomeBase) error {
			previousZIP := cfg.Origin.HomeBaseZIP
			cfg.Origin.HomeBaseName = homeBase.Name
			cfg.Origin.HomeBaseZIP = homeBase.ZIP
			if previousZIP != homeBase.ZIP {
				// No geocoder is configured. A changed ZIP must not retain
				// coordinates belonging to the previous Home Base.
				cfg.Origin.Latitude = 0
				cfg.Origin.Longitude = 0
			}
			configPath := cfg.App.ConfigPath
			if configPath == "" {
				configPath = filepath.Join(dataDir, "config.yaml")
			}
			return config.Save(configPath, cfg)
		},
		CreateBackup: func(path string) error {
			return store.Backup(ctx, config.ExpandPath(path))
		},
		ExportSync:       exportSync,
		ImportSync:       importSync,
		ScheduleTimezone: cfg.Origin.Timezone,
		CreateMission: func(origin console.Origin, equipmentID string, plannedStart, plannedEnd *time.Time) (string, error) {
			timezone := origin.Timezone
			if timezone == "" {
				timezone = cfg.Origin.Timezone
			}
			mission, err := planner.CreateMission(ctx, application.PlanInput{Kind: origin.Kind, Label: origin.Label, ZIP: origin.ZIP, Latitude: origin.Latitude, Longitude: origin.Longitude, Timezone: timezone, EquipmentProfileID: equipmentID, PlannedStart: plannedStart, PlannedEnd: plannedEnd})
			return mission.ID, err
		},
		SaveMissionTargets: func(missionID string, selected []console.TargetSite) error {
			targets := make([]domain.MissionTarget, 0, len(selected))
			for position, target := range selected {
				targets = append(targets, domain.MissionTarget{ID: target.ID, MissionID: missionID, Name: target.Name, Kind: target.Kind, RightAscension: target.RightAscension, Declination: target.Declination, Source: target.Source, Position: position, CreatedAt: time.Now().UTC()})
			}
			return planner.SetMissionTargets(ctx, missionID, targets)
		},
		EquipmentProfiles: projectEquipment(equipmentProfiles),
		EquipmentItems:    projectEquipmentItems(equipmentItems),
		SaveAtlas: func(path, version string) ([]console.AtlasSite, error) {
			catalog, err := atlas.LoadCSV(config.ExpandPath(path), version)
			if err != nil {
				return nil, err
			}
			if err := store.SaveAtlasCatalog(ctx, catalog); err != nil {
				return nil, err
			}
			return projectAtlas(catalog), nil
		},
		ExportAtlas:          exportAtlas,
		GenerateMissionBrief: generateMissionBrief,
		LoadMissions:         loadMissions,
		SaveEquipment: func(profile console.EquipmentProfile) error {
			equipment, err := domain.NewEquipmentProfile(profile.ID, profile.Name, profile.Description, time.Now().UTC())
			if err != nil {
				return err
			}
			return store.SaveEquipmentProfile(ctx, equipment)
		},
		SaveEquipmentItem: func(item console.EquipmentItem) error {
			equipment, err := domain.NewEquipmentItem(item.ID, item.ProfileID, item.Name, item.Category, item.Required, time.Now().UTC())
			if err != nil {
				return err
			}
			return store.SaveEquipmentItem(ctx, equipment)
		},
		StartMission: func(id string) error {
			_, err := planner.StartMission(ctx, id)
			return err
		},
		ActivateMission: func(id string) error {
			_, err := planner.ActivateMission(ctx, id)
			return err
		},
		RecordObservation: func(id, target, notes string) error {
			_, err := planner.RecordObservation(ctx, id, target, notes)
			return err
		},
		CompleteMission: func(id string) error {
			_, err := planner.CompleteMission(ctx, id)
			return err
		},
		RecordDebrief: func(id, summary string) error {
			_, err := planner.RecordDebrief(ctx, id, summary)
			return err
		},
		AstronomySummary: func(origin console.Origin) string {
			if origin.Latitude == nil || origin.Longitude == nil {
				return "unavailable until coordinates are known"
			}
			coordinates := astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}
			now := time.Now().UTC()
			snapshot, err := astronomy.Calculate(now, coordinates)
			if err != nil {
				return "unavailable: " + err.Error()
			}
			visibility, err := astronomy.CalculateVisibility(now, coordinates)
			if err != nil || visibility.AstronomicalDusk == nil || visibility.AstronomicalDawn == nil {
				return fmt.Sprintf("sun altitude %.1f° · moon %.0f%% illuminated · twilight unavailable", snapshot.SunAltitude, snapshot.MoonIllumination*100)
			}
			return fmt.Sprintf("sun altitude %.1f° · moon %.0f%% illuminated · astro dusk %s · dawn %s", snapshot.SunAltitude, snapshot.MoonIllumination*100, formatLocalTime(*visibility.AstronomicalDusk, originTimezone(cfg, origin)), formatLocalTime(*visibility.AstronomicalDawn, originTimezone(cfg, origin)))
		},
		WeatherSummary: func(origin console.Origin) string {
			return weatherSummary(ctx, cfg, store, weatherProvider, origin)
		},
		WeatherAssessment: func(origin console.Origin, selected []console.TargetSite) string {
			return missionWeatherAssessment(ctx, cfg, store, weatherProvider, origin, selected)
		},
		ForecastSummary: func(origin console.Origin) string {
			return weatherForecastSummary(ctx, cfg, store, weatherProvider, origin)
		},
		ForecastPoints: func(origin console.Origin) []console.ForecastPoint {
			return weatherForecastPoints(ctx, cfg, store, weatherProvider, origin)
		},
		MissionWindow: func(origin console.Origin, selected []console.TargetSite) (*time.Time, *time.Time, string) {
			return liveMissionWindow(cfg, origin, selected, time.Now().UTC())
		},
		ForecastTimezone:             cfg.Origin.Timezone,
		ForecastCloudMax:             cfg.Weather.ForecastCloudCoverMax,
		ForecastPrecipMax:            cfg.Weather.ForecastPrecipitationMax,
		ForecastThresholdsConfigured: true,
		RouteEnabled: func(origin console.Origin) bool {
			return origin.Latitude != nil && origin.Longitude != nil && (cfg.Origin.Latitude != 0 || cfg.Origin.Longitude != 0)
		},
		RouteSummary: func(origin console.Origin) string {
			return routeSummary(ctx, cfg, store, routeProvider, origin)
		},
		TargetsEnabled: targetsEnabled,
		Targets:        targetSites,
		TargetSummary: func(origin console.Origin, target console.TargetSite) string {
			if origin.Latitude == nil || origin.Longitude == nil {
				return "unavailable until coordinates are known"
			}
			minimumAltitude := targetMinimumAltitude(cfg)
			visibility, err := astronomy.CalculateTargetVisibility(time.Now().UTC(), astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}, astronomy.TargetCoordinates{RightAscension: target.RightAscension, Declination: target.Declination}, minimumAltitude)
			if err != nil {
				return "unavailable: " + err.Error()
			}
			if visibility.NoDarkWindow || visibility.Start == nil || visibility.End == nil {
				return fmt.Sprintf("not visible above %.0f° during astronomical night", minimumAltitude)
			}
			return fmt.Sprintf("%.0f°+ %s–%s · peak %.0f°", minimumAltitude, formatLocalTime(*visibility.Start, originTimezone(cfg, origin)), formatLocalTime(*visibility.End, originTimezone(cfg, origin)), visibility.MaximumAltitude)
		},
		TargetForecastSummary: func(origin console.Origin, target console.TargetSite, points []console.ForecastPoint) string {
			return targetForecastSummary(cfg, origin, target, points)
		},
		SlewToTarget:   slewToTarget,
		SlewToTargetAt: slewToTargetAt,
	})
	if cfg.Features.API {
		apiServer := api.New(api.Config{ListenAddr: cfg.API.ListenAddr, AllowSync: cfg.API.AllowSync, AuthEnv: cfg.API.AuthEnv}, api.Dependencies{
			ListMissions: func(_ context.Context) ([]api.Mission, error) {
				missions, err := loadMissions()
				if err != nil {
					return nil, err
				}
				projected := make([]api.Mission, 0, len(missions))
				for _, mission := range missions {
					projected = append(projected, api.Mission{ID: mission.ID, Name: mission.Name, Status: mission.Status, LaunchSiteName: mission.LaunchSiteName, Timezone: mission.Timezone, EquipmentID: mission.EquipmentProfileID, PlannedStart: mission.PlannedStart, PlannedEnd: mission.PlannedEnd, CreatedAt: mission.CreatedAt, UpdatedAt: mission.UpdatedAt})
				}
				return projected, nil
			},
			FindMission: func(_ context.Context, id string) (api.Mission, error) {
				mission, err := store.FindMission(ctx, id)
				if err != nil {
					return api.Mission{}, err
				}
				site, err := store.FindLaunchSite(ctx, mission.LaunchSiteID)
				if err != nil {
					return api.Mission{}, err
				}
				return api.Mission{ID: mission.ID, Name: mission.Name, Status: string(mission.Status), LaunchSiteName: site.Name, Timezone: site.Timezone, EquipmentID: mission.EquipmentProfileID, PlannedStart: mission.PlannedStart, PlannedEnd: mission.PlannedEnd, CreatedAt: mission.CreatedAt, UpdatedAt: mission.UpdatedAt}, nil
			},
			ExportSync: func(context.Context) (syncbundle.Bundle, error) {
				return buildSyncBundle(ctx, store, deviceID)
			},
			ImportSync: func(_ context.Context, bundle syncbundle.Bundle) (syncbundle.MergeReport, error) {
				return bundle.Merge(ctx, store)
			},
		})
		apiContext, cancelAPI := context.WithCancel(ctx)
		defer cancelAPI()
		go func() {
			if err := apiServer.Run(apiContext); err != nil {
				log.Error("local API stopped", "error", err)
			}
		}()
	}
	program := tea.NewProgram(model, tea.WithContext(ctx))
	_, err = program.Run()
	return err
}

// warmTargetKnowledge refreshes only missing target reference records before
// the console opens. Existing records make target selection fully offline and
// avoid network work while the observer is in the field.
func warmTargetKnowledge(ctx context.Context, cache *sqlite.Store, provider application.TargetKnowledgeProvider, sites []console.TargetSite) {
	if provider == nil {
		return
	}
	warmContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var wait sync.WaitGroup
	for _, site := range sites {
		target := domain.MissionTarget{ID: site.ID, Name: site.Name, Kind: site.Kind, RightAscension: site.RightAscension, Declination: site.Declination, Source: site.Source}
		if cached, err := cache.FindTargetKnowledge(warmContext, target.ID); err == nil && cached.Summary != "" {
			continue
		}
		wait.Add(1)
		go func(target domain.MissionTarget) {
			defer wait.Done()
			knowledge, err := provider.Fetch(warmContext, target)
			if err != nil {
				log.Warn("target reference unavailable during startup", "target", target.Name, "error", err)
				return
			}
			if err := cache.SaveTargetKnowledge(warmContext, knowledge); err != nil {
				log.Warn("target reference cache write failed", "target", target.Name, "error", err)
			}
		}(target)
	}
	wait.Wait()
}

func formatSyncMergeReport(report syncbundle.MergeReport) string {
	message := fmt.Sprintf("%d added · %d updated · %d skipped", report.Added, report.Updated, report.Skipped)
	if report.Conflicts == 0 {
		return message
	}
	message += fmt.Sprintf(" · %d conflicts retained locally", report.Conflicts)
	if len(report.ConflictIDs) == 0 {
		return message
	}
	const maxDisplayedConflicts = 8
	displayed := report.ConflictIDs
	if len(displayed) > maxDisplayedConflicts {
		displayed = displayed[:maxDisplayedConflicts]
	}
	message += " · review IDs: " + strings.Join(displayed, ", ")
	if len(report.ConflictIDs) > len(displayed) {
		message += fmt.Sprintf(" (+%d more)", len(report.ConflictIDs)-len(displayed))
	}
	return message
}

func zipOrigin(result geocoding.Result, timezone string) console.Origin {
	latitude, longitude := result.Latitude, result.Longitude
	return console.Origin{Kind: "ZIP Code", Label: "ZIP " + result.ZIP + " · " + result.Label, ZIP: result.ZIP, Timezone: timezone, Latitude: &latitude, Longitude: &longitude}
}

func liveMissionWindow(cfg config.Config, origin console.Origin, selected []console.TargetSite, now time.Time) (*time.Time, *time.Time, string) {
	if origin.Latitude == nil || origin.Longitude == nil {
		return nil, nil, "coordinates required"
	}
	coordinates := astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}
	visibility, err := astronomy.CalculateVisibility(now, coordinates)
	if err != nil || visibility.AstronomicalDusk == nil || visibility.AstronomicalDawn == nil {
		return nil, nil, "astronomical night unavailable"
	}
	start, end := visibility.AstronomicalDusk, visibility.AstronomicalDawn
	if len(selected) == 0 {
		return start, end, "astronomical dusk → dawn"
	}
	minimumAltitude := targetMinimumAltitude(cfg)
	for _, target := range selected {
		targetVisibility, targetErr := astronomy.CalculateTargetVisibility(now, coordinates, astronomy.TargetCoordinates{RightAscension: target.RightAscension, Declination: target.Declination}, minimumAltitude)
		if targetErr != nil || targetVisibility.Start == nil || targetVisibility.End == nil {
			continue
		}
		if targetVisibility.Start.Before(*start) {
			start = targetVisibility.Start
		}
		if targetVisibility.End.After(*end) {
			end = targetVisibility.End
		}
	}
	return start, end, fmt.Sprintf("live tonight · %d target(s) evaluated above %.0f°", len(selected), minimumAltitude)
}

func missionWeatherAssessment(ctx context.Context, cfg config.Config, cache weather.Cache, provider weather.Provider, origin console.Origin, selected []console.TargetSite) string {
	if len(selected) == 0 {
		return "select targets to evaluate"
	}
	if origin.Latitude == nil || origin.Longitude == nil {
		return "cannot evaluate without coordinates"
	}
	points := weatherForecastPoints(ctx, cfg, cache, provider, origin)
	if len(points) == 0 {
		return "UNKNOWN · hourly weather unavailable"
	}
	coordinates := astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}
	eligibleTargets := 0
	for _, target := range selected {
		forecast := make([]weather.ForecastPoint, 0, len(points))
		for _, point := range points {
			forecast = append(forecast, weather.ForecastPoint{At: point.At, Dark: point.Dark, TemperatureC: point.TemperatureC, CloudCoverPercent: point.CloudCoverPercent, PrecipitationProbability: point.PrecipitationProbability})
		}
		ranked, err := weather.RankTargetForecast(forecast, coordinates, astronomy.TargetCoordinates{RightAscension: target.RightAscension, Declination: target.Declination}, weather.RankingConfig{MinimumAltitude: targetMinimumAltitude(cfg), MaximumCloudCover: float64(cfg.Weather.ForecastCloudCoverMax), MaximumPrecipitation: float64(cfg.Weather.ForecastPrecipitationMax)})
		if err != nil {
			continue
		}
		for _, point := range ranked {
			if point.Eligible {
				eligibleTargets++
				break
			}
		}
	}
	if eligibleTargets == 0 {
		return fmt.Sprintf("NO-GO · no selected target meets cloud ≤%d%% and precip ≤%d%%", cfg.Weather.ForecastCloudCoverMax, cfg.Weather.ForecastPrecipitationMax)
	}
	return fmt.Sprintf("GO · %d/%d selected target(s) have a dark, weather-qualified window", eligibleTargets, len(selected))
}

func targetForecastSummary(cfg config.Config, origin console.Origin, target console.TargetSite, points []console.ForecastPoint) string {
	if origin.Latitude == nil || origin.Longitude == nil {
		return "unavailable until coordinates are known"
	}
	if len(points) == 0 {
		return "unavailable until hourly forecast data is available"
	}
	forecast := make([]weather.ForecastPoint, 0, len(points))
	for _, point := range points {
		forecast = append(forecast, weather.ForecastPoint{At: point.At, TemperatureC: point.TemperatureC, CloudCoverPercent: point.CloudCoverPercent, PrecipitationProbability: point.PrecipitationProbability})
		// Preserve the locally calculated darkness flag; it is not provider data.
		forecast[len(forecast)-1].Dark = point.Dark
	}
	ranked, err := weather.RankTargetForecast(forecast, astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}, astronomy.TargetCoordinates{RightAscension: target.RightAscension, Declination: target.Declination}, weather.RankingConfig{MinimumAltitude: targetMinimumAltitude(cfg), MaximumCloudCover: float64(cfg.Weather.ForecastCloudCoverMax), MaximumPrecipitation: float64(cfg.Weather.ForecastPrecipitationMax)})
	if err != nil {
		return "unavailable: " + err.Error()
	}
	for _, point := range ranked {
		if !point.Eligible {
			continue
		}
		return fmt.Sprintf("score %d · %s · target %.0f° · clouds %.0f%% · precip %.0f%%", point.Score, formatLocalTime(point.Point.At, originTimezone(cfg, origin)), point.TargetAltitude, *point.Point.CloudCoverPercent, *point.Point.PrecipitationProbability)
	}
	if len(ranked) == 0 {
		return "no qualifying window"
	}
	return "no qualifying window · " + ranked[0].Reason
}

func originTimezone(cfg config.Config, origin console.Origin) string {
	if origin.Timezone != "" {
		return origin.Timezone
	}
	return cfg.Origin.Timezone
}

func targetMinimumAltitude(cfg config.Config) float64 {
	if cfg.Astronomy.MinimumTargetAltitude >= 1 && cfg.Astronomy.MinimumTargetAltitude <= 90 {
		return float64(cfg.Astronomy.MinimumTargetAltitude)
	}
	return 30
}

func nonEmptyConfig(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func projectEquipment(profiles []domain.EquipmentProfile) []console.EquipmentProfile {
	projected := make([]console.EquipmentProfile, 0, len(profiles))
	for _, profile := range profiles {
		projected = append(projected, console.EquipmentProfile{ID: profile.ID, Name: profile.Name, Description: profile.Description})
	}
	return projected
}

func projectEquipmentItems(items []domain.EquipmentItem) []console.EquipmentItem {
	projected := make([]console.EquipmentItem, 0, len(items))
	for _, item := range items {
		projected = append(projected, console.EquipmentItem{ID: item.ID, ProfileID: item.ProfileID, Name: item.Name, Category: item.Category, Required: item.Required})
	}
	return projected
}

func projectPlugins(manifests []plugins.Manifest) []console.PluginManifest {
	projected := make([]console.PluginManifest, 0, len(manifests))
	for _, manifest := range manifests {
		projected = append(projected, console.PluginManifest{ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Description: manifest.Description, Entrypoint: manifest.Entrypoint, Capabilities: append([]string(nil), manifest.Capabilities...)})
	}
	return projected
}

func projectAtlas(catalog atlas.Catalog) []console.AtlasSite {
	locations := make([]console.AtlasSite, 0, len(catalog.Locations))
	for _, location := range catalog.Locations {
		bortle := ""
		if location.BortleClass != nil {
			bortle = fmt.Sprintf("%.1f", *location.BortleClass)
		}
		locations = append(locations, console.AtlasSite{ID: location.ID, Name: location.Name, Latitude: location.Latitude, Longitude: location.Longitude, Timezone: location.Timezone, Bortle: bortle})
	}
	return locations
}

func routeSummary(ctx context.Context, cfg config.Config, cache routing.Cache, provider routing.Provider, origin console.Origin) string {
	if origin.Latitude == nil || origin.Longitude == nil || (cfg.Origin.Latitude == 0 && cfg.Origin.Longitude == 0) {
		return "unavailable until Home Base and launch-site coordinates are known"
	}
	from := astronomy.Coordinates{Latitude: cfg.Origin.Latitude, Longitude: cfg.Origin.Longitude}
	to := astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}
	fromLabel := nonEmptyConfig(cfg.Origin.HomeBaseName, "Home Base")
	toLabel := nonEmptyConfig(origin.Label, "Launch Site")
	key := routing.Key(from, to)
	if cached, fetchedAt, expiresAt, err := cache.FindRoute(ctx, key); err == nil && time.Now().UTC().Before(expiresAt) {
		cached.FromLabel, cached.ToLabel = fromLabel, toLabel
		return formatRoutePlan(cached, true)
	} else if err == nil {
		_ = fetchedAt
	}
	if provider != nil {
		plan, err := provider.Route(ctx, from, to)
		if err == nil {
			plan.FromLabel, plan.ToLabel = fromLabel, toLabel
			if saveErr := cache.SaveRoute(ctx, key, plan, time.Now().UTC(), time.Now().UTC().Add(routeCacheDuration(cfg))); saveErr == nil {
				return formatRoutePlan(plan, false)
			}
		}
		if cached, _, _, cacheErr := cache.FindRoute(ctx, key); cacheErr == nil {
			cached.FromLabel, cached.ToLabel = fromLabel, toLabel
			return formatRoutePlan(cached, true)
		}
	}
	plan, err := routing.Build(fromLabel, from, toLabel, to)
	if err != nil {
		return "unavailable: " + err.Error()
	}
	return formatRoutePlan(plan, false)
}

func routeCacheDuration(cfg config.Config) time.Duration {
	minutes := cfg.Routing.CacheMinutes
	if minutes <= 0 {
		minutes = 360
	}
	return time.Duration(minutes) * time.Minute
}

func formatRoutePlan(plan routing.Plan, cached bool) string {
	if cached {
		plan.Provider += " (cached)"
	}
	return fmt.Sprintf("FROM %s\nTO %s\nROAD DISTANCE  %.1f km / %.1f mi\nTRAVEL TIME    %s\nSOURCE         %s", plan.FromLabel, plan.ToLabel, plan.DistanceKm, plan.DistanceMi, plan.TravelTime, plan.Provider)
}

func initializeWeather(ctx context.Context, cfg config.Config, cache weather.Cache, provider weather.Provider, latitude, longitude float64) string {
	if !cfg.Features.Weather {
		return "DISABLED"
	}
	if latitude == 0 && longitude == 0 {
		return "NOT CONFIGURED"
	}
	key := providers.LocationKey(latitude, longitude)
	if snapshot, err := cache.FindWeatherSnapshot(ctx, key); err == nil && snapshot.Fresh(time.Now().UTC()) {
		return "READY"
	}
	if provider == nil {
		return "NOT CONFIGURED"
	}
	snapshot, err := provider.Fetch(ctx, latitude, longitude)
	if err == nil {
		if err := cache.SaveWeatherSnapshot(ctx, snapshot); err != nil {
			log.Error("save weather snapshot", "error", err)
			return "ERROR"
		}
		return "READY"
	}
	if snapshot, cacheErr := cache.FindWeatherSnapshot(ctx, key); cacheErr == nil && snapshot.Validate() == nil {
		log.Warn("weather provider unavailable; using cached snapshot", "error", err)
		return "STANDBY"
	}
	log.Warn("weather provider unavailable", "error", err)
	return "ERROR"
}

func weatherSummary(ctx context.Context, cfg config.Config, cache weather.Cache, provider weather.Provider, origin console.Origin) string {
	if !cfg.Features.Weather {
		return "unavailable (weather disabled)"
	}
	if origin.Latitude == nil || origin.Longitude == nil {
		return "unavailable until coordinates are known"
	}
	latitude, longitude := *origin.Latitude, *origin.Longitude
	key := providers.LocationKey(latitude, longitude)
	if snapshot, err := cache.FindWeatherSnapshot(ctx, key); err == nil && snapshot.Fresh(time.Now().UTC()) {
		return formatWeatherSnapshot(snapshot, false)
	}
	if provider == nil {
		return "unavailable (provider not configured)"
	}
	snapshot, err := provider.Fetch(ctx, latitude, longitude)
	if err == nil {
		if saveErr := cache.SaveWeatherSnapshot(ctx, snapshot); saveErr != nil {
			return "unavailable (cache error)"
		}
		return formatWeatherSnapshot(snapshot, false)
	}
	if cached, cacheErr := cache.FindWeatherSnapshot(ctx, key); cacheErr == nil && cached.Validate() == nil {
		return formatWeatherSnapshot(cached, true)
	}
	return "unavailable (provider error)"
}

func formatWeatherSnapshot(snapshot weather.Snapshot, stale bool) string {
	parts := make([]string, 0, 3)
	if snapshot.TemperatureC != nil {
		parts = append(parts, fmt.Sprintf("%.1f°C", *snapshot.TemperatureC))
	}
	if snapshot.CloudCoverPercent != nil {
		parts = append(parts, fmt.Sprintf("clouds %.0f%%", *snapshot.CloudCoverPercent))
	}
	parts = append(parts, snapshot.Source)
	if stale {
		parts = append(parts, "stale")
	}
	return strings.Join(parts, " · ")
}

func weatherForecastSummary(ctx context.Context, cfg config.Config, cache weather.Cache, provider weather.Provider, origin console.Origin) string {
	if !cfg.Features.Weather {
		return "unavailable (weather disabled)"
	}
	if origin.Latitude == nil || origin.Longitude == nil {
		return "unavailable until coordinates are known"
	}
	key := providers.LocationKey(*origin.Latitude, *origin.Longitude)
	var cached weather.Snapshot
	if snapshot, err := cache.FindWeatherSnapshot(ctx, key); err == nil {
		cached = snapshot
		if snapshot.Fresh(time.Now().UTC()) && len(snapshot.Forecast) > 0 {
			return formatForecast(snapshot.Forecast, originTimezone(cfg, origin), false)
		}
	}
	forecastProvider, ok := provider.(weather.ForecastProvider)
	if !ok {
		if len(cached.Forecast) > 0 {
			return formatForecast(cached.Forecast, originTimezone(cfg, origin), true)
		}
		return "unavailable (forecast provider not configured)"
	}
	snapshot, err := forecastProvider.FetchForecast(ctx, *origin.Latitude, *origin.Longitude)
	if err == nil {
		if saveErr := cache.SaveWeatherSnapshot(ctx, snapshot); saveErr != nil {
			return "unavailable (cache error)"
		}
		return formatForecast(snapshot.Forecast, originTimezone(cfg, origin), false)
	}
	if len(cached.Forecast) > 0 {
		return formatForecast(cached.Forecast, originTimezone(cfg, origin), true)
	}
	return "unavailable (forecast provider error)"
}

func formatForecast(points []weather.ForecastPoint, timezone string, stale bool) string {
	if len(points) == 0 {
		return "unavailable (no hourly data)"
	}
	parts := make([]string, 0, 4)
	for _, point := range points {
		if len(parts) == 3 {
			break
		}
		value := formatLocalTime(point.At, timezone)
		if point.TemperatureC != nil {
			value += fmt.Sprintf(" %.1f°C", *point.TemperatureC)
		}
		if point.CloudCoverPercent != nil {
			value += fmt.Sprintf(" clouds %.0f%%", *point.CloudCoverPercent)
		}
		if point.PrecipitationProbability != nil {
			value += fmt.Sprintf(" precip %.0f%%", *point.PrecipitationProbability)
		}
		parts = append(parts, value)
	}
	if stale {
		parts = append(parts, "stale")
	}
	return strings.Join(parts, " | ")
}

func weatherForecastPoints(ctx context.Context, cfg config.Config, cache weather.Cache, provider weather.Provider, origin console.Origin) []console.ForecastPoint {
	if !cfg.Features.Weather || origin.Latitude == nil || origin.Longitude == nil {
		return nil
	}
	key := providers.LocationKey(*origin.Latitude, *origin.Longitude)
	var cached weather.Snapshot
	if snapshot, err := cache.FindWeatherSnapshot(ctx, key); err == nil {
		cached = snapshot
		if snapshot.Fresh(time.Now().UTC()) && len(snapshot.Forecast) > 0 {
			return projectForecast(snapshot.Forecast, origin)
		}
	}
	forecastProvider, ok := provider.(weather.ForecastProvider)
	if !ok {
		return projectForecast(cached.Forecast, origin)
	}
	snapshot, err := forecastProvider.FetchForecast(ctx, *origin.Latitude, *origin.Longitude)
	if err == nil {
		if saveErr := cache.SaveWeatherSnapshot(ctx, snapshot); saveErr == nil {
			return projectForecast(snapshot.Forecast, origin)
		}
	}
	return projectForecast(cached.Forecast, origin)
}

func projectForecast(points []weather.ForecastPoint, origin console.Origin) []console.ForecastPoint {
	projected := make([]console.ForecastPoint, 0, len(points))
	coordinatesAvailable := origin.Latitude != nil && origin.Longitude != nil
	var coordinates astronomy.Coordinates
	if coordinatesAvailable {
		coordinates = astronomy.Coordinates{Latitude: *origin.Latitude, Longitude: *origin.Longitude}
	}
	for _, point := range points {
		dark := false
		if coordinatesAvailable {
			if snapshot, err := astronomy.Calculate(point.At, coordinates); err == nil {
				dark = snapshot.SunAltitude <= -18
			}
		}
		projected = append(projected, console.ForecastPoint{At: point.At, TemperatureC: point.TemperatureC, CloudCoverPercent: point.CloudCoverPercent, PrecipitationProbability: point.PrecipitationProbability, Dark: dark})
	}
	return projected
}

func projectMissions(ctx context.Context, store *sqlite.Store, missions []domain.Mission) []console.MissionSummary {
	projected := make([]console.MissionSummary, 0, len(missions))
	for _, mission := range missions {
		siteName := ""
		timezone := ""
		if site, err := store.FindLaunchSite(ctx, mission.LaunchSiteID); err == nil {
			siteName = site.Name
			timezone = site.Timezone
		}
		projected = append(projected, console.MissionSummary{ID: mission.ID, Name: mission.Name, Status: string(mission.Status), LaunchSiteName: siteName, Timezone: timezone, EquipmentProfileID: mission.EquipmentProfileID, PlannedStart: mission.PlannedStart, PlannedEnd: mission.PlannedEnd, CreatedAt: mission.CreatedAt, UpdatedAt: mission.UpdatedAt})
	}
	return projected
}

func buildSyncBundle(ctx context.Context, store *sqlite.Store, deviceID string) (syncbundle.Bundle, error) {
	missions, err := store.ListMissions(ctx)
	if err != nil {
		return syncbundle.Bundle{}, err
	}
	sites := make([]domain.LaunchSite, 0)
	siteIDs := make(map[string]struct{})
	observations := make([]domain.Observation, 0)
	for _, mission := range missions {
		if _, seen := siteIDs[mission.LaunchSiteID]; !seen {
			site, findErr := store.FindLaunchSite(ctx, mission.LaunchSiteID)
			if findErr != nil {
				return syncbundle.Bundle{}, findErr
			}
			sites = append(sites, site)
			siteIDs[mission.LaunchSiteID] = struct{}{}
		}
		missionObservations, listErr := store.ListObservations(ctx, mission.ID)
		if listErr != nil {
			return syncbundle.Bundle{}, listErr
		}
		observations = append(observations, missionObservations...)
	}
	profiles, err := store.ListEquipmentProfiles(ctx)
	if err != nil {
		return syncbundle.Bundle{}, err
	}
	items := make([]domain.EquipmentItem, 0)
	for _, profile := range profiles {
		profileItems, listErr := store.ListEquipmentItems(ctx, profile.ID)
		if listErr != nil {
			return syncbundle.Bundle{}, listErr
		}
		items = append(items, profileItems...)
	}
	return syncbundle.NewBundleWithDevice(deviceID, time.Now().UTC(), sites, missions, observations, profiles, items)
}

func formatLocalTime(value time.Time, timezone string) string {
	if timezone != "" {
		if location, err := time.LoadLocation(timezone); err == nil {
			value = value.In(location)
		}
	}
	return value.Format("15:04 MST")
}

func coordinatePointer(latitude, longitude float64, latitudeValue bool) *float64 {
	if latitude == 0 && longitude == 0 {
		return nil
	}
	if latitudeValue {
		return &latitude
	}
	return &longitude
}

func obsidianStatus(cfg config.Config) string {
	if !cfg.Obsidian.Enabled {
		return "DISABLED"
	}
	if cfg.Obsidian.VaultDir == "" {
		return "NOT CONFIGURED"
	}
	_, notesDir, err := configuredObsidianPath(cfg)
	if err != nil {
		return "NOT CONFIGURED"
	}
	info, err := os.Stat(notesDir)
	if err != nil || !info.IsDir() {
		return "NOT CONFIGURED"
	}
	return "READY"
}

func configuredObsidianPath(cfg config.Config) (string, string, error) {
	if !cfg.Obsidian.Enabled || strings.TrimSpace(cfg.Obsidian.VaultDir) == "" {
		return "", "", fmt.Errorf("Obsidian is disabled or no vault directory is configured")
	}
	root := config.ExpandPath(cfg.Obsidian.VaultDir)
	notes := root
	if strings.TrimSpace(cfg.Obsidian.NotesDir) != "" {
		notes = filepath.Join(root, cfg.Obsidian.NotesDir)
	}
	return root, notes, nil
}

func prepareObsidianWorkspace(cfg config.Config) (string, string, error) {
	root, notes, err := configuredObsidianPath(cfg)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(notes, 0o755); err != nil {
		return "", "", fmt.Errorf("create Obsidian notes directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(notes, ".obsidian"), 0o755); err != nil {
		return "", "", fmt.Errorf("create Obsidian vault metadata directory: %w", err)
	}
	return root, notes, nil
}

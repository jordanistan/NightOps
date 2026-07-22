# Roadmap

## Milestone 1
- Go foundation and package boundaries
- Configuration validation and structured logging
- Bubble Tea startup, Mission Origin, Home Base, and settings navigation
- SQLite migrations and mission lifecycle invariants
- Atomic Obsidian mission projection
- Linked Obsidian mission, location, and observed-target notes
- Equipment profile creation, selection, and durable mission association
- Durable launch-site and mission repositories
- Mission Planning confirmation and export integration
- Explicit timezone-aware mission observing windows
- Operation lifecycle: launch, activate, observation logging, completion, and debrief
- CI, Makefile, tests, and development guide
- Versioned offline Austin observing-site catalog with source attribution
- Atlas browser and mission-origin selection
- Deterministic solar and lunar planning summary for known coordinates
- Open-Meteo current-conditions adapter with cache-first startup behavior
- Open-Meteo hourly forecast parsing and durable offline cache
- Capability-gated hourly forecast browser and hour selection
- Forecast filters for astronomical darkness, cloud cover, and precipitation
- Configurable forecast cloud-cover and precipitation thresholds
- Target-aware ranking of cached forecast windows using local altitude and darkness
- Offline geodesic route facts between Home Base and launch site
- Optional OSRM-compatible driving route adapter with SQLite cache
- Weather details shown in Mission Planning with source and stale-cache labels
- Deterministic civil, nautical, and astronomical twilight transitions
- Versioned offline celestial target seed catalog
- Target selector with astronomical-night and minimum-altitude visibility windows
- Optional cache-first ZIP geocoding with explicit unknown-coordinate fallback

## Milestone 2
- Configurable forecast thresholds and target-aware weather ranking
- Provider-specific routing policies, bounded retry/backoff, and route freshness controls
- Visibility windows for time-varying target tracks and configurable constraints (minimum altitude)
- Route planning and travel-time estimates
- Expanded SkyBase Atlas import and update workflow (validated local CSV, persisted active catalog)
- Capability-aware command palette with typed route transitions
- Atomic local SQLite backup workflow from Settings
- Provenance-preserving local SkyBase Atlas contribution export

## Milestone 3
- Alpaca-compatible and Dwarf II telescope control adapters with explicit provider boundaries
- Equipment inventory expansion and session readiness checks (initial local inventory slice)
- Plugin manifest discovery and read-only registry
- AI mission planner
- Local Ollama-compatible mission brief provider foundation
- Offline Mission Archive and read-only mission detail view
- Versioned offline sync bundle export/import with conflict-safe merge
- Loopback-first local API for mission archive and sync clients
- `nightopsctl` companion client for status, archive, and sync workflows
- Read-only API mission-detail projection for companion clients
- Embedded responsive offline companion for archive and sync workflows
- Multi-device sync identity and equal-timestamp conflict reporting
- MkDocs Material documentation site and GitHub Pages deployment
- Configured mission-control and observatory console themes
- Community Atlas contribution package with schema and provenance envelope

## Milestone 4
- Mobile companion
- Multi-user sync

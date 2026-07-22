# NightOps — Powered by SkyBase

**Mission planning for amateur astronomers.**

Motto:
> Every night under the stars is a mission worth remembering.

## Repository Layout

- `cmd/nightops/` — Go + Bubble Tea application entrypoint
- `internal/` — domain, console, configuration, storage, and export packages
- `migrations/` — human-readable database schema history
- `docs/` — architecture, development, roadmap, and website documentation
- `docs/site/` — MkDocs Material source for the GitHub Pages documentation site
- `repos/` — ecosystem repository planning

## Phase 1
- Production foundation and first vertical slice
- Animated launcher and Mission Origin console
- Home Base, ZIP Code, and Current GPS origin paths
- SQLite migrations and durable mission model
- Obsidian Markdown projection

## Current status

The repository contains the initial Go foundation: configuration validation,
SQLite initialization, mission lifecycle rules, a staged cinematic boot,
responsive launch console, theme tokens, feature flags, atomic Obsidian
export, durable mission planning, tests, CI, and development documentation.
The root console includes a `Ctrl+K` command palette that filters and routes only
to capabilities initialized for the current installation.
Mission operations now support launch, activation, observation logging,
completion, debrief, and preserved Obsidian flight records.
Mission Planning includes a retained final review step that groups the origin,
target sequence, observing window, equipment, and known conditions before
creating a mission. It also includes source-attributed offline targets and calculated
astronomical-night visibility windows when coordinates are available, plus
cached hourly weather details and selectable forecast hours when weather is
enabled and forecast points are available, with darkness, configurable cloud,
and configurable precipitation filters. Selecting a target also ranks actual
forecast windows by local target altitude and reported conditions without
inventing missing measurements.
Target visibility constraints are configurable through
`astronomy.minimum_target_altitude` and default to 30°.
Obsidian exports now maintain linked mission, location, selected-target,
reusable-equipment, and mission-scoped equipment notes. Startup warms cached
target reference pages before a session; Final Review can generate the full
mission template and open the vault in one action. Mission notes include
auto-filled properties, detailed cached weather, equipment checklists,
ordered target capture guidance, and target reference/image links. Target
pages retain links and details for every mission that reused the object.
Settings and Mission Planning also support user-created equipment profiles;
selected profile IDs are persisted with missions. Profiles can record required
inventory items and expose an honest session-readiness check before launch.
Mission Planning accepts an explicit local observing window and persists it as
UTC timestamps for durable offline review.
Settings can create owner-readable, atomic SQLite backups without interrupting
the active mission database.
The launch console also includes an offline Mission Archive and read-only detail
view for planned, active, completed, and cancelled missions.
Settings can export and merge versioned offline sync bundles for future mobile
or API clients; imports preserve stable IDs, skip older records, and delete
nothing.
An opt-in loopback API exposes the same archive and sync contracts for future
mobile clients without exposing SQLite directly.
The companion `nightopsctl` command can read API status and mission details and move
versioned sync bundles between a running installation and local files.
Bundles carry a stable local device identity; equal-timestamp divergent records
are retained locally and reported as conflicts rather than silently overwritten.
When the API is enabled, open `/companion/` for the responsive browser companion.
Coordinate-backed missions also expose offline straight-line route facts while
clearly leaving driving distance and ETA unavailable without a routing service.
An optional OSRM-compatible routing adapter can provide and cache road distance
and duration when explicitly enabled. Routing requests use bounded configurable
timeouts and retries, while stale cached route facts remain clearly labeled.
Network and hosted AI systems remain opt-in. Optional local AI mission briefs
now support an Ollama-compatible provider when explicitly configured; only
displayed planning facts are sent and unknown values are not filled in.
Optional telescope control now supports Alpaca-compatible local HTTP and Dwarf
II local WebSocket adapters when `features.telescope` is enabled and an
endpoint is configured. Dwarf slews require real mission-origin coordinates.
When `features.plugins` is enabled,
NightOps can inspect validated local plugin manifests without executing plugin
code. SkyBase Atlas includes a small, versioned offline
Austin-area catalog with source-attributed observing fields. When enabled,
users can import a validated local CSV catalog from Settings; the active
catalog is persisted in SQLite and preferred on offline startup. A loaded
catalog can also be exported locally as a provenance-preserving CSV for
community review or contribution. Fresh installations enable the Open-Meteo
adapter and ZIP geocoding for online-first enrichment, both with local SQLite
caches. Existing configurations retain their explicit feature settings.
Optional ZIP geocoding uses a
bounded Nominatim-compatible adapter with a durable local cache; unavailable
providers never fabricate coordinates.
Community Atlas contributions can also be validated and wrapped as explicit
`unreviewed` JSON packages with `nightopsctl`; no contribution is silently
promoted into the active local catalog.

Start locally with:

```sh
make verify
make run
```

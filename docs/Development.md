# Development

NightOps is intentionally a small modular monolith at this stage. The core
domain must remain independent of Bubble Tea, SQLite, network providers, and
the filesystem.

## Local setup

Install Go 1.26 or newer, then run:

```sh
cp config.example.yaml config.yaml
make verify
make run
```

The default database is created under `~/.local/share/nightops`. An explicit
`--config config.yaml` path takes precedence; otherwise NightOps reloads the
owner-readable configuration under the data directory when it exists and uses
validated defaults for a fresh checkout. This keeps local settings persistent
without requiring a checked-in machine-specific file.

## Package rules

- `internal/domain` owns entities, value objects, and invariants.
- `internal/application` will own use cases and ports as the first vertical
  slice grows.
- `internal/console` owns Bubble Tea models and user interaction only.
- `internal/storage` owns SQLite and migrations.
- `internal/providers` will own network or hardware adapters.
- `internal/export` owns projections such as Obsidian Markdown.
- `internal/app` is the composition root and the only place that wires
  concrete implementations together.

Use UTC for persisted timestamps and retain the source/timezone alongside
astronomy and provider results. Do not store mutable forecast data directly on
completed missions.

## Verification expectations

Every change should pass `make verify`. Domain tests should cover state-machine
invariants, storage tests should use temporary or in-memory databases, and
provider tests should use deterministic fixtures. UI behavior should be
testable without requiring a live terminal.

## Startup experience

The initial console is organized as a short staged boot followed by the launch
console. Boot stages communicate local initialization without inventing remote
data and can be skipped with Enter or Space. The launch console presents one
primary neon `LAUNCH MISSION` action and shows database, Obsidian, weather,
Atlas, and GPS capability states as secondary telemetry. The action uses the
configured Home Base or opens setup, then opens the target selector. Target
selection is the first mission input; the current run, astronomical darkness,
selected-target visibility, and live/cached weather build the mission context
automatically.

The configured `app.theme` is applied at startup. Supported palettes are
`mission-control` and `observatory`; configuration validation rejects other
values.

When no `-config` flag is supplied, NightOps reloads the owner-readable
`<data_dir>/config.yaml` written by Settings. This preserves Home Base and
other persisted settings across restarts. An explicit `-config` path always
takes precedence.

The launch layout collapses vertically below wide-terminal sizing and places
status beside the origin panel on wider terminals. Atlas is omitted entirely
when its feature flag is disabled. The UI uses Lip Gloss theme tokens rather
than raw terminal escape sequences.

The embedded Atlas catalog lives at
`internal/atlas/data/austin-v1.csv`. It is versioned with the application and
each row records its source in the catalog data. Keep additions source-grounded
and validate them through the Atlas parser tests. The initial Austin-area
entries are based on the [Austin Astronomical Society observing fields](https://austinastro.org/index.php/aas-observing-fields/)
and [Texas Parks and Wildlife Bortle ratings](https://tpwd.texas.gov/state-parks/parks/things-to-do/stargazing/bortle-ratings).

Weather uses `internal/providers.OpenMeteoProvider` behind the
`internal/weather` provider/cache interfaces. Fresh installations enable it;
existing configurations can set `features.weather: true` and configure the
endpoint and cache duration under `weather`. Startup reads a fresh SQLite
snapshot before making a request. A
stale snapshot is retained as `STANDBY` when the provider is unreachable, and
no weather value is fabricated. Provider tests use an in-process transport and
never depend on the public network.

Hourly weather is represented by `weather.ForecastPoint` values and persisted
by migration `004_weather_forecast.sql`. Keep forecast arrays time-aligned and
ascending. The UI shows only points actually returned by the provider and
labels cached fallback data as stale.

The forecast browser is capability-gated by the presence of actual forecast
points. Its Enter action records the selected hour in Mission Planning, and Esc
returns to the prior route. Do not expose the browser when the provider/cache
cannot supply points.

Forecast browser filters are explicit: `d` restricts to locally calculated
astronomical darkness, `c` restricts cloud cover to the configured
`weather.forecast_cloud_cover_max`, and `p` restricts precipitation probability
to `weather.forecast_precipitation_max`. Defaults are 50% and 20%. Missing
provider fields do not pass a filter. Keep the empty-result state visible
instead of silently restoring unfiltered data.

After a target is selected, the application ranks available forecast points
with the same local darkness and target-altitude model. The ranking is a
planning aid, not fabricated advice: only points inside the configured weather
thresholds and above 30° target altitude qualify, and missing weather fields
produce an explicit unavailable reason.

Offline route facts belong in `internal/routing`. Use the WGS84 coordinate
validation already owned by astronomy and label the result `offline geodesic`.
Never convert straight-line distance into a fabricated driving ETA; provider
integration must introduce its own source, timeout, cache, and failure tests.

Routing configuration is under `routing` and is disabled by default through
`features.routing`. The initial adapter targets the OSRM route API and stores
provider results in `route_cache`; keep the endpoint user-configurable so a
self-hosted OSRM instance can be used. The adapter follows the documented OSRM
route response distance in meters and duration in seconds. `timeout_seconds`,
`max_retries`, and `retry_backoff_millis` bound transient provider failures;
semantic no-route and malformed-response errors are not retried. A stale local
route remains available as a clearly cached fallback when the provider fails.

Mission Planning also receives a weather summary callback from the composition
root. Keep this callback provider-aware but UI-agnostic: cached values must
retain their source and stale state, while origins without coordinates must
remain unavailable. Astronomy summaries use local twilight calculations and
must not substitute fixed sunset or sunrise times.

The embedded target seed catalog is at `internal/targets/data/targets-v1.csv`.
Each entry includes an explicit source and fixed equatorial coordinates. The
initial entries are transcribed from [NASA Hubble M31](https://science.nasa.gov/asset/hubble/andromeda-galaxy-m31/),
[NASA Hubble M42](https://science.nasa.gov/asset/hubble/evolution-of-the-orion-nebula-m42/),
and [NASA Hubble M13](https://science.nasa.gov/asset/hubble/compass-and-scale-image-of-m13/).
Target visibility must remain a calculation result: do not label a target
recommended unless a future ranking contract supplies that evidence.

Target visibility uses `astronomy.minimum_target_altitude`, measured in degrees
above the horizon, with a default of 30°. The same constraint is applied to the
target-aware forecast ranking so the displayed window and condition score use
the same planning rule.

Mission Planning can build an ordered sequence from multiple catalog targets.
Space or Enter toggles a target, `c` confirms the sequence, and the selected
objects are persisted with the mission and exported into
`Missions/<mission>/Targets/` notes.

Atlas CSV imports must use the exact header in `internal/atlas/catalog.go`, a
non-empty version, and source-attributed rows with valid coordinates and
timezone values. Enable `features.atlas` before importing from Settings with
`i Import Atlas CSV`. Imports are local-file only, replace the active catalog
atomically, and are retained in SQLite for offline startup.

With a validated catalog loaded, Settings exposes `x Export Atlas CSV`. The
export is local-only, atomic, owner-readable, and preserves the exact source
provenance required for community review. The command palette exposes the same
workflow as `Export SkyBase Atlas`. Empty or invalid catalogs never produce a
contribution file.

For community review, `nightopsctl` can validate a CSV or wrap it in a
self-describing contribution package without network access:

```sh
go run ./cmd/nightopsctl -atlas-version community-2026-07 \
  atlas-validate ./sites.csv
go run ./cmd/nightopsctl -atlas-version community-2026-07 \
  atlas-package ./sites.csv ./nightops-atlas-contribution.json
```

The package records its schema, catalog version, generation time, explicit
`unreviewed` status, and every row's source provenance. It is a review/share
artifact, not an automatic catalog update; imports still require the normal
local validation workflow.

Launch actions are routed through the root `Route` state machine. Home Base,
ZIP, GPS, Mission Planning, Settings, Help, and Error are separate retained
states; child forms own their input models and return through `previousRoute`.
The launch console owns the visible launch action. Enter starts the configured
Home Base workflow (or Home Base setup), and the countdown leads into Mission
Planning. Origin changes, ZIP entry, GPS acquisition, and Atlas browsing remain
available through their typed routes, Settings, and the command palette; they
are not dead menu entries on the primary screen.

The root console also provides a capability-aware command palette with `Ctrl+K`.
Type to filter local commands, use the arrow keys or `j`/`k` to move, and press
Enter to execute. Atlas, equipment, and plugin commands are omitted when their
corresponding capability is unavailable. `Esc` returns to the console that
opened the palette.

Target reference knowledge uses a bounded Wikipedia-compatible page-summary
adapter configured under `target_knowledge`. Startup warms missing catalog
entries into SQLite; target selection is cache-first during a live session and
only falls back to a live request when no cached summary exists. The adapter
stores source page, summary, representative image, status, and fetch time. It
never blocks mission persistence when the network is unavailable.

Optional telescope control uses the `internal/telescope` port. Set
`features.telescope: true` and configure either an Alpaca endpoint or a Dwarf
II host under `telescope` to expose `SLEW TELESCOPE TO TARGET` after selecting
an offline catalog target. Dwarf hosts use the local WebSocket API on port
`9900` unless a complete `ws://` endpoint is supplied. Startup does not
contact the device; the first control request is bounded by `timeout_seconds`.
Dwarf commands use the mission origin's real coordinates and fail clearly when
those coordinates are unavailable. Transport or device errors remain visible.
The active Deep Space console labels telescope control unavailable when neither
adapter is configured; it never presents a nonfunctional slew action.

AI mission briefs are opt-in and local-only in the initial implementation. Set
`features.ai: true`, configure an Ollama-compatible `ai.endpoint` and `ai.model`,
and use `GENERATE MISSION BRIEF` from Mission Planning. The provider receives
only already-visible planning facts, uses bounded requests, and must state when
a value is unknown; no AI request is made during startup.

Launch also exposes `OPEN MISSION ARCHIVE`. The archive reads current records
from SQLite, supports keyboard selection, and opens a read-only detail view;
`Esc` returns through both levels without changing mission state. An empty
archive is an explicit state, not a dead menu action.

Settings exposes `y Export Sync Bundle` and `u Import Sync Bundle`. Bundles are
versioned JSON with stable IDs, UTC timestamps, and a generated source-device
identity stored at `<data_dir>/sync/device-id`. Import uses newer-records-win
merging and reports added, updated, skipped, and equal-timestamp conflict
counts; it never deletes local records. Equal-timestamp divergent records keep
the local value and expose a conflict ID for review. This format is
intentionally local-only and can be consumed by future mobile or API clients
without opening the SQLite file directly.

The local API is enabled with `features.api: true`. Its default listener is
`127.0.0.1:8787`; `GET /api/v1/status`, `GET /api/v1/missions`, and
`GET /api/v1/missions/{id}` are read-only.
Sync endpoints remain disabled unless `api.allow_sync: true`. Do not bind it to
a non-loopback address unless remote access is intentional and
`api.allow_remote: true` is also set.

Remote binding also requires `api.auth_env` to name an environment variable
whose value authenticates requests. Clients send it as a Bearer credential; the
credential itself is never stored in YAML or SQLite. For example:

```sh
NIGHTOPS_API_AUTH='set-by-the-operator' make run
go run ./cmd/nightopsctl -auth-env NIGHTOPS_API_AUTH status
```

The embedded browser companion accepts the same credential in an in-memory
field when remote authentication is enabled. Static companion assets remain
loadable before authentication so the user can enter that credential. The
credential itself is never persisted in browser storage, and API requests
remain protected.

The companion client is `nightopsctl`; it talks to the API and never opens the
NightOps SQLite database directly:

```sh
go run ./cmd/nightopsctl status
go run ./cmd/nightopsctl missions
go run ./cmd/nightopsctl mission MISSION_ID
go run ./cmd/nightopsctl export-sync /path/to/nightops-sync.json
go run ./cmd/nightopsctl import-sync /path/to/nightops-sync.json
```

The server must be enabled in configuration. Sync commands also require
`api.allow_sync: true`; the default listener remains loopback-only.

When the API is enabled, the same listener serves the responsive NightOps
Companion at `http://127.0.0.1:8787/companion/`. It is an installable web app
with a cached shell and cached read-only mission list. API records are never
fabricated: when the API is unavailable, the companion clearly reports offline
state and only shows records previously cached in that browser. Sync controls
remain disabled unless the API reports sync enabled.

Use Settings `b Create Database Backup` to create a consistent SQLite snapshot.
The default destination is inside the NightOps data directory under `backups/`;
users may provide another local path. Backup writes are atomic, use owner-only
permissions, and never overwrite the active database.

Mission Planning's `REVIEW & CREATE OBSIDIAN MISSION` route groups the selected
origin, ordered targets, per-target windows, automatic observing window,
equipment, and weather GO/NO-GO assessment before any write.
Returning to planning preserves the retained child model. The final review
launch actions then persist the planned mission
and its launch site. When Obsidian is configured, the same confirmation writes
a mission note using an atomic replacement. A ZIP origin is intentionally
exported with `unknown` coordinates until geocoding is available. Atlas
origins retain the catalog site's timezone for schedules and local display.
Changing a Home Base ZIP clears previously stored coordinates because no
geocoder is configured; NightOps never carries coordinates across unrelated
home-base locations.

ZIP geocoding is enabled for fresh installations. Existing configurations can
set `features.geocoding: true` and configure the Nominatim-compatible endpoint
under `geocoding` to resolve coordinates. Results are cached in the local data
directory and reused offline; if the provider is unavailable, the mission still
continues with explicitly unknown coordinates.

Obsidian exports also create `Locations/<site>.md` notes and, when a target is
recorded, `Targets/<target>.md` notes. Mission, location, and target links use
the sanitized note paths; user-authored `## Notes` and Flight Recorder sections
must survive rewrites.

The Operation console follows the durable lifecycle: planned, launched, active,
and completed. Active operations can open the Flight Recorder form, which
requires a target name and persists notes locally. Completing an operation
opens a Debrief screen and preserves all observation entries in the mission
note.

Mission windows are calculated from the current run's astronomical darkness and
the selected target visibility intervals. No date or time entry is required.
An `OVERRIDE AUTO WINDOW (ADVANCED)` action remains available for exceptional
planning cases; it uses `YYYY-MM-DD HH:MM` in the configured origin timezone.
Persistence stores UTC timestamps and Obsidian exports include the calculated
window in frontmatter and the Mission Window section.

Equipment profiles are intentionally user-created; the application does not
seed fictional telescope or camera data. Create them through Settings, select
one in Mission Planning, and keep the association validated against
`equipment_profiles` before persisting a mission. Use `v Equipment Inventory`
to record owned items by category and mark required items with `Ctrl+r` while
creating an item. Mission Planning exposes `CHECK EQUIPMENT READINESS`; it
reports `NOT CONFIGURED` until at least one required item is recorded and does
not pretend to verify physical presence automatically.

The current build sets `-buildvcs=false` because source archives and exported
working trees may not contain Git metadata. This keeps reproducible local and
CI builds independent of checkout state.

## Commit practice

Keep commits narrow and descriptive. Main should always build, test, and start.
Incomplete integrations belong behind feature flags and must have a documented
reason for being disabled.

Plugins currently have a metadata-only contract. Enable `features.plugins` and
set `plugins.dir` to a local directory containing one subdirectory per plugin;
each plugin may provide a `manifest.yaml` with `id`, `name`, `version`,
`description`, `entrypoint`, and `capabilities`. Settings opens the Plugin
Registry with `p`. Discovery validates and displays metadata but does not run
the declared entrypoint; execution requires a future sandboxed contract.

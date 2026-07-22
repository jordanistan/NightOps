# NightOps v1 Architecture

The detailed pre-implementation review is in
[`ARCHITECTURE_REVIEW.md`](../ARCHITECTURE_REVIEW.md). The implementation
starts as an offline-first modular monolith: one Go binary, one local SQLite
database, and ports for future providers and clients.

## Core Concepts
- NightOps = User application
- SkyBase = Data engine
- SkyBase Atlas = Community location database

## Startup Flow
1. Splash animation
2. Choose Origin:
   - Home Base
   - Current GPS
   - ZIP Code
3. Rank observing locations.
4. Launch Mission.

## Initial Database Tables
- locations
- missions
- observations
- equipment
- weather_cache
- usersettings

## Obsidian Export
Each mission exports to:

Obsidian/
└── NightOps/
    ├── Missions/
    ├── Locations/
    ├── Targets/
    └── Equipment/

Markdown files contain YAML frontmatter and Obsidian links for backlinks. Mission
exports create the linked `Locations/` note, and planned or recorded targets
create a canonical `Targets/` note plus a mission-scoped
`Missions/<mission>/Targets/` note with backlinks. Rewrites preserve existing
observation and user `## Notes` sections.

Canonical ownership remains with SQLite. Obsidian is a readable projection and
interoperability surface. Export is treated as a durable operation and uses
atomic file replacement so a partial write cannot corrupt a note.

## Current package boundaries

- `internal/domain`: mission and launch-site invariants
- `internal/console`: Bubble Tea screens, typed navigation, and command palette
- `internal/config`: YAML loading and validation
- `internal/storage/sqlite`: schema migrations and local persistence
- `internal/export/obsidian`: Markdown projection
- `internal/atlas`: versioned embedded launch-site catalog and nearby-site queries
- `internal/astronomy`: deterministic coordinate-based solar/lunar calculations
- `internal/weather`: provider/cache contracts and freshness validation
- `internal/targets`: versioned source-attributed offline celestial target catalog
- `internal/telescope`: optional Alpaca-compatible HTTP and Dwarf II WebSocket telescope adapters
- `internal/ai`: opt-in local mission-brief provider boundary
- `internal/sync`: versioned portable record bundles and newer-record merge policy
- `internal/api`: loopback-first HTTP projection for future local clients
- mission archive projections are loaded from SQLite on demand and remain read-only in the console
- `internal/plugins`: validated local manifest discovery; execution is outside the current contract
- equipment profiles are local SQLite aggregates associated with planned missions
- equipment items are persisted per profile; the inventory and readiness consoles
  report whether required user-recorded items exist before an operation
- `internal/app`: process composition and lifecycle

The initial Bubble Tea experience is composed inside `internal/console` from
boot, launch, form, mission-planning, status, footer, and command-palette consoles. A typed
`Route` on the root model owns navigation and retained child models. It receives
profile and capability status through `console.Options`; it does not open
databases or call providers. This keeps the launch surface replaceable while
preserving clean architecture.

The command palette opens with `Ctrl+K`. Its commands are generated from the
same capability statuses used by the visible consoles, so unavailable
integrations cannot appear as dead actions. Commands transition the root route
directly and preserve the originating route for Esc navigation.

SQLite backups are created through the storage boundary with `VACUUM INTO` and
an atomic temporary-file replacement. Settings exposes the backup form only
when the store is initialized; the backup destination is owner-readable and
the active database path is rejected.

Telescope control is an explicit application port. Alpaca uses the standard
HTTP protocol; Dwarf II uses its documented local WebSocket JSON command. Both
are disabled by default and NightOps does not probe or move a device during
startup. When configured, the UI reports `STANDBY` and exposes a slew action
only after a real offline catalog target has been selected. Dwarf slews also
require actual latitude and longitude from the mission origin; no coordinates
are guessed. Provider failures become actionable errors.

AI mission briefs use a separate provider port and are disabled by default. The
initial adapter targets a local Ollama-compatible endpoint. Startup never
contacts it; Mission Planning sends only the factual origin, target, astronomy,
weather, forecast, route, and equipment fields already displayed by the UI.
Provider output stays in the active planning model and failures become an
error route.

The Mission Archive is a read-only SQLite projection. Launch exposes it as a
separate action, and selecting a record opens a detail console without mutating
mission lifecycle state. The archive reloads from storage each time it opens so
newly created missions are visible during the same process.

Sync bundles are versioned JSON snapshots of launch sites, missions,
observations, equipment profiles, and inventory. Settings can export or import
them locally. Import is a merge: stable IDs identify records, `updated_at`
determines whether an incoming record is newer, older records are skipped, and
no local record is deleted. Current bundles include a stable source-device
identity; equal-timestamp divergent records are retained locally and reported
as conflicts instead of being silently overwritten. A legacy version-1 bundle
can still be read without a source identity. This is the interchange boundary
for future mobile and API clients.

The optional API exposes `/api/v1/status`, `/api/v1/missions`,
`/api/v1/missions/{id}`, and
`/api/v1/sync`. It binds to loopback by default, does not require startup
network access, bounds sync request bodies, and refuses remote binding unless
`allow_remote` is explicitly enabled with `auth_env` configured. Remote
API requests must present the corresponding Bearer credential; static companion
assets remain public so a remote user can load the credential-entry shell.
Loopback defaults remain unauthenticated for local clients.
The same listener serves the embedded responsive companion at `/companion/`;
its shell is cacheable, while mission data remains API-owned and is only
displayed from a prior local cache when the API is offline.

Mission Planning confirmation calls the application `MissionPlanner` service.
That service creates a launch-site aggregate and a planned mission, persists
both through SQLite ports, and projects the result to Obsidian when configured.
ZIP origins retain NULL coordinates until a real geocoder supplies them; the
system never substitutes fabricated coordinates.
Mission Planning calculates the live observing window from the current run's
astronomical darkness and the selected target visibility intervals. It stores
the resulting UTC timestamps in SQLite and renders them in the origin timezone.
Atlas-selected origins use their catalog timezone, while ZIP origins use the
configured profile timezone. An advanced manual override remains available for
exceptional cases, but normal sessions do not ask the operator for a date or
time.

The first Atlas data slice is intentionally local and bounded: it embeds
source-attributed Austin-area observing fields, including coordinates, timezone,
and Bortle metadata. When Atlas is enabled, an imported validated CSV catalog
is stored in SQLite as the active offline catalog; startup prefers that catalog
and falls back to the embedded starter data when no import exists. Settings
exposes the import flow only when the Atlas capability is enabled, and the
console only exposes browsing when a catalog loads successfully. Weather is
modeled separately as an expiring snapshot and is
stored in `weather_cache`. The optional Open-Meteo adapter supplies current
temperature and cloud-cover observations. Startup checks the local cache first,
refreshes only when weather is enabled and coordinates are known, and reports
`STANDBY` when an expired cached result is available during an upstream failure.

ZIP geocoding is a separate optional provider boundary. Its results are cached
locally before they enter mission planning; an unavailable provider leaves the
origin coordinate-less rather than substituting a guessed centroid.
Mission Planning requests a summary through an application callback, so the
console remains independent of both SQLite and network clients. Weather
summaries include the source and explicitly label stale cache values. Hourly
forecast points are validated and stored as JSON in the local weather cache;
the planning view renders the next available points from that cache or a real
provider refresh. When points are available, the root route state machine
exposes a retained hourly forecast browser; when they are not available, that
action is omitted rather than leaving a dead screen. Darkness is calculated
locally from the same solar model used by astronomy planning. Cloud and
precipitation filters apply only when the provider returned those values.

Astronomy planning uses the local solar model to locate civil, nautical, and
astronomical twilight crossings. Missing crossings are represented as nil
transitions rather than fabricated times, preserving correct behavior for
high-latitude observing sites.

Route planning currently uses `internal/routing` to calculate a Haversine
straight-line distance between Home Base and a coordinate-backed launch site.
When `features.routing` is enabled, the composition root can use the
OSRM-compatible adapter and persist road distance and duration in `route_cache`.
Fresh cached provider results are preferred offline; provider failures fall back
to the geodesic plan. It does not invent driving distance or ETA when neither a
provider nor a cached result exists. The provider policy is bounded by a
configured request timeout, retry count, and exponential backoff. Only
transient transport and HTTP failures are retried. The route console is hidden
when either endpoint lacks coordinates.

The target planner uses fixed equatorial coordinates from the embedded target
catalog and calculates horizontal altitude against the selected launch site.
The initial window requires astronomical darkness and the configured minimum
target altitude (30° by default);
the target selector is hidden if the catalog cannot load, so it cannot expose a
dead action. When hourly forecast points exist, `internal/weather` ranks them
locally for the selected target using target altitude, darkness, cloud cover,
and precipitation. Missing provider values remain ineligible, and the console
shows the best qualifying point or the reason no point qualifies.

Equipment profiles are created from Settings and selected before mission
confirmation. Their stable ID is stored on the mission and included in the
Obsidian mission frontmatter; an unknown profile cannot be attached because the
application validates it through the equipment repository. Mission creation
also projects the selected profile and its recorded inventory into a reusable
`Equipment/` note and a mission-scoped `Missions/<mission>/Equipment/` snapshot.
Reruns replace generated content while preserving a user-authored `## Notes`
section. Settings also owns a
local inventory editor for profile items. Readiness is intentionally honest:
profiles with no required recorded items are `NOT CONFIGURED`; otherwise the
console reports the required items recorded in that profile. The system does
not claim that a physical item is present beyond what the user recorded.

After planning, the root console enters an Operation route. Launching and
activating the mission use domain lifecycle transitions; observations are
stored as first-class records, and completion enters a Debrief route. Obsidian
mission rewrites preserve the existing Flight Recorder section so status
updates cannot erase recorded observations. Location and target notes are
written atomically alongside the mission projection.

Future integrations must enter through application ports and remain disabled
behind feature flags until their contracts, failure behavior, and tests exist.

The initial plugin contract is deliberately metadata-only. With
`features.plugins` enabled, NightOps scans direct child directories of the
configured plugin directory for `manifest.yaml`, validates IDs, versions,
entrypoints, and declared capabilities, and exposes the result in the Settings
Plugin Registry. It never executes an entrypoint during discovery.

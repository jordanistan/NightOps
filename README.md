# NightOps

NightOps is an offline-first mission planner for amateur astrophotography. It
uses a Go/Bubble Tea TUI to turn a short target-selection session into a
complete Obsidian mission workspace.

## The live observing flow

1. Start NightOps and choose `LAUNCH MISSION`.
2. Select the observing origin: Home Base, ZIP, GPS, or Atlas site.
3. Select tonight's targets in capture order.
4. NightOps calculates the local mission date, astronomical darkness,
   per-target visibility windows, and a weather `GO`, `NO-GO`, or `UNKNOWN`
   assessment.
5. Choose `REVIEW & CREATE OBSIDIAN MISSION`. The review screen is a launch
   gate: inspect the generated mission and choose `EDIT MISSION DETAILS` if
   something needs changing.
6. Choose `LAUNCH MISSION + OPEN OBSIDIAN` or continue in the TUI.

No date or normal mission-window form is required. The current run and selected
targets provide the mission timing. An advanced manual override is available
only for exceptional planning cases.

## Quick start

```sh
make verify
make run
```

For a fresh device:

```sh
git clone git@github.com:jordanistan/NightOps.git
cd NightOps
make verify
make run
```

NightOps creates its local SQLite database, configuration, caches, logs, and
device identity under `~/.local/share/nightops`. These runtime files are not
part of the repository.

The default Obsidian workspace is `~/Documents/Obsidian/NightOps`. Startup
creates `~/Documents/Obsidian` and the `NightOps` vault directory, including
`.obsidian`, when they do not exist. A custom `obsidian.vault_dir` or
`obsidian.notes_dir` can override that location.

## Obsidian output

Final Review creates or updates a vault with this structure:

```text
NightOps/
├── Index.md
├── Missions/Index.md
├── Targets/Index.md
├── Locations/Index.md
└── Equipment/Index.md
```

Mission notes include auto-filled properties, launch-site details, current
weather, a repeatable equipment pre-flight checklist, and a target table with
capture guidance plus recommended starting settings. The long hourly forecast
is placed at the bottom of the note. Target notes retain source-attributed
reference summaries, reusable capture settings, and a mission-by-mission
history with links to each mission and its observing location.

When no custom name is supplied, missions are named with their local observing
date and start time, such as `Mission 2026-07-22 21-04-05`. The Mission Archive
lists the newest observing date first, and the Obsidian mission index keeps
date-stamped mission notes newest-first so past sessions are easy to find.

## Configuration

Copy [`config.example.yaml`](config.example.yaml) to the local configuration
path when a customized setup is needed. Fresh defaults enable weather,
geocoding, and target knowledge. Existing configurations retain explicit
feature settings.

Important sections:

- `origin` — Home Base ZIP, coordinates, and timezone.
- `weather` — Open-Meteo endpoint, cache duration, and forecast thresholds.
- `geocoding` — ZIP-to-coordinate provider and local cache policy.
- `target_knowledge` — Wikipedia-compatible reference provider.
- `obsidian` — vault directory and `NightOps` notes directory.
- `features` — opt-in adapters such as routing, telescope control, AI, and API.

If a configured Home Base has a ZIP but no coordinates, startup attempts to
resolve and persist the coordinates. If the provider is unavailable, the TUI
labels the missing data instead of inventing it.

## Repository map

- [`cmd/`](cmd/README.md) — executable entrypoints.
- [`internal/`](internal/README.md) — application and adapter packages.
- [`migrations/`](migrations/README.md) — canonical schema history.
- [`docs/`](docs/README.md) — architecture, development, and operator docs.
- [`repos/`](repos/README.md) — ecosystem repository planning notes.

## Verification

```sh
go test ./...
go vet ./...
make verify
git diff --check
```

The repository must not contain local databases, generated vault content,
configuration secrets, API tokens, or provider credentials.

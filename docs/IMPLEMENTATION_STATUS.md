# NightOps MVP Implementation Status

Updated: 2026-07-22

This document is the execution ledger for the repository MVP. A requirement is
marked complete only when its user workflow, persistence boundary, failure path,
tests, and documentation are present.

## Current baseline

The repository is a working Go/Bubble Tea modular monolith with SQLite as the
canonical local store. The first mission workflow is substantially implemented:
boot, origin selection, mission planning, lifecycle, observations, debrief
screen, Obsidian projection, Atlas, optional providers, sync, API, and CLI.

## Requirement audit

| Requirement | Status | Evidence or remaining work |
| --- | --- | --- |
| Boot animation and skip | Complete | Typed boot route, Bubble Tea tests |
| Home Base setup and persistence | Complete | Config persistence and failure-path tests |
| ZIP validation and planning | Complete | Async optional geocoding, cache, unknown fallback |
| Current-location acquisition | Partial | Real adapter port and honest unavailable state exist; no platform GPS adapter is bundled |
| SkyBase Atlas browse/select | Complete | Versioned local catalog, import/export, provenance tests |
| Mission planning and SQLite mission persistence | Complete | Application service, repositories, restart-compatible config/data paths |
| Ordered multi-target mission plans | Complete for local catalog targets | Multiple catalog objects can be selected in order, persisted in SQLite, and exported into mission-scoped Obsidian target notes |
| Guided mission review | Complete | Retained review route groups origin, ordered targets, observing window, equipment, and known conditions before the launch write; back navigation preserves planning state |
| Visible route/action capability audit | Complete for audited slice | Deep Space no longer advertises telescope slew without an adapter; disabled Atlas import/export and empty inventory controls are gated or described honestly |
| Launch countdown and Deep Space console | Complete for local mission launch | First console is a single highlighted Launch Mission action; launch enters a 10-to-0 sequence and then an interactive target console |
| Mission archive/detail | Complete | Read-only SQLite projections and UI tests |
| Operation launch/activate/complete lifecycle | Complete | Domain transitions and persisted mission timestamps |
| Observation/Flight Recorder | Complete | SQLite observations, local image imports, Obsidian embeds, and preservation tests |
| User-authored debrief persistence | Complete | Completed missions accept validated summaries, persist them in SQLite, reload them, and export an idempotent Obsidian section |
| Obsidian mission/location/target/equipment links | Complete | Mission, location, target, reusable equipment, and mission-scoped equipment notes are deterministic, linked, and rerunnable |
| Automated Obsidian mission knowledge template | Complete for catalog targets | Startup-warmed SQLite target knowledge, cache-first target projection, detailed weather/equipment/target sections, target mission history, image links, and Final Review vault-open action |
| Offline astronomy calculations | Complete | Deterministic local solar/lunar/twilight/visibility tests |
| Weather and hourly forecast | Complete when enabled | Open-Meteo adapter, online-first fresh defaults, cache-first behavior, stale/error states, tests |
| Offline routing | Complete | Geodesic facts; optional cached OSRM road route |
| Equipment profiles/inventory/readiness | Complete | SQLite persistence and UI failure-path tests |
| Telescope control | Complete when configured | Alpaca and Dwarf adapters are opt-in and tested |
| AI mission brief | Complete when configured | Local Ollama boundary, factual input, bounded failure path |
| Sync/export/import | Complete | Versioned bundles, stable device IDs, conflict reporting |
| Local API/CLI/companion | Complete for archive/sync scope | Loopback-first API, auth boundary, responsive companion |
| Plugin discovery | Partial by design | Metadata-only; sandboxed execution is not part of the MVP contract |
| Documentation and verification | Complete for current slice | Agent contract, decision log, status ledger, tests, and verification commands are synchronized |

## Implementation order

1. Improve the GPS adapter boundary only when a real platform adapter can be
   provided without fabricated coordinates.
2. Complete release documentation and perform a requirement-by-requirement
   final audit.

## Known limitations

- GPS acquisition is unavailable unless a real adapter is injected; NightOps
  does not fabricate coordinates.
- Fresh installations enable weather and ZIP geocoding for online-first
  enrichment; existing persisted configurations retain their explicit feature
  settings.
- ZIP geocoding depends on an enabled Nominatim-compatible provider or a prior
  local cache; failure leaves coordinates unknown.
- Plugin code is discovered as metadata but is not executed.
- Mobile/cloud ecosystem work beyond the embedded archive/sync companion is
  outside the current MVP.

## Verification ledger

The required local gate is `go test ./...`, `go vet ./...`, and builds of both
`nightops` and `nightopsctl`. Generated `bin/`, database, log, environment, and
temporary files must be removed before handoff. The debrief and equipment
projection slices have targeted tests for domain validation, SQLite round trips
and updates, Bubble Tea input and failure paths, application persistence, and
idempotent Obsidian output.
The final full-gate result is recorded below after execution.

## Latest verification

2026-07-22: passed `gofmt -w cmd internal`, `go test ./...`, `go vet ./...`,
both executable builds, and the repository `make verify` target after adding
the cache-first Obsidian knowledge workflow. Target-provider fixtures, SQLite
knowledge round trips, rich mission projections, target history, and the
vault-open TUI action are covered. `git diff --check` passed and generated
executables were removed after verification.

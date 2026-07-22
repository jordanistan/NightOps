# NightOps Architecture Review

Status: pre-implementation review  
Date: 2026-07-21

This review covers `README.md`, `docs/Architecture.md`, `docs/Roadmap.md`,
`docs/Website.md`, and `repos/RepoTree.txt`. The documentation establishes a
strong product direction, but it is not yet an implementation specification.

## Executive assessment

NightOps should begin as an offline-first modular monolith: one Go desktop
binary, one local SQLite database, and a small set of stable domain and port
interfaces. This gives the project a dependable product quickly while keeping
the core reusable by a future CLI, mobile client, API, and sync service.

Do not begin with separate services, cloud accounts, or a marketplace. Those
systems add operational and security costs before the mission model is proven.
The important early investment is durable local data, deterministic planning,
and explicit boundaries.

## What is strong already

- The product has a memorable mission-centered vocabulary and clear purpose.
- Offline-first, SQLite, and Obsidian export are meaningful differentiators.
- The proposed concepts—locations, missions, observations, equipment, and
  weather—are a sensible first domain slice.
- The roadmap identifies valuable integrations without making all of them
  Milestone 1 requirements.

## Weaknesses and missing decisions

1. The mission lifecycle is undefined. Specify draft, planned, launched,
   active, paused, completed, cancelled, and archived states, including which
   transitions are legal.
2. “Current GPS,” ZIP lookup, and weather behavior are undefined offline.
   Every external result needs a provider, timestamp, accuracy, freshness, and
   an explicit stale/unknown state.
3. The schema lacks stable IDs, timestamps, units, provenance, soft deletion,
   migrations, and versioning. These are prerequisites for sync and recovery.
4. Obsidian export has no conflict, retry, atomic-write, or vault-change policy.
   Export must be a durable outbox operation, not a best-effort side effect.
5. “Usersettings” assumes one local user but the roadmap later introduces
   multi-user sync. Separate local installation settings from user-owned data.
6. Atlas governance is missing: license, attribution, source imports, spatial
   queries, moderation, versioning, and update distribution need owners.
7. Plugin architecture and AI are named but have no trust model, capability
   permissions, compatibility policy, or privacy boundary.
8. The TUI design needs accessibility, keyboard-only navigation, terminal-size
   behavior, color fallback, and error presentation requirements.
9. Testing and release policy are absent. Define deterministic astronomy
   calculations, migration tests, property tests for mission transitions, and
   fixture-based provider tests before feature growth.

## Recommended architecture

Use a dependency rule in which UI and adapters depend inward, while domain
logic depends on neither Bubble Tea nor SQLite:

```text
cmd/nightops
  -> app (composition, lifecycle, configuration)
  -> console (Bubble Tea models and navigation)
  -> application (use cases and ports)
  -> domain (missions, sites, targets, observations, equipment)
  <- adapters (sqlite, atlas files, GPS, weather, Obsidian, logging)
```

Suggested repository shape:

```text
cmd/nightops/
internal/
  app/                 composition root and process lifecycle
  domain/              entities, value objects, invariants
  application/         use cases and interfaces
  console/              Bubble Tea screens, commands, keymap
  storage/sqlite/      migrations and repositories
  atlas/               import, query, provenance, update packages
  providers/            GPS, ZIP, weather, astronomy integrations
  export/obsidian/     renderer and durable export worker
  platform/             filesystem, clock, terminal, OS integration
  config/               YAML loading, defaults, validation
  featureflags/         local capability evaluation
  plugins/              versioned, permissioned extension boundary
docs/
migrations/
testdata/
web/
```

Keep `domain` free of infrastructure. Use constructor injection from a single
composition root. Start with interfaces at application boundaries, not for
every struct. A modular monolith is the right default; packages can later be
extracted only when a real deployment or ownership boundary exists.

## Foundational data model

Use UUIDs (or another documented stable identifier), UTC timestamps, explicit
units, and schema migrations from the first commit. Initial aggregates should
be `Mission`, `LaunchSite`, `Target`, `Observation`, `EquipmentProfile`, and
`ExportJob`. Add `WeatherSnapshot`, `AstronomySnapshot`, and `SyncMetadata`
with provenance rather than storing mutable “current” values on missions.

Treat a mission plan as a snapshot of its inputs. If a forecast or target
catalog changes later, the completed mission remains historically accurate.

## Offline-first rules

- Local writes are authoritative and must succeed without network access.
- External providers are adapters with timeouts, cached responses, and typed
  unavailable/stale results.
- Every user-visible operation has a recoverable error and retry path.
- Export jobs use an outbox with idempotency keys and atomic file replacement.
- Backups are an explicit product feature: portable database backup plus export
  manifest and schema version.
- Sync, when introduced, must reconcile records by stable IDs and revisions;
  never silently overwrite local observations.

## Ecosystem strategy

Define a versioned core contract now, but implement one client first:

- `nightops-core`: domain and application packages reused by desktop and CLI.
- `nightops-desktop`: the initial Bubble Tea client.
- `nightops-cli`: later, a thin automation and import/export client.
- `skybase-atlas`: independently versioned data distribution and tooling.
- API/cloud/mobile: future adapters around the same contracts, not alternate
  business logic.
- Plugins/themes: later extension points with manifests and compatibility
  versions; themes should be declarative and unable to execute code.
- AI: an optional planner assistant that proposes plans and records its input,
  model, and output; it must never silently mutate a mission.

The Obsidian vault is an interoperability surface, not the primary database.
NightOps owns canonical records; Markdown is a readable, linkable projection.

## 12-month roadmap

### Months 1–2: foundation

Lock the domain glossary and mission state machine. Create the Go module,
configuration validation, structured logging, SQLite migrations, repositories,
application state, Bubble Tea navigation, theme tokens, and CI. Add backup and
diagnostic commands before integrations.

### Months 3–4: first complete mission

Implement launch-site selection (manual/home base first), target and equipment
models, mission planning, operation logging, debrief, search, and reliable
Obsidian export. Ship with fixtures, migration tests, and a documented release
process.

### Months 5–6: astronomy and local atlas

Add coordinates, time zones, twilight, moon, altitude/visibility calculations,
and a versioned Austin starter atlas. Make all calculations reproducible from a
recorded time and location. Add import validation and attribution.

### Months 7–8: providers and usability

Add GPS, ZIP geocoding, weather caching, stale-data UI, command palette,
keyboard navigation, accessibility checks, and terminal-size resilience.

### Months 9–10: extensibility

Specify plugin and theme manifests, capability permissions, compatibility rules,
and the CLI. Prototype telescope integration behind a feature flag.

### Months 11–12: ecosystem readiness

Define sync semantics and an API boundary without committing to hosted service
operations. Publish developer docs, atlas contribution rules, security policy,
release artifacts, telemetry-free diagnostics, and a community feedback loop.

## Risk assessment

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Unclear mission model | Rework across every client | Specify invariants and transitions first |
| Data loss or export conflicts | Loss of user trust | Transactions, outbox, backups, restore tests |
| Fragile external providers | Broken offline experience | Ports, cache, stale states, fixtures |
| Atlas licensing/provenance | Legal and data-quality problems | Source manifests, attribution, review workflow |
| Premature ecosystem scope | Slow, fragmented delivery | Modular monolith and one complete client |
| Unsafe plugins or AI | Privacy/security incidents | Permissions, sandboxing, audit trail, opt-in |
| Astronomy inaccuracies | Bad observing decisions | Published algorithms, reference fixtures, uncertainty |
| TUI-only usability gaps | Adoption barrier | Accessibility and terminal compatibility requirements |

## CTO recommendation

Approve the vision and revise the architecture documents before coding. The
next decision record should settle the mission lifecycle, canonical data model,
time/coordinate/unit conventions, backup guarantees, Obsidian conflict policy,
and Atlas licensing. Once those are explicit, implement the foundation as a
small vertical slice: create a launch site, plan a mission, persist it, export
it, reopen it offline, and complete a debrief.

NightOps will become “the best” through trust and depth: correct astronomy,
excellent records, fast offline workflows, transparent provenance, and a
community atlas that is easier to contribute to than competing tools. The
ecosystem should amplify that core rather than distract from it.

## Decisions requiring explicit approval

Before implementation, the project owner should approve or revise:

- modular monolith versus multiple repositories in the first year;
- canonical ownership of data between NightOps, Atlas, and Obsidian;
- mission states and whether completed missions are immutable;
- backup/restore and export guarantees;
- coordinate, time zone, unit, and astronomy-library policy;
- plugin execution model and AI privacy policy;
- Atlas data sources, license, moderation, and update cadence.

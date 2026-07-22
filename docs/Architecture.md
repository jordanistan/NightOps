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

Markdown files contain YAML frontmatter for backlinks.

Canonical ownership remains with SQLite. Obsidian is a readable projection and
interoperability surface. Export is treated as a durable operation and uses
atomic file replacement so a partial write cannot corrupt a note.

## Current package boundaries

- `internal/domain`: mission and launch-site invariants
- `internal/console`: Bubble Tea screens and navigation
- `internal/config`: YAML loading and validation
- `internal/storage/sqlite`: schema migrations and local persistence
- `internal/export/obsidian`: Markdown projection
- `internal/app`: process composition and lifecycle

Future integrations must enter through application ports and remain disabled
behind feature flags until their contracts, failure behavior, and tests exist.

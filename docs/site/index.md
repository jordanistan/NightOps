# NightOps

## Mission planning for amateur astronomers

NightOps is an offline-first mission planner for nights under the stars.

It combines a calm mission-control console, SQLite durability, SkyBase Atlas
location data, deterministic astronomy calculations, online-first weather and
ZIP enrichment with cached fallback, optional routing providers, and Obsidian
projections.

> Every night under the stars is a mission worth remembering.

## Design promises

- Local data remains usable without internet access.
- SQLite is the canonical owner of mission records.
- Network enrichment is bounded by timeouts and falls back to local cache or an
  honest unavailable state.
- Unknown weather, coordinates, and recommendations remain unknown.
- Imports validate completely before replacing or merging local data.
- Visible actions either work or are hidden behind their capability flag.

Start with [Installation](installation.md), then plan your [First Mission](first-mission.md).

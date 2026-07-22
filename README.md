# NightOps — Powered by SkyBase

**Mission planning for amateur astronomers.**

Motto:
> Every night under the stars is a mission worth remembering.

## Repository Layout

- `cmd/nightops/` — Go + Bubble Tea application entrypoint
- `internal/` — domain, console, configuration, storage, and export packages
- `migrations/` — human-readable database schema history
- `docs/` — architecture, development, roadmap, and website documentation
- `repos/` — ecosystem repository planning

## Phase 1
- Production foundation and first vertical slice
- Animated launcher and Mission Origin console
- Home Base, ZIP Code, and Current GPS origin paths
- SQLite migrations and durable mission model
- Obsidian Markdown projection

## Current status

The repository contains the initial Go foundation: configuration validation,
SQLite initialization, mission lifecycle rules, console navigation, theme
tokens, feature flags, atomic Obsidian export, tests, CI, and development
documentation. Network, atlas, weather, telescope, plugin, and AI systems are
not enabled until their contracts are specified.

Start locally with:

```sh
make verify
make run
```

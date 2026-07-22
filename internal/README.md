# Internal packages

The `internal/` tree contains the application implementation and is intentionally
not importable by external Go modules.

## Boundaries

- `domain/` — durable mission, target, equipment, observation, and debrief rules.
- `application/` — mission use cases and ports.
- `console/` — Bubble Tea routes, screens, and user interactions.
- `storage/sqlite/` — canonical SQLite persistence and migrations.
- `export/obsidian/` — atomic Markdown projection into an Obsidian vault.
- `providers/` — bounded HTTP/WebSocket adapters for external services.
- `astronomy/` and `weather/` — deterministic visibility and forecast ranking.
- `sync/` — versioned offline bundles and conflict-safe merges.
- `app/` — runtime composition and provider wiring.

Keep provider access behind ports, preserve source attribution, and never
fabricate missing coordinates, weather, target facts, or equipment data.

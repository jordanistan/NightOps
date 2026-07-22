# NightOps commands

This directory contains the executable entrypoints for the project.

## `nightops`

The primary Bubble Tea TUI. It loads configuration, opens the local SQLite
store, initializes enabled providers, and starts the mission-control console.

```sh
go run ./cmd/nightops
go run ./cmd/nightops --config /path/to/config.yaml
```

## `nightopsctl`

The local companion CLI for API status, mission inspection, sync export/import,
and validated Atlas contribution workflows. It does not access SQLite directly;
it talks to the configured local API or works with explicit bundle paths.

```sh
go run ./cmd/nightopsctl --help
```

Build both executables with `make verify`.

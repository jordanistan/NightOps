# Installation

## Requirements

- Go 1.26 or newer
- A terminal with at least 80×24 recommended for the compact console
- SQLite is embedded; no database server is required

## Build and verify

From the repository root:

```sh
make verify
```

This formats Go code, runs tests and vet, and builds both `nightops` and
`nightopsctl` into the ignored `bin/` directory.

## Run locally

```sh
make run
```

NightOps uses safe defaults. To customize it, copy
`config.example.yaml` to a user-owned path outside the repository and start:

```sh
go run ./cmd/nightops -config /path/to/config.yaml
```

If the flag is omitted, NightOps reloads the persisted configuration at
`<data_dir>/config.yaml` when it exists, so Settings changes such as Home Base
survive a restart.

Do not commit the user configuration or local database. The repository ignores
`config.yaml`, SQLite files, logs, `.env`, and common editor directories.

# Development

NightOps is intentionally a small modular monolith at this stage. The core
domain must remain independent of Bubble Tea, SQLite, network providers, and
the filesystem.

## Local setup

Install Go 1.26 or newer, then run:

```sh
cp config.example.yaml config.yaml
make verify
make run
```

The default database is created under `~/.local/share/nightops`. Configuration
is loaded only when `--config config.yaml` is supplied; this keeps a fresh
checkout runnable without a local secret or machine-specific file.

## Package rules

- `internal/domain` owns entities, value objects, and invariants.
- `internal/application` will own use cases and ports as the first vertical
  slice grows.
- `internal/console` owns Bubble Tea models and user interaction only.
- `internal/storage` owns SQLite and migrations.
- `internal/providers` will own network or hardware adapters.
- `internal/export` owns projections such as Obsidian Markdown.
- `internal/app` is the composition root and the only place that wires
  concrete implementations together.

Use UTC for persisted timestamps and retain the source/timezone alongside
astronomy and provider results. Do not store mutable forecast data directly on
completed missions.

## Verification expectations

Every change should pass `make verify`. Domain tests should cover state-machine
invariants, storage tests should use temporary or in-memory databases, and
provider tests should use deterministic fixtures. UI behavior should be
testable without requiring a live terminal.

The current build sets `-buildvcs=false` because source archives and exported
working trees may not contain Git metadata. This keeps reproducible local and
CI builds independent of checkout state.

## Commit practice

Keep commits narrow and descriptive. Main should always build, test, and start.
Incomplete integrations belong behind feature flags and must have a documented
reason for being disabled.

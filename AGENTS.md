# NightOps Development Instructions

## Mission

Build NightOps as a functional, local-first astronomy mission planner. A
screen, interface, adapter, or package is not complete unless its user
workflow functions.

## Source of truth

Read these before making changes:

1. `README.md`
2. `ARCHITECTURE_REVIEW.md`
3. everything under `docs/`
4. existing tests
5. existing implementation

When documentation and implementation disagree, follow documented product
intent unless a newer explicit decision in `docs/DECISIONS.md` supersedes it.
Track current coverage and gaps in `docs/IMPLEMENTATION_STATUS.md`.

## Engineering behavior

- Work autonomously on ordinary implementation decisions.
- Prefer complete vertical workflows over disconnected scaffolding.
- Fix root causes rather than adding state-specific patches.
- Do not fabricate external data.
- Do not expose nonfunctional menu items; gate incomplete capabilities.
- Preserve user data and keep documentation synchronized with implementation.
- Record significant technical decisions in `docs/DECISIONS.md`.

## Completion definition

A feature is complete only when its UI is reachable, actions work, navigation
into and out of it works, input is validated, errors are useful, relevant data
is persisted, tests cover primary and failure paths, and documentation matches
behavior.

## Required verification

Before completing a substantial task, run:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build -o bin/nightops ./cmd/nightops
```

Also run `make verify` when the target exists. Keep generated binaries and
machine-local data out of the repository after verification.

## Architecture

Maintain separation between domain, application, ports, adapters, persistence,
providers, console, configuration, Atlas, and Obsidian export. Bubble Tea views
must not directly perform database, provider, or filesystem business logic.
Use dependency injection, explicit typed routes, and persistent child models.

## Product honesty

Never show a capability as ready unless it is initialized and usable. Use
accurate states such as Ready, Disabled, Not configured, Unavailable, Loading,
Error, Unknown, and Unverified. Never invent coordinates, weather, Bortle
ratings, safety claims, access hours, or observing recommendations.

## User experience

No visible action may be a dead end. Every screen needs an obvious forward or
back route, useful empty/loading/unavailable/error states, visible focus, and
support for arrows, j/k, Enter, Esc, help, settings, and quit where relevant.
Handle terminal resizing and compact layouts.

## Git behavior

Do not rewrite history or perform destructive Git operations. Make focused
commits when repository access permits.

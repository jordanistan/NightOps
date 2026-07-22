# Storage adapters

Storage owns durable local state. SQLite is the canonical source of truth for
missions, launch sites, selected targets, target knowledge, equipment,
weather/route caches, observations, debriefs, and sync records.

The [`sqlite/`](sqlite/) package provides the runtime store, transactions,
backups, and embedded migrations. Use temporary or in-memory databases in
tests. Runtime databases belong under the user's data directory and must never
be committed.

# SQLite migrations

This directory is the human-readable copy of the NightOps SQLite schema
history. The embedded runtime migrations under
[`internal/storage/sqlite/migrations/`](../internal/storage/sqlite/migrations/)
are the files loaded by the application; keep both copies synchronized.

Migrations are append-only. Add a new numbered SQL file for schema changes,
make it safe for a fresh database, and add a storage test covering the new
behavior. Do not rewrite a migration that may already have run on a user's
database.

The SQLite database is canonical. Obsidian Markdown is a generated projection
and must not become a second source of truth.

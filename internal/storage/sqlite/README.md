# SQLite store

`Store` implements the application persistence ports and applies the embedded
numbered migrations when opened. The store is deliberately local-first:
network providers enrich cached records, but a provider outage must not destroy
or fabricate local mission data.

When changing schema:

1. Add the next numbered migration in this directory.
2. Mirror it under the repository-level `migrations/` directory.
3. Add round-trip and migration coverage in `store_test.go`.
4. Preserve stable IDs and existing records.

Use `Store.Backup` for consistent owner-readable backups. Never place a live
`nightops.db` in the repository.

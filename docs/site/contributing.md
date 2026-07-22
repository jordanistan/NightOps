# Contributing

NightOps is intended to become a long-lived open-source project. Before
opening a change:

1. Read the architecture and development guides.
2. Preserve clean package boundaries and existing user data.
3. Gate incomplete capabilities rather than exposing dead actions.
4. Add tests for state transitions, persistence, provider failures, or import
   validation as appropriate.
5. Update the roadmap and documentation for major features.
6. Run `make verify` and the repository secret/artifact scans.

Do not commit local YAML configuration, SQLite databases, logs, credentials, or
generated build output. Keep provenance attached to imported astronomy data.

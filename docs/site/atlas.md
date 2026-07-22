# SkyBase Atlas

SkyBase Atlas is the location intelligence layer. The current implementation
ships with a small source-attributed offline starter catalog and supports local
CSV imports.

Every imported row must include:

- stable ID and name
- latitude and longitude
- IANA timezone
- source provenance
- optional Bortle class from 1 through 9

The entire file is validated before it becomes the active local catalog. A
failed import leaves the previous catalog unchanged.

## Community contributions

Validate or package a local catalog for review:

```sh
go run ./cmd/nightopsctl -atlas-version community-2026-07 \
  atlas-validate ./sites.csv
go run ./cmd/nightopsctl -atlas-version community-2026-07 \
  atlas-package ./sites.csv ./nightops-atlas-contribution.json
```

Contribution packages are explicitly marked `unreviewed`. They are shareable
review artifacts and are never silently promoted into the active catalog.

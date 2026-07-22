# FAQ

## Does NightOps require internet access?

No. Mission planning, SQLite persistence, the starter Atlas, astronomy
calculations, and cached records work offline. Fresh installations try online
weather and ZIP enrichment first; road routing, AI, GPS, and telescope control
remain optional integrations.

## Does NightOps invent missing values?

No. Missing coordinates, forecast fields, travel times, and provider results are
shown as unavailable or not configured.

ZIP geocoding is online-first on fresh installations. When enabled, successful results are cached locally
for offline reuse; if resolution fails, the ZIP mission remains usable with
coordinates explicitly marked unknown.

## Can an import delete my data?

No. Sync imports use stable IDs and newer-records-win merging. Older records are
skipped, equal-timestamp divergent records are reported as conflicts, and no
local record is deleted.

## Where is the database?

By default it is under the configured application data directory. SQLite is the
canonical owner; Obsidian and sync files are projections or interchange
artifacts.

## What is the local API?

When enabled, it serves a loopback-first archive and sync API plus the responsive
companion at `/companion/`. Remote binding requires explicit authentication
configuration. The companion accepts that credential in an in-memory field; it
is not stored in browser storage or sent anywhere except the configured local
API.

## Can I change the console palette?

Yes. Set `app.theme` to `mission-control` or `observatory` in the user-owned
YAML configuration. Unsupported theme names are rejected before startup.

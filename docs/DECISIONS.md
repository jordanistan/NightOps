# NightOps Decisions

## D-001: SQLite owns user data

SQLite is the canonical owner of missions, launch sites, observations,
equipment, and local caches. Obsidian and sync bundles are projections or
interchange artifacts. This keeps offline writes durable and makes exports
rebuildable.

## D-002: Unknown beats fabricated

When GPS, geocoding, weather, routing, astronomy inputs, or Atlas provenance
are unavailable, the UI uses Unknown, Unavailable, or Not configured. It never
substitutes a guessed coordinate, forecast, safety claim, or recommendation.

## D-003: Optional integrations are capability-gated

Hardware, AI, plugin, and routing providers remain disabled by default. Weather
and ZIP geocoding are enabled for fresh installations as bounded online-first
enrichment, with local cache and honest failure states. All providers enter
through application ports, and a visible action must not exist unless its
capability is wired.

## D-004: Typed route ownership

The root Bubble Tea model owns a typed route and retained child models. Screen
models own interaction state; the composition root injects persistence and
provider callbacks. This prevents route changes from being overwritten by
independent boolean flags or reconstructed child models.

## D-005: Conflict-safe synchronization

Sync uses stable IDs and newer-records-win semantics. Equal-timestamp divergent
records retain the local value and surface conflict IDs. Imports never delete
local records.

## D-006: Debrief is a durable user record

Completing an operation is not itself a debrief. The Debrief console must collect
user-authored content, persist it locally, and project it to Obsidian without
overwriting the Flight Recorder.

## D-007: Online-first opportunistic enrichment

Fresh installations attempt configured weather and ZIP geocoding providers
when coordinates or a postal origin are needed. Provider failures fall back to
validated local cache or an explicit unknown state. Network availability is not
treated as GPS: NightOps requires a real location adapter for coordinates and
never derives latitude/longitude from the mere presence of Wi-Fi or 5G.

# Architecture

NightOps is a modular Go monolith organized around Clean Architecture:

```text
Console → Application services → Domain ports → SQLite / providers / exports
```

The Bubble Tea console owns interaction and typed routes. Application services
own mission use cases. Domain types own invariants. SQLite owns durable state.
Providers for weather, routing, AI, GPS, and telescopes are optional adapters.
Obsidian, sync bundles, the local API, and the responsive companion are
projections or client boundaries rather than alternate database owners.

The repository also contains a longer architecture review for maintainers;
this page is the published overview and source-of-truth concepts remain the
same.

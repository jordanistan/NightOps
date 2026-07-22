# Obsidian exporter

The Obsidian exporter turns a persisted mission projection into a readable
NightOps vault. It creates mission, target, location, and equipment notes plus
the related `Index.md` files.

Generated mission content includes:

- YAML properties and the live mission date/window.
- Launch-site coordinates and linked location note.
- Current conditions and hourly weather forecast.
- Equipment checklist and readiness context.
- Ordered targets, visibility windows, capture guidance, and reference links.
- Target mission history and representative image links when available.

Writes are atomic. Existing user-authored notes and recorder sections must
survive re-export. The exporter receives data from the application projection;
it does not independently invent or fetch facts.

# Obsidian exporter

The Obsidian exporter turns a persisted mission projection into a readable
NightOps vault. It creates mission, target, location, and equipment notes plus
the related `Index.md` files.

Generated mission content includes:

- YAML properties and the live mission date/window.
- Launch-site coordinates and linked location note.
- Current conditions and an hourly weather forecast placed at the bottom of the
  mission note.
- A static pre-flight equipment checklist plus configured equipment context.
- An ordered target table with visibility context, capture guidance, reusable
  starting settings, and reference links.
- Target mission history and representative image links when available.

Writes are atomic. Existing user-authored notes and recorder sections must
survive re-export. The exporter receives data from the application projection;
it does not independently invent or fetch facts.

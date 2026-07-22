# Obsidian

Obsidian is an interoperability projection, not the canonical database.
NightOps keeps mission state in local SQLite and writes linked Markdown notes
using atomic replacement.

The vault projection contains:

```text
NightOps/
├── Index.md
├── Missions/
│   ├── Index.md
│   └── <Mission>.md
├── Locations/
│   └── Index.md
├── Targets/
│   └── Index.md
└── Equipment/
    └── Index.md
```

The normal dark-session workflow is intentionally short:

1. Start NightOps before observing. Startup warms the local SQLite target
   knowledge cache from the configured Wikipedia-compatible endpoint.
2. Choose `LAUNCH MISSION`. NightOps immediately asks for the ordered targets;
   select them, then choose
   `REVIEW & CREATE OBSIDIAN MISSION`, then inspect the final summary. Use
   `EDIT MISSION DETAILS` to return to planning, or choose `LAUNCH MISSION +
   OPEN OBSIDIAN`. `LAUNCH MISSION + CONTINUE IN NIGHTOPS` stays available when
   you want to remain in the TUI.
3. NightOps writes the mission, location, static equipment checklist, detailed
   weather summary, a reusable target/settings table, reference summaries, and
   image links before opening the configured vault directory. The hourly
   forecast is deliberately placed at the bottom so actionable mission
   information stays near the top.

The default vault is `~/Documents/Obsidian/NightOps`. NightOps creates the
parent `Obsidian` directory, the `NightOps` vault directory, and `.obsidian`
metadata on startup if they are missing. The Final Review action opens the
`NightOps` vault itself, not its parent directory.

Mission frontmatter auto-fills the mission name, status, launch-site link,
mission date, live-session flag, calculated dark-sky window, creation and update
timestamps, and equipment profile. The mission date is the local date at the
current run; the window is calculated from astronomical darkness and the
selected targets. No date or time form is required.

Mission notes include launch-site facts, all cached hourly weather values,
equipment checkboxes, ordered target links, per-target visibility windows,
capture guidance, recommended starting settings, and mission-scoped
target/equipment notes. Each target note retains its source summary,
representative image link, capture guidance, reusable capture settings, and a mission history line with
mission, status, location, and date. Reusing a target adds another history line
without deleting previous links. SQLite is the cache and source of truth;
Obsidian is the readable projection. Missing live data is labeled unavailable
and prior cached material is retained.

Generated mission names include the local mission date and start time. The
mission archive and `Missions/Index.md` are ordered newest-first, making recent
captures immediately visible while preserving links to older sessions.

Settings `o Open Obsidian Vault` and the command palette perform the same local
open action. The Final Review open action appears only when the configured vault
directory exists. Mission persistence continues locally if Obsidian is disabled.

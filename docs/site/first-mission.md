# Using NightOps

NightOps is designed for a short preparation pass before going outside and an
offline-friendly capture log while observing. SQLite is canonical; Obsidian is
the automatically generated mission notebook.

## Before the observing session

1. Start NightOps with `make run` or the built `nightops` executable. Startup
   initializes SQLite, loads cached weather, and warms missing catalog target
   references from the configured Wikipedia-compatible endpoint.
2. Complete the boot animation or press Enter to skip it.
3. Choose `LAUNCH MISSION`. Select Home Base, ZIP, SkyBase Atlas, or a real GPS
   adapter. Unknown coordinates remain unknown; NightOps never guesses them.
4. NightOps immediately opens `TONIGHT'S TARGETS`. Use Space to select targets
   in capture order, then press `c` to build the mission. The target list shows
   visibility and weather context when coordinates and forecast data exist.
5. Select an equipment profile when available. Add required inventory in
   Settings before the session if you want the generated checklist to be
   meaningful. The mission date and window are filled automatically from the
   current run, astronomical darkness, and the selected targets.
6. Choose `REVIEW & CREATE OBSIDIAN MISSION` and inspect the origin, target
   windows, equipment, detailed weather, and GO/NO-GO assessment. No date or
   time entry is required; an advanced override is available only when needed.
   The review screen is the final launch gate and includes `EDIT MISSION
   DETAILS` to return to planning before anything is created.
7. Choose `LAUNCH MISSION + OPEN OBSIDIAN` to save the mission, generate the
   complete vault structure, and open the configured vault. Choose `LAUNCH
   MISSION + CONTINUE IN NIGHTOPS` to remain in the TUI.

## Generated vault

The clean NightOps vault has this structure:

```text
NightOps/
├── Index.md
├── Missions/Index.md and mission notes
├── Targets/Index.md and reusable target knowledge notes
├── Locations/Index.md and launch-site notes
└── Equipment/Index.md and reusable equipment notes
```

Every mission note contains auto-filled properties, local mission date,
launch-site details, current weather, a static equipment checklist, and an
ordered target table with capture guidance and recommended starting settings.
The long hourly forecast is at the bottom. Target notes retain reference
summaries, representative image links, reusable capture settings, and a
history line for every mission and location that used the object.

Mission names default to the local observing date and start time. The offline
Mission Archive is newest-first, and the generated Missions index is also kept
newest-first, so selecting a previous night's capture does not require searching
for an arbitrary mission name.

## During an operation

After the countdown, Deep Space shows the selected target sequence. Use `a` to
record an observation, `o` to open the operation lifecycle, and `Esc` to go
back. Complete the operation to enter Debrief and save a user-authored summary.
Flight Recorder entries and debriefs remain in the mission note.

## If you stay offline

Cached target knowledge, weather, Atlas data, and mission records remain
available. New reference data is marked unavailable rather than invented. The
mission still saves locally and can be exported when the vault is available.

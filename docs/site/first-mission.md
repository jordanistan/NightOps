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
4. In Mission Planning choose `BUILD TARGET SEQUENCE`. Use Space to select
   targets in order, then press `c` to confirm. Target pages and cached
   reference summaries are prepared from SQLite, so this step is quick in the
   field.
5. Select an equipment profile when available. Add required inventory in
   Settings before the session if you want the generated checklist to be
   meaningful. A mission window is optional; live missions automatically use
   the local date at mission creation.
6. Choose `REVIEW MISSION` and inspect the origin, target order, equipment,
   weather, and available astronomy facts.
7. Choose `LAUNCH + OPEN OBSIDIAN` to save the mission, generate the complete
   vault structure, and open the configured vault. Choose `LAUNCH + CONTINUE IN
   NIGHTOPS` to remain in the TUI.

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
launch-site details, current and hourly weather, an equipment checklist, the
ordered target list, capture guidance, and links into target/location/equipment
pages. Target notes retain reference summaries, representative image links,
and a history line for every mission and location that used the object.

## During an operation

After the countdown, Deep Space shows the selected target sequence. Use `a` to
record an observation, `o` to open the operation lifecycle, and `Esc` to go
back. Complete the operation to enter Debrief and save a user-authored summary.
Flight Recorder entries and debriefs remain in the mission note.

## If you stay offline

Cached target knowledge, weather, Atlas data, and mission records remain
available. New reference data is marked unavailable rather than invented. The
mission still saves locally and can be exported when the vault is available.

CREATE TABLE IF NOT EXISTS equipment_items (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES equipment_profiles(id),
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    required INTEGER NOT NULL DEFAULT 1 CHECK (required IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_equipment_items_profile ON equipment_items(profile_id);

CREATE TABLE IF NOT EXISTS mission_targets (
    mission_id TEXT NOT NULL REFERENCES missions(id),
    target_id TEXT NOT NULL,
    target_name TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    right_ascension REAL NOT NULL,
    declination REAL NOT NULL,
    source TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (mission_id, target_id)
);

CREATE INDEX IF NOT EXISTS idx_mission_targets_mission_position ON mission_targets(mission_id, position);

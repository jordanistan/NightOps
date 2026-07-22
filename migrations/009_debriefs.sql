CREATE TABLE IF NOT EXISTS debriefs (
    id TEXT PRIMARY KEY,
    mission_id TEXT NOT NULL UNIQUE REFERENCES missions(id),
    summary TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

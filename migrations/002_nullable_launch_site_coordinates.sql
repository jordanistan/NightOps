PRAGMA foreign_keys=off;

CREATE TABLE launch_sites_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    timezone TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO launch_sites_new (id, name, latitude, longitude, timezone, source, created_at, updated_at)
SELECT id, name, latitude, longitude, timezone, source, created_at, updated_at FROM launch_sites;

CREATE TABLE missions_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'planned', 'launched', 'active', 'paused', 'completed', 'cancelled', 'archived')),
    launch_site_id TEXT NOT NULL REFERENCES launch_sites_new(id),
    planned_start TEXT,
    planned_end TEXT,
    started_at TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO missions_new SELECT * FROM missions;

CREATE TABLE observations_new (
    id TEXT PRIMARY KEY,
    mission_id TEXT NOT NULL REFERENCES missions_new(id),
    target_name TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    observed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO observations_new SELECT * FROM observations;

CREATE TABLE export_jobs_new (
    id TEXT PRIMARY KEY,
    mission_id TEXT NOT NULL REFERENCES missions_new(id),
    destination TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    idempotency_key TEXT NOT NULL UNIQUE,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO export_jobs_new SELECT * FROM export_jobs;

DROP TABLE export_jobs;
DROP TABLE observations;
DROP TABLE missions;
DROP TABLE launch_sites;

ALTER TABLE launch_sites_new RENAME TO launch_sites;
ALTER TABLE missions_new RENAME TO missions;
ALTER TABLE observations_new RENAME TO observations;
ALTER TABLE export_jobs_new RENAME TO export_jobs;

PRAGMA foreign_keys=on;

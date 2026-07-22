CREATE TABLE IF NOT EXISTS atlas_catalogs (
    version TEXT PRIMARY KEY,
    imported_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
);

CREATE TABLE IF NOT EXISTS atlas_locations (
    id TEXT NOT NULL,
    catalog_version TEXT NOT NULL REFERENCES atlas_catalogs(version),
    name TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    timezone TEXT NOT NULL,
    bortle_class REAL,
    source TEXT NOT NULL,
    PRIMARY KEY (catalog_version, id)
);

CREATE INDEX IF NOT EXISTS idx_atlas_catalogs_active ON atlas_catalogs(active);

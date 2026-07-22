CREATE TABLE IF NOT EXISTS weather_cache (
    location_key TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    temperature_c REAL,
    cloud_cover_percent REAL,
    payload TEXT NOT NULL DEFAULT ''
);

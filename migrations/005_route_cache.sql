CREATE TABLE IF NOT EXISTS route_cache (
    route_key TEXT PRIMARY KEY,
    route_json TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

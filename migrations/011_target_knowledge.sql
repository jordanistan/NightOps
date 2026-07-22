CREATE TABLE IF NOT EXISTS target_knowledge (
    target_id TEXT PRIMARY KEY,
    target_name TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    fetched_at TEXT
);

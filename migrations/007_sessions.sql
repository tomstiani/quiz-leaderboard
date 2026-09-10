CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('player', 'owner')),
    player_id TEXT NOT NULL DEFAULT '',
    credential TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

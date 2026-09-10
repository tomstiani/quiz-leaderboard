CREATE TABLE submissions (
    id TEXT PRIMARY KEY,
    player_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    game_day TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'confirmed')),
    filename TEXT NOT NULL,
    media_type TEXT NOT NULL,
    raw_score INTEGER,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TEXT,
    UNIQUE (player_id, game_id, game_day)
);

CREATE INDEX submissions_day_status ON submissions (game_day, status);

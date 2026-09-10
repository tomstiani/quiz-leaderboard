CREATE TABLE completion_notifications (
    player_id TEXT NOT NULL,
    game_day TEXT NOT NULL,
    claimed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TEXT,
    PRIMARY KEY (player_id, game_day)
);

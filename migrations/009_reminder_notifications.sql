CREATE TABLE reminder_notifications (
    player_id TEXT NOT NULL,
    game_day TEXT NOT NULL,
    sent_at TEXT,
    PRIMARY KEY (player_id, game_day)
);

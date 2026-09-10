CREATE TABLE vision_attempts (
    player_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    game_day TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    PRIMARY KEY (player_id, game_id, game_day)
);

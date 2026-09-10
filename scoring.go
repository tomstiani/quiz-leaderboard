package main

import (
	"database/sql"
	"fmt"
	"sort"
)

func normalizeScore(raw int, maximum float64) float64 {
	return float64(raw) / maximum * 100
}

func recalculateNormalizedScores(db *sql.DB, cfg config) error {
	rows, err := db.Query(`SELECT id, game_id, raw_score FROM submissions
		WHERE status = 'confirmed'`)
	if err != nil {
		return fmt.Errorf("find scores to normalize: %w", err)
	}
	type pendingScore struct {
		id, gameID string
		raw        int
	}
	var pending []pendingScore
	for rows.Next() {
		var score pendingScore
		if err := rows.Scan(&score.id, &score.gameID, &score.raw); err != nil {
			rows.Close()
			return fmt.Errorf("read score to normalize: %w", err)
		}
		pending = append(pending, score)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read scores to normalize: %w", err)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, score := range pending {
		game, ok := findGame(cfg, score.gameID)
		if !ok {
			continue
		}
		if _, err := db.Exec("UPDATE submissions SET normalized_score = ? WHERE id = ?", normalizeScore(score.raw, game.MaxScore), score.id); err != nil {
			return fmt.Errorf("normalize stored score: %w", err)
		}
	}
	return nil
}

func rankPlayers(players []dashboardPlayer) {
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].CombinedScore > players[j].CombinedScore
	})
	for index := range players {
		if index == 0 || players[index].CombinedScore != players[index-1].CombinedScore {
			players[index].Rank = index + 1
		} else {
			players[index].Rank = players[index-1].Rank
		}
	}
}

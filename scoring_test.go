package main

import "testing"

func TestNormalizeAndRankPlayers(t *testing.T) {
	if got := normalizeScore(450, 900); got != 50 {
		t.Fatalf("normalizeScore()=%v, want 50", got)
	}
	players := []dashboardPlayer{
		{Name: "Half", CombinedScore: 50},
		{Name: "Winner A", CombinedScore: 100},
		{Name: "Winner B", CombinedScore: 100},
		{Name: "Missing", CombinedScore: 0},
	}
	rankPlayers(players)
	wantNames := []string{"Winner A", "Winner B", "Half", "Missing"}
	wantRanks := []int{1, 1, 3, 4}
	for index := range players {
		if players[index].Name != wantNames[index] || players[index].Rank != wantRanks[index] {
			t.Fatalf("player %d=%+v, want %s rank %d", index, players[index], wantNames[index], wantRanks[index])
		}
	}
}

func TestRecalculateNormalizedScores(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type, raw_score)
		VALUES ('score', 'alice', 'geopolitix', '2026-09-10', 'confirmed', 'score.png', 'image/png', 450)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE submissions SET normalized_score = 99 WHERE id = 'score'"); err != nil {
		t.Fatal(err)
	}
	if err := recalculateNormalizedScores(db, testConfig()); err != nil {
		t.Fatal(err)
	}
	var normalized float64
	if err := db.QueryRow("SELECT normalized_score FROM submissions WHERE id = 'score'").Scan(&normalized); err != nil || normalized != 50 {
		t.Fatalf("normalized=%v err=%v", normalized, err)
	}
}

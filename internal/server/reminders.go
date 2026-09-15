package server

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func runReminderScheduler(ctx context.Context, cfg config, db *sql.DB) {
	if !cfg.WebPush.enabled() {
		return
	}
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		log.Printf("load reminder timezone: %v", err)
		return
	}
	for {
		now := time.Now().In(oslo)
		if now.Hour() >= 18 {
			reminderCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			if err := sendDailyReminders(reminderCtx, cfg, db, now.Format(time.DateOnly), nil); err != nil {
				log.Printf("send daily reminders: %v", err)
			}
			cancel()
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, oslo)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

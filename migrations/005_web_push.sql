ALTER TABLE completion_notifications ADD COLUMN push_sent_at TEXT;
UPDATE completion_notifications SET push_sent_at = sent_at WHERE sent_at IS NOT NULL;

CREATE TABLE push_subscriptions (
    endpoint TEXT PRIMARY KEY,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    subscriber_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

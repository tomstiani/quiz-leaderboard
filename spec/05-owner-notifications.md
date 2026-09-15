# Milestone 5: Owner corrections and notifications

**Status:** `blocked`

**Depends on:** Milestone 4

## Goal

Let the owner repair today's mistakes, notify the group when a player completes both games, and remind subscribed players about unfinished games.

## Tasks

- [x] **OWN-001:** List today's submissions through an owner-only endpoint and screen.
- [x] **OWN-002:** Correct a raw score while retaining its screenshot.
- [x] **OWN-003:** Recalculate the normalized and combined scores after correction.
- [x] **OWN-004:** Reopen or remove an invalid submission so its player can submit again.
- [x] **OWN-005:** Restrict owner changes to the current Oslo day.
- [x] **NTFY-001:** Configure the existing ntfy endpoint, private topic, and credentials.
- [x] **NTFY-002:** Publish the player's name and combined total after their final game is confirmed.
- [x] **NTFY-003:** Record completion notification state once per player and day.
- [x] **NTFY-004:** Suppress repeat notifications after an owner reopens and the player reconfirms.
- [x] **NTFY-005:** Keep submission confirmation successful when ntfy is unavailable.
- [x] **OWN-006:** Test owner authorization, correction, reopen, score recalculation, and notification deduplication.
- [x] **PUSH-001:** Store authenticated Web Push subscriptions and private VAPID configuration.
- [x] **PUSH-002:** Register a service worker and let players enable or disable browser notifications.
- [x] **PUSH-003:** Deliver completion notifications while the dashboard is closed and remove expired subscriptions.
- [x] **PUSH-004:** Remove the current browser subscription on explicit logout.
- [x] **PUSH-005:** Deduplicate browser notifications per player and day and test encrypted delivery.
- [x] **PUSH-006:** At 18:00 Europe/Oslo, remind each subscribed player with unfinished games once.

## Acceptance criteria

- [x] Players cannot call owner actions.
- [x] The owner can repair today's score or permit a replacement screenshot.
- [x] Owner actions cannot alter retained past days.
- [x] Completing both games sends one message containing the player name and combined total.
- [x] ntfy or Web Push failure does not lose or roll back a confirmed submission.
- [x] An opted-in browser can display completion notifications after the dashboard is closed.
- [x] An incomplete subscribed player receives one 18:00 reminder; completed and unsubscribed players do not.

## Blockers

- Existing ntfy connection details and the public HTTPS domain are required for real-service notification verification. Automated tests cover ntfy success, failure, and retry plus encrypted Web Push delivery, stale subscription cleanup, and per-channel deduplication.

## Verification

- `go test -count=1 -race ./...`
- `go vet ./...`
- `cd web && pnpm test && pnpm typecheck && pnpm build`
- `cd web && pnpm e2e` verifies owner correction, reopen, screenshot retention, and live player updates.

## Deferred

- Player and token management remain configuration-only.
- No notification preferences or per-player topics.

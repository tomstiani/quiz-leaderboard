# Milestone 5: Owner corrections and notifications

**Status:** `not_started`

**Depends on:** Milestone 4

## Goal

Let the owner repair today's mistakes and notify the group once when a player completes both games.

## Tasks

- [ ] **OWN-001:** List today's submissions through an owner-only endpoint and screen.
- [ ] **OWN-002:** Correct a raw score while retaining its screenshot.
- [ ] **OWN-003:** Recalculate the normalized and combined scores after correction.
- [ ] **OWN-004:** Reopen or remove an invalid submission so its player can submit again.
- [ ] **OWN-005:** Restrict owner changes to the current Oslo day.
- [ ] **NTFY-001:** Configure the existing ntfy endpoint, private topic, and credentials.
- [ ] **NTFY-002:** Publish the player's name and combined total after their final game is confirmed.
- [ ] **NTFY-003:** Record completion notification state once per player and day.
- [ ] **NTFY-004:** Suppress repeat notifications after an owner reopens and the player reconfirms.
- [ ] **NTFY-005:** Keep submission confirmation successful when ntfy is unavailable.
- [ ] **OWN-006:** Test owner authorization, correction, reopen, score recalculation, and notification deduplication.

## Acceptance criteria

- [ ] Players cannot call owner actions.
- [ ] The owner can repair today's score or permit a replacement screenshot.
- [ ] Owner actions cannot alter retained past days.
- [ ] Completing both games sends one message containing the player name and combined total.
- [ ] ntfy failure does not lose or roll back a confirmed submission.

## Blockers

- Existing ntfy connection details are required for end-to-end verification.

## Deferred

- Player and token management remain configuration-only.
- No notification preferences or per-player topics.

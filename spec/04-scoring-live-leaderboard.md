# Milestone 4: Scoring and live leaderboard

**Status:** `not_started`

**Depends on:** Milestone 3

## Goal

Rank players by an equally weighted combined daily score and update open dashboards immediately.

## Tasks

- [ ] **SCORE-001:** Implement explicit normalization for Krillion and Geopolitix.
- [ ] **SCORE-002:** Store raw and normalized scores on confirmation.
- [ ] **SCORE-003:** Count each unfinished game as zero.
- [ ] **SCORE-004:** Calculate the combined score from 0 to 200.
- [ ] **SCORE-005:** Assign the same rank to equal combined scores without a time tie-breaker.
- [ ] **DASH-002:** Display rank, combined score, completion count, raw scores, normalized scores, and screenshots.
- [ ] **LIVE-001:** Add an authenticated server-sent events endpoint.
- [ ] **LIVE-002:** Publish a leaderboard-change event after confirmation or owner correction.
- [ ] **LIVE-003:** Re-fetch dashboard data after an event and reconnect after interruption.
- [ ] **SCORE-006:** Test normalization, missing games, ties, and the Oslo day boundary.

## Acceptance criteria

- [ ] Each game contributes at most 100 points.
- [ ] Missing submissions contribute zero.
- [ ] Equal totals share a rank.
- [ ] A confirmed result appears on another open dashboard without a manual refresh.
- [ ] Reconnecting after a dropped event stream restores updates.

## Blockers

- The Geopolitix maximum score must be confirmed.

## Deferred

- No WebSockets; server-sent events cover the required one-way updates.
- No historical, weekly, or streak calculations.

# Milestone 4: Scoring and live leaderboard

**Status:** `complete`

**Depends on:** Milestone 3

## Goal

Rank players by an equally weighted combined daily score and update open dashboards immediately.

## Tasks

- [x] **SCORE-001:** Implement explicit normalization for Krillion and Geopolitix.
- [x] **SCORE-002:** Store raw and normalized scores on confirmation.
- [x] **SCORE-003:** Count each unfinished game as zero.
- [x] **SCORE-004:** Calculate the combined score from 0 to 200.
- [x] **SCORE-005:** Assign the same rank to equal combined scores without a time tie-breaker.
- [x] **DASH-002:** Display rank, combined score, completion count, raw scores, normalized scores, and screenshots.
- [x] **DASH-003:** Let players select and view a read-only past daily leaderboard.
- [x] **LIVE-001:** Add an authenticated server-sent events endpoint.
- [x] **LIVE-002:** Publish a leaderboard-change event after confirmation. Milestone 5 owner actions must publish through the same broker.
- [x] **LIVE-003:** Re-fetch dashboard data after an event and use native EventSource reconnection after interruption.
- [x] **SCORE-006:** Test normalization, missing games, ties, and the Oslo day boundary.

## Acceptance criteria

- [x] Each game contributes at most 100 points.
- [x] Missing submissions contribute zero.
- [x] Equal totals share a rank.
- [x] A confirmed result appears on another open dashboard without a manual refresh.
- [x] Reconnecting after a dropped event stream restores updates.

## Blockers

None.

## Verification

- `go test -count=1 -race ./...`
- `go vet ./...`
- `cd web && pnpm test && pnpm typecheck && pnpm build`
- `cd web && pnpm e2e` verified a second signed-in browser updated after confirmation without reloading.

## Deferred

- No WebSockets; server-sent events cover the required one-way updates.
- No streak calculations or historical summaries beyond daily, weekly, and monthly totals.

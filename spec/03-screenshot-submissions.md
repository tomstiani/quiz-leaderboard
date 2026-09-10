# Milestone 3: Screenshot submissions

**Status:** `complete`

**Depends on:** Milestone 2

## Goal

Let a player upload, validate, review, and permanently confirm one result screenshot per game and Oslo day.

## Tasks

- [x] **SUB-001:** Add submission and draft persistence with a unique player/game/day constraint for confirmed submissions.
- [x] **SUB-002:** Accept PNG, JPEG, and WebP files up to 10 MB and verify content type from file contents.
- [x] **SUB-003:** Store screenshots under generated names in the configured data directory.
- [x] **SUB-004:** Select a current low-cost vision model that accepts both game result formats.
- [x] **SUB-005:** Send the selected game and screenshot to the vision API with a strict structured response.
- [x] **SUB-006:** Require the model to return screenshot validity, raw total score when readable, and a rejection reason.
- [x] **SUB-007:** Reject screenshots that do not show the selected game's completed result page.
- [x] **SUB-008:** Show the screenshot and an editable score before confirmation.
- [x] **SUB-009:** Allow manual score entry only after the screenshot passes model validation.
- [x] **SUB-010:** Validate the confirmed score against the configured game range.
- [x] **SUB-011:** Finalize the submission atomically and prevent player changes afterward.
- [x] **SUB-012:** Serve screenshots only through authenticated routes.
- [x] **SUB-013:** Remove abandoned drafts and files with a small opportunistic cleanup.
- [x] **SUB-014:** Disclose third-party screenshot processing in the upload interface.
- [x] **SUB-015:** Test file limits, invalid images, duplicate confirmation, score bounds, and private image access.

## Acceptance criteria

- [x] A player can confirm one valid result for each game today.
- [x] Invalid, oversized, or unsupported uploads never reach the vision API.
- [x] An unreadable score can be entered manually only after image validation succeeds.
- [x] A confirmed submission cannot be replaced by its player.
- [x] Another player can view the confirmed screenshot after authentication.
- [x] Direct unauthenticated screenshot requests fail.

## Blockers

None.

## Verification

- `go test -count=1 -race ./...`
- `go vet ./...`
- `cd web && pnpm test && pnpm typecheck && pnpm build`
- `cd web && pnpm e2e` (4 Chromium flow tests, including upload through the full application)
- The configured OpenRouter model successfully analyzed and rejected an invalid probe image.

## Deferred

- No model-provider abstraction. Replace the direct integration only if a second provider is actually needed.
- No player-facing upload history.

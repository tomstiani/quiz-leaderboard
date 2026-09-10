# Milestone 3: Screenshot submissions

**Status:** `not_started`

**Depends on:** Milestone 2

## Goal

Let a player upload, validate, review, and permanently confirm one result screenshot per game and Oslo day.

## Tasks

- [ ] **SUB-001:** Add submission and draft persistence with a unique player/game/day constraint for confirmed submissions.
- [ ] **SUB-002:** Accept PNG, JPEG, and WebP files up to 10 MB and verify content type from file contents.
- [ ] **SUB-003:** Store screenshots under generated names in the configured data directory.
- [ ] **SUB-004:** Select a current low-cost vision model that accepts both game result formats.
- [ ] **SUB-005:** Send the selected game and screenshot to the vision API with a strict structured response.
- [ ] **SUB-006:** Require the model to return screenshot validity, raw total score when readable, and a rejection reason.
- [ ] **SUB-007:** Reject screenshots that do not show the selected game's completed result page.
- [ ] **SUB-008:** Show the screenshot and an editable score before confirmation.
- [ ] **SUB-009:** Allow manual score entry only after the screenshot passes model validation.
- [ ] **SUB-010:** Validate the confirmed score against the configured game range.
- [ ] **SUB-011:** Finalize the submission atomically and prevent player changes afterward.
- [ ] **SUB-012:** Serve screenshots only through authenticated routes.
- [ ] **SUB-013:** Remove abandoned drafts and files with a small opportunistic cleanup.
- [ ] **SUB-014:** Disclose third-party screenshot processing in the upload interface.
- [ ] **SUB-015:** Test file limits, invalid images, duplicate confirmation, score bounds, and private image access.

## Acceptance criteria

- [ ] A player can confirm one valid result for each game today.
- [ ] Invalid, oversized, or unsupported uploads never reach the vision API.
- [ ] An unreadable score can be entered manually only after image validation succeeds.
- [ ] A confirmed submission cannot be replaced by its player.
- [ ] Another player can view the confirmed screenshot after authentication.
- [ ] Direct unauthenticated screenshot requests fail.

## Blockers

- The Geopolitix maximum score must be confirmed before enforcing its final score range.
- Vision API credentials are required for end-to-end verification.

## Deferred

- No model-provider abstraction. Replace the direct integration only if a second provider is actually needed.
- No player-facing upload history.

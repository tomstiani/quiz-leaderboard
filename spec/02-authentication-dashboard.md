# Milestone 2: Authentication and dashboard

**Status:** `complete`

**Depends on:** Milestone 1

## Goal

Let configured players and the owner sign in, remain signed in until logout, and view today's empty leaderboard.

## Tasks

- [x] **AUTH-001:** Load stable player IDs, display names, and personal tokens from private configuration.
- [x] **AUTH-002:** Validate player and owner tokens without leaking timing or token values.
- [x] **AUTH-003:** Issue signed, secure, HTTP-only, same-site session cookies.
- [x] **AUTH-004:** Reject sessions for players removed from configuration.
- [x] **AUTH-005:** Add login, current-session, and logout endpoints.
- [x] **AUTH-006:** Protect private API routes and distinguish player and owner access.
- [x] **DAY-001:** Resolve the current calendar date in `Europe/Oslo`.
- [x] **DASH-001:** Return today's configured games, players, completion counts, and empty scores.
- [x] **UI-001:** Build the accessible token login screen with clear invalid-token errors.
- [x] **UI-002:** Build the responsive dashboard shell for phone and desktop.
- [x] **UI-003:** Display Geopolitix and Krillion links.
- [x] **UI-004:** Add logout and an owner entry point.
- [x] **AUTH-007:** Test valid login, invalid login, logout, route protection, and Oslo date handling.

## Acceptance criteria

- [x] A configured player can sign in once and remain signed in after reopening the browser.
- [x] Logout removes access.
- [x] A removed player loses access even with an old cookie.
- [x] The owner token cannot be used as a player identity.
- [x] Unauthenticated users cannot read players or dashboard data.
- [x] The dashboard works at narrow phone and wide desktop sizes.

## Blockers

None.

## Verification

- `go test -count=1 ./...`
- `go vet ./...`
- `cd web && pnpm test && pnpm typecheck && pnpm build`
- Container smoke test covered login and the authenticated dashboard API.

## Deferred

- Player management remains configuration-only.
- Password recovery, OAuth, and registration are out of scope.

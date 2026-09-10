# Milestone 2: Authentication and dashboard

**Status:** `not_started`

**Depends on:** Milestone 1

## Goal

Let configured players and the owner sign in, remain signed in until logout, and view today's empty leaderboard.

## Tasks

- [ ] **AUTH-001:** Load stable player IDs, display names, and personal tokens from private configuration.
- [ ] **AUTH-002:** Validate player and owner tokens without leaking timing or token values.
- [ ] **AUTH-003:** Issue signed, secure, HTTP-only, same-site session cookies.
- [ ] **AUTH-004:** Reject sessions for players removed from configuration.
- [ ] **AUTH-005:** Add login, current-session, and logout endpoints.
- [ ] **AUTH-006:** Protect private API routes and distinguish player and owner access.
- [ ] **DAY-001:** Resolve the current calendar date in `Europe/Oslo`.
- [ ] **DASH-001:** Return today's configured games, players, completion counts, and empty scores.
- [ ] **UI-001:** Build the accessible token login screen with clear invalid-token errors.
- [ ] **UI-002:** Build the responsive dashboard shell for phone and desktop.
- [ ] **UI-003:** Display Geopolitix and Krillion links.
- [ ] **UI-004:** Add logout and an owner entry point.
- [ ] **AUTH-007:** Test valid login, invalid login, logout, route protection, and Oslo date handling.

## Acceptance criteria

- [ ] A configured player can sign in once and remain signed in after reopening the browser.
- [ ] Logout removes access.
- [ ] A removed player loses access even with an old cookie.
- [ ] The owner token cannot be used as a player identity.
- [ ] Unauthenticated users cannot read players or dashboard data.
- [ ] The dashboard works at narrow phone and wide desktop sizes.

## Blockers

None.

## Deferred

- Player management remains configuration-only.
- Password recovery, OAuth, and registration are out of scope.

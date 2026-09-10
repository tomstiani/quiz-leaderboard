# Milestone 1: Project foundation

**Status:** `complete`

## Goal

Create one locally runnable and deployable application with a Go backend, React/Vite frontend, SQLite storage, and private configuration.

## Tasks

- [x] **FND-001:** Initialize the Go module and server entry point.
- [x] **FND-002:** Initialize React, TypeScript, Vite, and TanStack Router.
- [x] **FND-003:** Serve the production frontend from the Go process.
- [x] **FND-004:** Add a development proxy from Vite to the Go API.
- [x] **FND-005:** Define and validate private configuration for players, tokens, games, storage, sessions, vision API, and ntfy.
- [x] **FND-006:** Add an example configuration and ignore the private runtime copy.
- [x] **FND-007:** Open SQLite and apply embedded schema migrations at startup.
- [x] **FND-008:** Use one configurable data directory for SQLite and screenshots.
- [x] **FND-009:** Add `GET /api/health`.
- [x] **FND-010:** Add production Dockerfile and Docker Compose configuration with a persistent data mount.
- [x] **FND-011:** Document local development, build, test, and configuration commands.
- [x] **FND-012:** Add the smallest automated backend check and frontend build check.

## Acceptance criteria

- [x] The Go test command passes.
- [x] The frontend type check and production build pass.
- [x] The production Go process serves the frontend and `/api/health`.
- [x] Docker Compose starts the application with persistent storage.
- [x] Secrets and runtime data are excluded from version control.

## Blockers

None.

## Verification

- `go test ./...`
- `go vet ./...`
- `cd web && pnpm typecheck && pnpm build`
- `docker --context default compose up --build -d`
- Health and frontend requests passed after container recreation; SQLite remained on the host mount.

## Deferred

- Authentication behavior belongs to milestone 2.
- Submission tables beyond the minimum migration mechanism belong to milestone 3.
- Do not add CI until the repository needs a remote build gate.

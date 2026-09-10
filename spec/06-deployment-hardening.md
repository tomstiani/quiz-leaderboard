# Milestone 6: Deployment and hardening

**Status:** `blocked`

**Depends on:** Milestone 5

## Goal

Deploy the finished MVP safely behind the existing HTTPS reverse proxy and verify its complete daily flow.

## Tasks

- [x] **OPS-001:** Build a minimal production container containing the Go service and compiled frontend.
- [x] **OPS-002:** Run the application with Docker Compose and one backed-up persistent data mount.
- [ ] **OPS-003:** Configure the public domain and existing HTTPS reverse proxy, including server-sent event streaming.
- [x] **SEC-001:** Keep player tokens, owner token, session secret, vision credentials, ntfy credentials, and VAPID keys outside version control.
- [x] **SEC-002:** Enforce request limits, server timeouts, safe upload paths, vision quotas, and bounded notification fanout.
- [x] **SEC-003:** Protect cookie-authenticated mutations against cross-site requests.
- [x] **SEC-004:** Prevent public file serving and accidental secret logging.
- [x] **SEC-005:** Run as a non-root, read-only, capability-free container with resource and log limits.
- [x] **SEC-006:** Revoke server-side sessions on logout and reject placeholder deployment credentials.
- [x] **SEC-007:** Restrict Web Push outbound traffic, validate subscriptions, and cap stored devices.
- [x] **OPS-004:** Verify SQLite and screenshots survive container recreation.
- [ ] **OPS-005:** Verify the mounted data directory is included in existing backups.
- [ ] **OPS-006:** Run a public-domain smoke test with two players, both games, a correction, live updates, ntfy, and browser push.
- [x] **OPS-007:** Document deploy, update, backup, restore, and rollback commands.
- [x] **OPS-008:** Verify a stopped-service backup restores with valid SQLite integrity and preserved secret modes.

## Acceptance criteria

- [ ] The public application is available only over HTTPS.
- [ ] Authentication, uploads, screenshots, live updates, owner corrections, and notifications work through the reverse proxy.
- [x] Restarting or recreating the container preserves all retained data.
- [x] No secret or uploaded screenshot is publicly accessible from the application container.
- [x] The documented restore procedure recovers configuration, SQLite, WAL files, and screenshots together.

## Blockers

- Public domain and reverse-proxy details are required for HTTPS and SSE verification.
- Existing backup-system details are required to verify off-host backup inclusion.
- Existing ntfy connection details are required for real notification verification.

## Verification

- `go test -count=1 -race ./...` and `go vet ./...`
- `cd web && pnpm test && pnpm typecheck && pnpm build && pnpm audit --prod`
- `cd web && pnpm e2e`
- `govulncheck ./...` found no reachable vulnerabilities.
- Trivy found zero high or critical vulnerabilities in the final Alpine image and Go binary.
- Docker inspection verified non-root UID 1000, read-only root filesystem, dropped capabilities, and no-new-privileges.
- Container recreation preserved the submission count and screenshot SHA-256.
- A stopped-service archive restored with `PRAGMA integrity_check` equal to `ok` and mode `0600` on `config.json`.

## Deferred

- No Kubernetes, managed object storage, or application-managed backup scheduler.
- Add CI only when a remote repository or team workflow needs it.

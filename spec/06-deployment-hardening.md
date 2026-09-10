# Milestone 6: Deployment and hardening

**Status:** `not_started`

**Depends on:** Milestone 5

## Goal

Deploy the finished MVP safely behind the existing HTTPS reverse proxy and verify its complete daily flow.

## Tasks

- [ ] **OPS-001:** Build a minimal production container containing the Go service and compiled frontend.
- [ ] **OPS-002:** Run the application with Docker Compose and one backed-up persistent data mount.
- [ ] **OPS-003:** Configure the public domain and existing HTTPS reverse proxy, including server-sent event streaming.
- [ ] **SEC-001:** Keep player tokens, owner token, session secret, vision credentials, and ntfy credentials outside version control.
- [ ] **SEC-002:** Enforce request body limits, server timeouts, and safe upload paths.
- [ ] **SEC-003:** Protect cookie-authenticated mutations against cross-site requests.
- [ ] **SEC-004:** Prevent public file serving and accidental secret logging.
- [ ] **OPS-004:** Verify SQLite and screenshots survive container recreation.
- [ ] **OPS-005:** Verify the mounted data directory is included in existing backups.
- [ ] **OPS-006:** Run a public-domain smoke test with two players, both games, a correction, live updates, and ntfy.
- [ ] **OPS-007:** Document deploy, update, backup, restore, and rollback commands.

## Acceptance criteria

- [ ] The public application is available only over HTTPS.
- [ ] Authentication, uploads, screenshots, live updates, owner corrections, and notifications work through the reverse proxy.
- [ ] Restarting or recreating the container preserves all retained data.
- [ ] No secret or uploaded screenshot is publicly accessible.
- [ ] The documented restore procedure recovers both SQLite and screenshot files together.

## Blockers

- Public domain and reverse-proxy details are required for final verification.

## Deferred

- No Kubernetes, managed object storage, or application-managed backup scheduler.
- Add CI only when a remote repository or team workflow needs it.

# Implementation specifications

These files are the project execution ledger. Update task checkboxes and milestone status in the same change that completes work.

The product requirements live in [`../docs/product.md`](../docs/product.md). Product requirements take precedence over implementation details here.

## Status values

- `not_started`
- `in_progress`
- `blocked`
- `complete`

## Milestones

| Order | Milestone | Status | Depends on |
|---|---|---|---|
| 1 | [Project foundation](01-project-foundation.md) | `complete` | — |
| 2 | [Authentication and dashboard](02-authentication-dashboard.md) | `complete` | 1 |
| 3 | [Screenshot submissions](03-screenshot-submissions.md) | `complete` | 2 |
| 4 | [Scoring and live leaderboard](04-scoring-live-leaderboard.md) | `not_started` | 3 |
| 5 | [Owner corrections and notifications](05-owner-notifications.md) | `not_started` | 4 |
| 6 | [Deployment and hardening](06-deployment-hardening.md) | `not_started` | 5 |

## Working rules

1. Work milestones in order unless a task explicitly has no dependency.
2. Mark a milestone `in_progress` before changing application code.
3. Mark each task complete only when its check and relevant acceptance criteria pass.
4. Record blockers under the milestone's **Blockers** heading.
5. Do not add speculative tasks. Add work only when a requirement or discovered defect needs it.
6. Keep one agent responsible for each file area during parallel work to avoid conflicting edits.

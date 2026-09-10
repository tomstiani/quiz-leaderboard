# Product specification

## Status

The MVP product decisions are defined. Implementation is tracked in [`../spec/`](../spec/README.md).

The Geopolitix maximum score and vision provider remain undecided.

## Purpose

Daily Game Leaderboard gives a small group of friends one place to compare their results from external daily games.

Players use personal tokens to open a private dashboard. They follow links to each game, play outside this application, and submit screenshots of their final results. The dashboard combines normalized game scores into one daily leaderboard.

## MVP scope

The MVP supports:

- One private group with fewer than 10 players
- Players managed through private configuration
- Personal token authentication with persistent sessions
- A separate owner token
- Two daily games: Krillion and Geopolitix
- One screenshot submission per player, game, and day
- Screenshot validation and score extraction with a low-cost vision model
- Player review and confirmation before final submission
- A combined daily leaderboard with real-time updates
- Visible screenshots, raw scores, and normalized scores
- One ntfy notification when a player completes all games
- Owner correction of today's invalid submissions

The dashboard must work equally well on phone and desktop browsers.

## Non-goals

The MVP does not include:

- Public registration
- Email addresses, passwords, or password recovery
- Multiple teams
- OAuth
- Streaks
- Weekly or historical statistics
- A player-facing history view
- Advanced anti-cheat controls
- Player edits after final submission
- Automatic discovery of games
- Late submissions

## Users and access

### Player

Each player receives a personal login token. A successful login creates a secure session that lasts until logout.

A player can:

- View today's games and leaderboard
- View every player's confirmed scores and screenshots
- Upload one screenshot for each game
- Review and edit an extracted score before confirmation
- Enter a score manually when the model validates the screenshot but cannot read its score

A player cannot replace or edit a confirmed submission.

### Owner

The owner signs in with a separate owner token. For the current day, the owner can:

- Correct a numeric score while retaining its screenshot
- Reopen or remove an invalid submission so the player can submit again

Players and their tokens are managed through private deployment configuration, not an owner UI. Changing this configuration may require restarting the service.

## Daily dashboard

The dashboard uses `Europe/Oslo` to determine the current day. Players can submit results only for that day.

For each player, show:

- Combined normalized score
- Completion count, such as `1/2`
- Raw score for each game
- Normalized score for each game
- Submitted screenshot for each completed game

Confirmed results become available immediately. An open dashboard receives updates in real time. The planned transport is server-sent events because updates flow only from the server to the browser.

Past submissions and screenshots remain stored but are not exposed through the MVP interface.

## Games

The fixed initial game set contains:

1. Krillion
2. Geopolitix

Each game opens on its external website:

- [Geopolitix](https://geopolitix.live/)
- [Krillion](https://krillion.io/)

### Krillion

Krillion has a maximum score of 700 points. Its result is displayed as depth, where 700 points equals 7,000 metres.

Game: [Krillion](https://krillion.io/)

### Geopolitix

The maximum score is not confirmed. Its published rules mention both 900 base points and 1,125 points with all bonuses. Use a real completed-result screenshot to confirm which value appears and should be normalized.

Game: [Geopolitix](https://geopolitix.live/)

## Submission workflow

1. The player opens a game from the dashboard.
2. The player completes the game on its external website.
3. The player uploads the final total-score screenshot.
4. A vision model checks that the screenshot matches the selected game and shows a completed result.
5. The vision model attempts to extract the raw score.
6. The application shows the screenshot and score to the player.
7. The player corrects or enters the score if necessary.
8. The player confirms the submission.
9. The application stores the submission and updates every open dashboard.

Accept PNG, JPEG, and WebP uploads up to 10 MB. Enforce the type and size limits on the server.

Reject an upload when the model considers it an invalid completed-result screenshot. Manual score entry is available only after the screenshot itself passes validation. Score values must fit the selected game's valid range.

Confirmation makes the submission immutable to the player. The player can retry freely before confirmation.

For each vision request, require at least:

- Selected game
- Extracted raw score when readable
- Valid-result decision
- Rejection reason when invalid

Compare current low-cost vision models against real screenshots before selecting a provider and model. A third-party vision API is acceptable. Players must be informed that uploaded screenshots are processed externally.

## Scoring and ranking

Each game contributes equally to the daily total.

```text
normalized game score = raw score / maximum game score × 100
combined daily score = Krillion normalized score + Geopolitix normalized score
```

The combined daily score ranges from 0 to 200. An unfinished game has a normalized score of zero.

Store both the raw and normalized scores. This preserves the submitted result and allows normalization rules to change later.

Players with equal combined scores share the same rank. Completion time does not break ties.

Do not finalize Geopolitix normalization until its maximum score is confirmed.

## Notifications

Use an existing self-hosted ntfy instance. Every player subscribes to one private team topic.

When a player confirms their final unfinished game, publish a message containing the player's name and combined total. Send at most one completion notification per player and day, including after an owner reopens and the player reconfirms a submission.

The ntfy endpoint, topic, and credentials belong in private deployment configuration.

## Trust and privacy

This is a low-risk application for friends. Social verification comes from making confirmed scores and screenshots visible to the group.

The application must:

- Protect personal and owner tokens
- Restrict scores and screenshots to authenticated group members
- Keep screenshots behind authenticated routes rather than public file URLs
- Enforce upload limits before vision processing
- Disclose third-party screenshot processing

Elaborate anti-cheat and identity systems are outside the MVP.

## Technical direction

These choices constrain implementation but do not define a complete architecture:

- Go backend
- React and TypeScript frontend built with Vite
- TanStack Router
- TanStack Query only if the final data flow benefits from it
- Server-sent events for leaderboard updates
- SQLite database
- Screenshots stored on local disk
- One mounted data directory for the database and screenshots
- Existing home-server backups protect the mounted data
- Docker Compose deployment
- Existing HTTPS domain and reverse proxy
- Existing self-hosted ntfy instance

The Go service should serve the built frontend so deployment needs one application process. Do not add a JavaScript server or TanStack Start.

## Product rules

- A day follows the `Europe/Oslo` calendar date.
- Players can submit only for the current day.
- All players receive the same configured games each day.
- Each player can confirm at most one submission per game and day.
- Confirmed submissions are immediately visible.
- Players cannot modify confirmed submissions.
- The owner can correct or reopen only today's submissions.
- Missing submissions score zero.
- Both games have equal weight.
- Equal totals share a rank.
- Completion notifications are sent at most once per player and day.
- Past data is retained but hidden.

## Required inputs

Provide these before their related implementation work begins:

1. The exact Krillion and Geopolitix daily URLs
2. Representative completed-result screenshots from both games
3. Confirmation of the Geopolitix maximum score
4. Connection details for the existing ntfy instance
5. The intended public application domain
6. The initial player names and token configuration

## Remaining evaluation

Test several current low-cost vision models with the representative screenshots. Compare:

- Completed-result validation accuracy
- Score extraction accuracy
- Structured-output reliability
- Latency
- Cost

Choose the cheapest model that reliably handles both games. Do not build provider abstraction beyond what is needed to call the selected model.

# Daily Game Leaderboard

A private daily leaderboard for a small quiz group. Players submit result screenshots, review the extracted score, and compete on a live normalized leaderboard.

## Requirements

### Local development

- Go 1.26.6
- Node.js 22
- Corepack and pnpm 12.2.1
- An OpenRouter API key for screenshot analysis

### Production

- Docker Engine with Docker Compose
- A public HTTPS domain
- A reverse proxy on the application host
- An off-host backup destination

## Local setup

1. Enable the repository's pinned pnpm version and install frontend dependencies:

   ```sh
   corepack enable
   cd web
   pnpm install --frozen-lockfile
   cd ..
   ```

2. Create the private runtime configuration:

   ```sh
   cp config.example.json config.json
   chmod 600 config.json
   ```

3. Generate credentials and copy each value into `config.json`:

   ```sh
   openssl rand -base64 48 # sessionSecret
   openssl rand -hex 24   # ownerToken
   openssl rand -hex 24   # one unique token for each player
   ```

4. Edit `config.json`:

   - Set `secureCookies` to `false` for local HTTP development.
   - Replace every `replace-*` value.
   - Add the OpenRouter API key under `vision.apiKey`.
   - Add each player with a unique ID, display name, and token.
   - Keep the configured game maximums aligned with each game's scoring system.
   - Leave `ntfy` and `webPush` empty if you do not need notifications locally.

5. Start the Go server from the repository root:

   ```sh
   go run .
   ```

6. Start Vite in another terminal:

   ```sh
   cd web
   pnpm dev
   ```

7. Open <http://localhost:5173>. Vite forwards `/api` requests to the Go server on port 8080.

The server creates `data/leaderboard.db` and the screenshot directory automatically. Both paths are ignored by Git.

## Runtime configuration

The application reads `config.json` by default. Set `CONFIG_FILE` to use another path.

| Field | Required | Description |
| --- | --- | --- |
| `address` | No | Go listen address. Defaults to `:8080`. |
| `dataDir` | No | SQLite and screenshot directory. Defaults to `data`. |
| `webDir` | No | Compiled frontend directory. Defaults to `web/dist`. |
| `sessionSecret` | Yes | Random value of at least 32 characters. Changing it invalidates every session. |
| `secureCookies` | Yes | Use `false` only for local HTTP. Production requires `true`. |
| `ownerToken` | Yes | Unique random owner credential of at least 16 characters. |
| `players` | Yes | At least one player. IDs and tokens must be unique; `owner` is a reserved ID. |
| `games` | Yes | At least one game with a unique ID, HTTPS URL, and positive `maxScore`. |
| `vision.url` | For uploads | OpenAI-compatible chat-completions endpoint. |
| `vision.model` | For uploads | Vision-capable model identifier. |
| `vision.apiKey` | For uploads | Private provider API key. |
| `ntfy.url` | No | ntfy server base URL, such as `https://ntfy.sh`. |
| `ntfy.topic` | No | ntfy topic. Use a protected or unguessable topic. |
| `ntfy.token` | No | Bearer token when the ntfy server requires authentication. |
| `webPush.publicKey` | No | Public VAPID key. All three Web Push fields must be set together. |
| `webPush.privateKey` | No | Private VAPID key. Keep it secret, stable, and backed up. |
| `webPush.subject` | No | A `mailto:` or HTTPS contact URI for the VAPID identity. |

The OpenRouter defaults in `config.example.json` use `google/gemini-2.5-flash-lite`. Screenshot analysis is limited to two concurrent requests and ten attempts per player, game, and Oslo day.

### Enable browser push

Generate a VAPID key pair once:

```sh
go run ./cmd/vapid
```

Copy the generated keys into `webPush.publicKey` and `webPush.privateKey`, then set a subject such as `mailto:admin@example.com`. Do not regenerate the keys after users subscribe. Browser push requires HTTPS, except on localhost.

### Enable ntfy

Set `ntfy.url` and `ntfy.topic`. Set `ntfy.token` when the server requires bearer authentication. The application sends one completion notification per player and Oslo day.

## Run checks

Install Chromium once:

```sh
cd web
pnpm exec playwright install chromium
cd ..
```

Run backend checks:

```sh
go test -count=1 -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Run frontend and end-to-end checks:

```sh
cd web
pnpm test
pnpm typecheck
pnpm build
pnpm audit --prod
pnpm e2e
```

The E2E suite uses `config.e2e.json`, a fake local vision service, and disposable `.e2e-data` storage.

## Production setup

1. Create private configuration and storage with the host user's IDs:

   ```sh
   cp config.example.json config.json
   install -d -m 700 data
   chmod 600 config.json
   printf 'APP_UID=%s\nAPP_GID=%s\n' "$(id -u)" "$(id -g)" > .env
   chmod 600 .env
   ```

2. Replace every placeholder credential and keep `secureCookies` set to `true`.

3. Build and start the hardened container:

   ```sh
   docker compose build --pull
   docker compose up -d
   docker compose ps
   curl --fail http://127.0.0.1:8080/api/health
   ```

4. Configure the HTTPS reverse proxy before signing in. The container publishes port 8080 only on `127.0.0.1`; do not expose it publicly.

5. Open the HTTPS domain and complete the smoke test in the [deployment runbook](docs/deployment.md).

Docker Compose runs the application as the host UID/GID with a read-only root filesystem, dropped capabilities, and only `data/` writable. It also requires secure cookies through `REQUIRE_SECURE_COOKIES=true`.

## Data and backups

The `data/` directory contains:

- SQLite data and WAL files;
- confirmed screenshots;
- unconfirmed screenshot drafts.

Back up `config.json` and the complete `data/` directory together while the application is stopped. Store the archive off-host. Losing `config.json` loses credentials and the VAPID private key; losing the VAPID key invalidates existing browser push subscriptions.

See the [deployment runbook](docs/deployment.md) for update, backup, restore, rollback, reverse-proxy, and production smoke-test procedures.

## Troubleshooting

- **`validate config` on startup:** Replace all placeholder secrets and check token lengths and uniqueness.
- **Login works locally but the session does not persist:** Set `secureCookies` to `false` for local HTTP development.
- **Compose rejects local development configuration:** Compose intentionally requires secure cookies. Use `go run .` and Vite for local HTTP.
- **`permission denied` under `data/`:** Set `APP_UID` and `APP_GID` in `.env` to the owner of the host data directory.
- **Screenshot analysis fails:** Check `vision.url`, `vision.model`, `vision.apiKey`, provider access, and model availability.
- **Browser notifications do not appear:** Use HTTPS, configure all Web Push fields, and allow notifications in the browser.

## Documentation

- [Product specification](docs/product.md) — workflows, scoring, and constraints
- [Implementation specifications](spec/README.md) — milestone status and acceptance criteria
- [Deployment runbook](docs/deployment.md) — production operation procedures

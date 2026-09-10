# Deploy and operate the leaderboard

This runbook deploys the application on one host behind an existing HTTPS reverse proxy.

## Prerequisites

- Docker Engine with Compose
- A public HTTPS domain
- A reverse proxy on the same host
- A backup destination outside this project directory

Only the reverse proxy may accept public traffic. Port 8080 binds to `127.0.0.1` by default.

## Prepare the host

1. Copy the project to the server.
2. Create the private runtime files:

   ```sh
   cp config.example.json config.json
   install -d -m 700 data
   chmod 600 config.json
   printf 'APP_UID=%s\nAPP_GID=%s\n' "$(id -u)" "$(id -g)" > .env
   chmod 600 .env
   ```

3. Set random player tokens, the owner token, the session secret, service credentials, and VAPID keys in `config.json`.
4. Keep `secureCookies` set to `true`.
5. Back up `config.json` and `data/`. Losing the VAPID private key invalidates browser push subscriptions.

## Configure the reverse proxy

Forward the public HTTPS domain to `http://127.0.0.1:8080`. Configure the proxy to:

- preserve the original `Host` header;
- disable response buffering for `/api/events`;
- allow streaming responses for at least 24 hours;
- allow request bodies up to 11 MiB;
- set `Strict-Transport-Security: max-age=31536000` after HTTPS is confirmed;
- redirect HTTP to HTTPS;
- rate-limit `/api/login` and `/api/games/*/draft`;
- never expose port 8080 publicly.

If the reverse proxy runs in another container, remove the published `ports` entry and connect both services through a private Docker network instead.

## Deploy

```sh
docker compose build --pull
docker compose up -d
docker compose ps
docker compose logs --tail=100 app
curl --fail http://127.0.0.1:8080/api/health
```

Open the public HTTPS URL and verify that login works. Do not use the loopback URL for normal access because production cookies are secure.

## Update

Create a backup before every update:

```sh
BACKUP=/path/outside/project/dailygame-$(date +%Y%m%d-%H%M%S).tar.gz
docker compose stop app
tar -czf "$BACKUP" config.json data
docker compose start app
tar -tzf "$BACKUP"
docker image tag dailygame-leaderboard:local dailygame-leaderboard:rollback

git pull --ff-only
docker compose build --pull
docker compose up -d
docker compose ps
curl --fail http://127.0.0.1:8080/api/health
```

Review `docker compose logs --tail=100 app` after startup.

## Back up

Stop the application briefly so the SQLite database, its WAL files, and screenshots are one consistent snapshot:

```sh
BACKUP=/path/outside/project/dailygame-$(date +%Y%m%d-%H%M%S).tar.gz
docker compose stop app
tar -czf "$BACKUP" config.json data
docker compose start app
tar -tzf "$BACKUP"
```

Configure the existing backup system to copy these archives off the application host. Test a restore after the first backup and periodically afterward.

## Restore

**Warning:** Restore replaces the current configuration and retained data.

```sh
docker compose down
mv config.json config.json.before-restore
mv data data.before-restore
tar -xzf /path/to/dailygame-backup.tar.gz
chmod 600 config.json
chmod 700 data
docker compose up -d
curl --fail http://127.0.0.1:8080/api/health
```

Log in through the HTTPS domain and verify a retained score and screenshot before deleting the `.before-restore` copies.

## Roll back

A rollback must restore both the retained previous image and the backup created before its migration:

```sh
docker compose down
docker image tag dailygame-leaderboard:rollback dailygame-leaderboard:local
mv config.json config.json.failed
mv data data.failed
tar -xzf /path/to/pre-update-backup.tar.gz
docker compose up -d --no-build
curl --fail http://127.0.0.1:8080/api/health
```

If the rollback image was removed, check out the previous known-good commit and rebuild it before starting the restored data.

Verify login, one retained score, and its screenshot through the HTTPS domain.

## Production smoke test

Use temporary test players if you do not want to alter real scores.

1. Sign in as two players in separate browsers.
2. Submit both games for one player.
3. Confirm that the second browser updates without a reload.
4. Confirm that ntfy and browser push each send one completion message.
5. Correct one score as the owner and confirm both player dashboards update.
6. Reopen one submission and confirm the player can upload a replacement.
7. Recreate the container with `docker compose up -d --force-recreate`.
8. Confirm that the score and screenshot remain available.
9. Confirm that an unauthenticated request to a screenshot URL returns `401`.

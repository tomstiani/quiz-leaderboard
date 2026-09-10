# Daily Game Leaderboard

A private daily leaderboard for a small quiz team.

## Documentation

- [Product specification](docs/product.md) — MVP scope, workflows, scoring, and constraints
- [Implementation specifications](spec/README.md) — ordered milestones, task status, and acceptance criteria

## Local development

Requires Go 1.26, Node.js 22, and pnpm.

1. Create private configuration:

   ```sh
   cp config.example.json config.json
   ```

2. Start the backend:

   ```sh
   go run .
   ```

3. In another terminal, start the frontend:

   ```sh
   cd web
   pnpm install
   pnpm dev
   ```

Open <http://localhost:5173>. Vite proxies `/api` requests to Go on port 8080.

## Checks

```sh
go test ./...
cd web && pnpm typecheck && pnpm build
```

## Production build

Build and run the same container used by Docker Compose:

```sh
cp config.example.json config.json
mkdir -p data
docker compose up --build
```

Open <http://localhost:8080> and verify <http://localhost:8080/api/health> returns `{"status":"ok"}`.

Runtime configuration lives in the ignored `config.json`. Set `secureCookies` to `true` behind production HTTPS. SQLite and screenshots live in the ignored `data/` directory. Back up those two runtime paths outside version control.

## Project status

Milestone progress is tracked in [`spec/README.md`](spec/README.md).

# Setup & Development Guide

## Prerequisites

- Go 1.26+
- Node.js 20+ (for collab sidecar and frontend)
- Docker + Docker Compose (for full stack)

## Project Structure

```
mostdoc/
├── cmd/server/main.go        Go API entrypoint
├── collab/                   Node.js Hocuspocus sidecar
│   ├── server.js
│   └── package.json
├── frontend/                 React + Vite frontend
│   ├── src/
│   └── package.json
├── internal/                 Go packages
│   ├── auth/
│   ├── pages/
│   ├── spaces/
│   ├── sse/
│   ├── db/
│   └── httputil/
├── migrations/               golang-migrate SQL files
├── docker/                   Nginx config
├── data/                     SQLite DB + markdown files (gitignored)
├── docker-compose.yml
├── Dockerfile.api
└── docs/                     Documentation
```

## Quick Start (Docker Compose)

The fastest way to run the full stack:

```bash
# 1. Build and start all services
docker-compose up --build

# 2. Open http://localhost:8081
# 3. Register the first admin account at /register
```

Services:
- `go-api` — Go REST API on :3000
- `collab` — Hocuspocus WebSocket on :3001, health on :3002
- `nginx` — Reverse proxy on :8081, serves frontend dist

### Docker Compose Configuration

```yaml
# Excerpt from docker-compose.yml
services:
  go-api:
    build:
      context: .
      dockerfile: Dockerfile.api
    volumes:
      - ./data:/data
    environment:
      - DATA_DIR=/data
      - PORT=3000

  collab:
    build:
      context: ./collab
    volumes:
      - ./data:/data
    environment:
      - API_URL=http://go-api:3000
      - PORT=3001

  nginx:
    image: nginx:alpine
    ports:
      - "8081:80"
    volumes:
      - ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./frontend/dist:/usr/share/nginx/html:ro
```

## Development (Local)

### 1. Go API

```bash
# Install Go dependencies
go mod download

# Generate sqlc queries (if .sql files changed)
# sqlc generate

# Run migrations (auto-run on startup)
# Migrations are in ./migrations/

# Start the API
go run cmd/server/main.go
# Server listens on :3000
# DATA_DIR defaults to ./data
```

### 2. Collaboration Sidecar

```bash
cd collab
npm install

# Start sidecar
API_URL=http://localhost:3000 PORT=3001 node server.js
```

### 3. Frontend

```bash
cd frontend
npm install

# Development server with HMR
npm run dev
# Vite dev server runs on :5173, proxies /api to localhost:3000
```

### 4. Nginx (optional for local dev)

For local development, Vite's dev server proxy is usually sufficient. Nginx is primarily used in Docker.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_DIR` | `./data` | Path to data directory (SQLite DB, markdown files, JWT key) |
| `PORT` | `3000` | Go API port |
| `MIGRATIONS_PATH` | `file://migrations` | golang-migrate source URL |
| `JWT_SECRET` | *(auto-generated)* | JWT signing secret. If unset, auto-generated and persisted to `data/jwt.key` and DB `settings` table |
| `BCRYPT_COST` | `12` | bcrypt cost factor (4–31) |
| `API_URL` | `http://localhost:3000` | URL the collab sidecar uses to reach Go API |
| `PORT` (collab) | `3001` | Hocuspocus WebSocket port |
| `DB_PATH` (collab) | `/data/collab.db` | *(legacy)* Sidecar no longer uses its own DB |

## Database Migrations

Migrations use `golang-migrate` and run automatically on Go API startup.

```bash
# Create a new migration
migrate create -ext sql -dir migrations -seq my_migration

# Run migrations manually
migrate -database "sqlite3://data/wiki.db" -path migrations up

# Rollback one
migrate -database "sqlite3://data/wiki.db" -path migrations down 1
```

### Migration History

| File | Description |
|------|-------------|
| `001_initial.up.sql` | `pages` and `page_snapshots` tables (Iteration 1) |
| `002_auth.up.sql` | `users`, `refresh_tokens`, `settings` tables (Iteration 2) |
| `003_spaces.up.sql` | `spaces` table (Iteration 2) |
| `004_pages_enrich.up.sql` | Add FKs to `pages`, indexes, `idx_pages_space_slug` unique (Iteration 2) |

## Building for Production

### Frontend

```bash
cd frontend
npm run build
# Output: frontend/dist/
```

### Go API

```bash
# Build binary
go build -o mostdoc cmd/server/main.go

# Or with Docker
docker build -f Dockerfile.api -t mostdoc-api .
```

The `Dockerfile.api` is a multi-stage build that compiles the Go binary and copies migrations into the image.

## Testing

### Go Tests

```bash
go test ./...
```

Current test coverage:
- `internal/auth/handler_test.go`
- `internal/pages/handler_test.go`

### Frontend Tests

```bash
cd frontend
npm test
```

### E2E Tests (Planned)

Playwright E2E tests for critical flows (login → create page → edit → logout).

## Debugging

### Go API

```bash
# Structured JSON logs to stdout
# Use jq for pretty printing:
go run cmd/server/main.go 2>&1 | jq .
```

### Sidecar

The sidecar logs all internal API calls and auth results:
```
[onLoadDocument] loading: <docId>
[onAuthenticate] success: <userId> <permission>
[onStoreDocument] doc: <docId> markdown: <length>
```

### Frontend

The collaborative editor has debug logging for Yjs updates and awareness states. Check the browser console.

## Common Issues

### SQLite `database is locked`

- Ensure `busy_timeout=5000` is set in the DSN (already configured in `internal/db/db.go`)
- `SetMaxOpenConns(1)` prevents concurrent write attempts

### Sidecar cannot reach Go API

- Verify `API_URL` is correct (Docker network uses service names: `http://go-api:3000`)
- Check that Go API is running and healthy

### JWT secret mismatch after wipe

- If `data/` is deleted, a new JWT secret is auto-generated
- Existing refresh tokens become invalid; users must re-login
- Set `JWT_SECRET` env var for consistency across wipes

### Frontend 401 on API calls

- Access token may have expired. The frontend auto-refreshes via `/api/auth/refresh` using the httpOnly cookie.
- If refresh also fails, user is redirected to `/login`.

## Deployment Checklist

- [ ] Set `JWT_SECRET` environment variable
- [ ] Set `BCRYPT_COST` (default 12 is usually fine)
- [ ] Build frontend: `cd frontend && npm run build`
- [ ] Ensure `frontend/dist/` exists for nginx volume mount
- [ ] Run `docker-compose up -d`
- [ ] Verify: `curl http://localhost:8081/health`
- [ ] Register first admin at `/register`

## HTTPS (Planned — Iteration 5)

HTTPS with automatic certificate renewal via Certbot will be added in Iteration 5. For now, Nginx serves HTTP on port 8081.

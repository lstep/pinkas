# Pinkas

A self-hosted, open-source collaborative Markdown wiki. Multiple users can edit documents simultaneously in real time (powered by Yjs + Hocuspocus). All documents are stored as plain `.md` files on the server filesystem — readable and portable without the application.

## Features

- **Real-time collaboration** — Multiple users edit the same document simultaneously with live cursor awareness and conflict resolution (Yjs + Hocuspocus)
- **WYSIWYG Markdown editor** — Rich text editing via Tiptap; supports headings, code blocks, blockquotes, lists, inline images, video embeds, and file attachments
- **Granular permissions** — Role-based access (viewer / editor / admin) at the space, directory, and page level
- **Spaces & directories** — Organize pages into a tree structure within named spaces
- **Full-text search** — Powered by SQLite FTS5, search across all pages with ranking and deduplication
- **Page history** — Automatic snapshots on every save; browse and restore previous versions
- **File attachments** — Upload images, videos, and files per page (stored on disk)
- **MCP server** — Model Context Protocol server (port `3100`) for AI tool integration (search, read, create, update pages)
- **SSE real-time events** — Server-Sent Events for live UI updates (page changes, directory moves, etc.)
- **Self-hosted & portable** — Single SQLite database, plain Markdown files on disk, Docker Compose deployment
- **Continuous backup** — Litestream replication of the SQLite database to S3, SFTP, or local volume

## Ports Overview

| Service | Port | Description |
|---------|------|-------------|
| Go API | `3000` | REST API, auth, permissions, SSE, file serving |
| Hocuspocus (collab) | `3001` | WebSocket for real-time collaboration |
| Hocuspocus health | `3002` | Sidecar health endpoint |
| MCP server | `3100` | Model Context Protocol (SSE + messages) |
| Nginx / Frontend | `8081` | Reverse proxy (production) |
| Vite dev server | `5173` | Frontend dev mode (development only) |

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Go 1.26+](https://go.dev/dl/) (for local development only)
- [Node.js 24+](https://nodejs.org/) (for local development only)

### Running with Docker Compose

```bash
# Build and start all services
docker compose up -d --build

# Check that everything is running
docker compose ps

# View logs
docker compose logs -f
```

The application will be available at:

| Service | URL |
|---------|-----|
| Frontend | http://localhost:8081 |
| Health check | http://localhost:8081/health |

To stop:

```bash
docker compose down
```

### Local Development

#### 1. Go API

```bash
# Install dependencies
go mod download

# Run the server
go run ./cmd/server
```

The API listens on TCP port `3000` by default.

#### 2. Collab Sidecar

```bash
cd collab
npm install
node server.js
```

The sidecar listens on port `3001` (WebSocket) and `3002` (health).

#### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server starts on port `5173`.

#### 4. Nginx (optional for local dev)

```bash
# Build the frontend first
cd frontend && npm run build && cd ..

# Start nginx with the local config
docker run -d --name pinkas-nginx \
  -p 8081:80 \
  -v $(pwd)/docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro \
  -v $(pwd)/frontend/dist:/usr/share/nginx/html:ro \
  nginx:alpine
```

## Architecture

```
Browser (React + Tiptap + Yjs)
  ├── WebSocket  →  Nginx :80/collab/*  →  Node.js Hocuspocus :3001
  └── HTTP       →  Nginx :80/api/*     →  Go API :3000

Go API :3000
  ├── REST API     (/api/*)
  ├── File serve   (/files/*)
  ├── SSE stream   (/api/events)
  └── Internal     (/internal/auth, /internal/save, /internal/restore, /internal/cleanup)

Shared volume /data
  ├── wiki.db              SQLite database (WAL mode)
  ├── docs/{space}/{id}.md  Markdown files
  └── attachments/{id}/     Uploaded files
```

### Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Go API | Go 1.26 + net/http | REST API, auth, permissions, storage, MCP |
| Collab | Node.js + Hocuspocus + Yjs | Real-time WebSocket collaboration |
| Frontend | React + TypeScript + Vite + Tiptap | WYSIWYG Markdown editor |
| Database | SQLite (mattn/go-sqlite3) | Sole database, WAL mode |
| Reverse proxy | Nginx | Routes WebSocket and HTTP traffic |
| Backup | Litestream | Continuous SQLite replication |

## Project Structure

```
├── cmd/server/              Go application entry point
├── internal/
│   ├── attachments/         File upload handling
│   ├── auth/                JWT auth, middleware, user CRUD
│   ├── db/                  Database connection, migrations, seed
│   │   └── query/           SQLC generated Go code
│   ├── directories/         Directory CRUD and tree operations
│   ├── groups/              Group CRUD and membership
│   ├── httputil/            JSON responses, error helpers
│   ├── mcp/                 MCP server (SSE + tools)
│   ├── mcptokens/           MCP API token management
│   ├── pages/               Page CRUD, history, search, snapshots
│   ├── permissions/         Permission levels, resolver, middleware
│   ├── spaces/              Space CRUD
│   └── sse/                 Server-Sent Events hub
├── collab/                  Node.js Hocuspocus sidecar (Yjs)
├── frontend/                React + TypeScript + Vite + Tiptap
├── docker/                  Nginx configuration
├── migrations/              SQL migration files
├── data/                    Runtime data (SQLite, docs, attachments)
├── docker-compose.yml       Docker orchestration
└── Dockerfile.api           Go API Docker image
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Go API TCP port |
| `DATA_DIR` | `./data` | Path for SQLite database and documents |
| `BCRYPT_COST` | `12` | Password hashing cost factor |
| `JWT_SECRET` | auto-generated | JWT signing key (auto-saved to `data/jwt.key` if not set) |

### Port Mapping

The default Docker Compose setup exposes the application on port `8081`. To change it, edit `docker-compose.yml`:

```yaml
services:
  nginx:
    ports:
      - "YOUR_PORT:80"
```

## Data Persistence

All runtime data is stored in the `./data` directory:

```
data/
├── wiki.db              SQLite database
├── wiki.db-wal          SQLite WAL file
├── wiki.db-shm          SQLite shared memory
├── docs/                Markdown documents
│   └── {space-slug}/
│       └── {page-id}.md
└── attachments/         Uploaded files
    └── {page-id}/
        └── {uuid}-{filename}
```

The `data/` directory is bind-mounted into the Docker containers. It is excluded from Git via `.gitignore`.

## Backup

Litestream provides continuous replication of the SQLite database to a configurable target (local volume, S3, or SFTP). Configure via environment variables in `docker-compose.yml`.

## License

MIT

# Architecture

## Overview

Mostdoc is a self-hosted collaborative markdown wiki. Multiple users edit documents simultaneously with real-time cursors. Documents are stored as plain `.md` files on disk. Access is controlled by a permission system with users, groups, and tree-inherited grants.

## Components

```
Browser (React + Tiptap + Yjs)
  ├── WebSocket  →  Nginx :8081/collab/*  →  Node.js Hocuspocus :3001
  └── HTTPS      →  Nginx :8081/api/*     →  Go API :3000

Go API :3000
  ├── REST API     (/api/*)
  ├── File serve   (/files/*)  — planned
  ├── MCP          (/mcp)      — planned
  ├── SSE stream   (/api/events)
  └── Internal     (/internal/auth, /internal/save, /internal/load, /internal/restore, /internal/cleanup)

Shared Docker volume /data
  ├── wiki.db              SQLite database
  ├── docs/{space}/{id}.md  Markdown files
  └── attachments/{id}/     Uploaded files — planned
```

### Go API

All business logic lives here. Built with Go stdlib `net/http`. No third-party HTTP framework.

- **Auth** (`internal/auth/`): JWT issue/validate, bcrypt password hashing, refresh token rotation, login/register/logout/me endpoints, auth middleware.
- **Pages** (`internal/pages/`): Page CRUD, move, breadcrumb (directory ancestors), slug generation with collision resolution, REST handlers, internal handlers for sidecar communication.
- **Directories** (`internal/directories/`): Directory CRUD, tree operations (children, breadcrumb, move), slug generation. Directories are self-referential containers; pages live inside them.
- **Spaces** (`internal/spaces/`): Space CRUD with default permissions, MCP write flags, snapshot retention.
- **SSE** (`internal/sse/`): Server-Sent Events hub with `sync.Map` subscribers and in-memory ring buffer for replay. Events: `page_created`, `page_updated`, `page_moved`, `page_deleted`, `directory_created`, `directory_updated`, `directory_moved`, `directory_deleted`.
- **DB** (`internal/db/`): SQLite connection (WAL mode, busy_timeout=5000ms, max 1 open connection). Migrations via `golang-migrate`. Type-safe queries via `sqlc`.
- **HTTP utilities** (`internal/httputil/`): Structured `AppError`, JSON helpers.

### Collaboration Sidecar

Minimal Node.js process (~120 lines) running Hocuspocus. Accepts WebSocket connections, calls Go internal endpoints for auth and persistence.

- `onLoadDocument`: fetches latest Yjs snapshot from Go via `/internal/load`
- `onAuthenticate`: validates JWT + resolves permission via `/internal/auth`
- `onStoreDocument` / `onDisconnect`: serializes Yjs → ProseMirror → Markdown, calls `/internal/save`
- No business logic; no database of its own (dropped `@hocuspocus/extension-database`)

### Frontend

React + TypeScript + Vite + Tailwind.

- **Routing**: `react-router-dom` v7 — `/login`, `/register`, `/`, `/s/:spaceSlug/*`
- **Auth store**: Zustand with `persist` middleware (localStorage for access token)
- **Tree store**: Zustand for sidebar tree state with optimistic updates
- **Editor store**: Zustand for collaborative editor config (docId, providerUrl)
- **API client** (`api/pages.ts`): Fetch wrapper with automatic 401 refresh retry via `/api/auth/refresh` (httpOnly cookie)
- **SSE client** (`api/sse.ts`): EventSource with `Last-Event-ID` reconnection
- **Components**: Sidebar (lazy-load, create/rename/move/delete), Breadcrumb, CollaborativeEditor (Tiptap + HocuspocusProvider + awareness cursors), ProtectedRoute

## Data Model

### SQLite Schema (current)

```sql
-- Spaces
create table spaces (
  id text primary key,
  workspace_id text default 'default',
  name text not null,
  slug text unique not null,
  default_permission text default 'none',
  mcp_write_enabled integer default 1,
  snapshot_retention_days integer default null,
  created_at integer default (strftime('%s', 'now'))
);

-- Directories (self-referential adjacency list)
create table directories (
  id text primary key,
  space_id text references spaces(id),
  parent_id text references directories(id),
  name text not null,
  slug text not null,
  position integer default 0,
  icon text,
  created_by text references users(id),
  created_at integer default (strftime('%s', 'now')),
  updated_at integer default (strftime('%s', 'now'))
);

-- Pages (leaves that live inside directories)
create table pages (
  id text primary key,
  space_id text references spaces(id),
  directory_id text references directories(id),
  title text default 'Untitled',
  slug text default 'untitled',
  position integer default 0,
  created_by text references users(id),
  created_at integer default (strftime('%s', 'now')),
  updated_at integer default (strftime('%s', 'now')),
  icon text
);

-- Page snapshots (Yjs + Markdown)
create table page_snapshots (
  id text primary key,
  page_id text references pages(id),
  yjs_snapshot blob,
  markdown text,
  author_id text,
  created_at integer default (strftime('%s', 'now')),
  label text,
  is_compacted integer default 0
);

-- Users
create table users (
  id text primary key,
  email text unique not null,
  name text,
  password_hash text not null,
  global_role text default 'user',
  created_at integer default (strftime('%s', 'now'))
);

-- Refresh tokens
create table refresh_tokens (
  id text primary key,
  user_id text not null references users(id),
  token_hash text not null,
  expires_at integer not null,
  created_at integer default (strftime('%s', 'now'))
);

-- Settings (key-value runtime config)
create table settings (
  key text primary key,
  value text not null
);
```

### Indexes

- `idx_spaces_slug` on `spaces(slug)`
- `idx_directories_space_id` on `directories(space_id)`
- `idx_directories_parent_id` on `directories(parent_id)`
- `idx_directories_slug` on `directories(slug)`
- `idx_directories_space_slug` unique on `directories(space_id, slug)`
- `idx_pages_space_id` on `pages(space_id)`
- `idx_pages_directory_id` on `pages(directory_id)`
- `idx_pages_slug` on `pages(slug)`
- `idx_pages_space_slug` unique on `pages(space_id, slug)`
- `idx_refresh_tokens_user_id` on `refresh_tokens(user_id)`
- `idx_refresh_tokens_expires_at` on `refresh_tokens(expires_at)`

## File Layout

```
data/
├── wiki.db                 SQLite database
├── docs/{space-slug}/{page-id}.md   Markdown exports
└── jwt.key                 Auto-generated JWT secret (if not in env/DB)
```

## Key Design Decisions

1. **Dual-format invariant**: Yjs binary in SQLite is the source of truth. Markdown files on disk are derived exports only. Go never parses Markdown.
2. **Unix domain sockets**: Sidecar↔Go communication uses Unix sockets for lower latency (currently HTTP over Docker network; Unix socket planned for production).
3. **CGO SQLite**: `mattn/go-sqlite3` chosen over pure Go for 3-10x write performance.
4. **sqlc**: Compile-time type-safe queries for all CRUD operations.
5. **SSE over WebSocket for tree updates**: SSE is sufficient for one-directional broadcasts; WebSocket reserved for collaborative editing.
6. **Permission stub**: Iteration 2 returns `admin` for all authenticated users. Full recursive CTE resolution deferred to Iteration 3.

## Authentication Flow

```
Login → bcrypt verify → issue access + refresh tokens
  → access token in JSON response (stored localStorage)
  → refresh token in httpOnly cookie (7-day expiry)

Refresh → read cookie → validate hash → rotate (delete old, create new)
  → new access token returned, new refresh cookie set

Logout → delete refresh token from DB → clear cookie
```

## SSE Event Flow

```
Page mutation handler → SSE Hub.Broadcast() → ring buffer push
  → all active EventSource connections receive formatted event
  → frontend tree store updates optimistically + from SSE
```

## Go Package Structure

```
internal/
├── auth/          JWT, bcrypt, users, refresh tokens, middleware
├── pages/         Page CRUD, move, breadcrumb, internal sidecar handlers
├── directories/   Directory CRUD, tree ops, slug utils
├── spaces/        Space CRUD
├── sse/           Hub, ring buffer, SSE handler
├── db/            Connection, migrations
│   └── query/     sqlc generated code
├── httputil/      JSON helpers, AppError
```

## Technology Stack

| Layer | Technology | License |
|-------|-----------|---------|
| Frontend | React + TypeScript + Vite + Tailwind + Tiptap + HocuspocusProvider + Zustand | MIT |
| Collab sidecar | Node.js + Hocuspocus + Yjs + y-prosemirror + prosemirror-markdown | MIT |
| Go REST API | Go stdlib net/http + golang-jwt/jwt + mattn/go-sqlite3 + sqlc + golang-migrate | BSD-3 / MIT |
| Database | SQLite (WAL mode, busy_timeout=5000) | Public Domain |
| Reverse proxy | Nginx | BSD |
| Deployment | Docker Compose | Apache-2 |

## Iteration Status

| Iteration | Status | Delivered |
|-----------|--------|-----------|
| 1 — Collaborative Editor | ✅ Committed | Single page, real-time editing, stub auth, docker-compose |
| 2 — Multi-Page Wiki | ✅ Done | Auth, spaces, directory/page split, sidebar, SSE, drag-drop reorder, slug routing, refresh tokens |
| 3 — Multi-User Access Control | 📋 Planned | Permissions, groups, invites, WebSocket auth enforcement |
| 4 — History & Attachments | 📋 Planned | Snapshots, diff, restore, file upload |
| 5 — Search & Production | 📋 Planned | FTS5, Litestream, HTTPS, rate limiting |
| 6 — MCP Integration | 📋 Planned | MCP endpoint, tools, resources, prompts, audit log |

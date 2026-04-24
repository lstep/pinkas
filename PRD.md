# Product Requirements Document

**Product:** Collaborative Markdown Wiki (Docmost Clone)
**Version:** 1.2
**Date:** April 2026
**Status:** Approved — Design Decisions Resolved

---

## Table of Contents

1. [Overview](#1-overview)
2. [Goals & Non-Goals](#2-goals--non-goals)
3. [Users & Personas](#3-users--personas)
4. [Functional Requirements](#4-functional-requirements)
5. [Non-Functional Requirements](#5-non-functional-requirements)
6. [System Architecture](#6-system-architecture)
7. [Data Model](#7-data-model)
8. [API Surface](#8-api-surface)
9. [Permissions Model](#9-permissions-model)
10. [Document History](#10-document-history)
11. [MCP Integration](#11-mcp-integration)
12. [Iterative Delivery Plan](#12-iterative-delivery-plan)
13. [Risks](#13-risks)
14. [Out of Scope](#14-out-of-scope)

---

## 1. Overview

A self-hosted, open-source collaborative wiki. Multiple users can edit documents simultaneously in real time. All documents are stored as plain Markdown files on the server filesystem — readable and portable without the application. Access is controlled by a fine-grained permission system with users, groups, and tree-inherited permissions.

### 1.1 Problem Statement

Teams need a shared knowledge base that is:
- **Truly self-hosted** — no data leaves the server, no dependency on a third-party SaaS
- **Portable** — documents remain readable as plain `.md` files if the application is ever decommissioned
- **Collaborative** — multiple editors can work on the same document simultaneously without conflicts
- **Controlled** — access is scoped per document tree, not just per workspace

Existing open-source wikis either store content in proprietary formats, require heavy infrastructure (PostgreSQL, Redis, object storage), or lack real-time collaboration.

### 1.2 Solution Summary

A web application with a Go REST API backend, a minimal Node.js WebSocket sidecar for real-time collaboration (Hocuspocus + Yjs), a React frontend (Tiptap editor), and SQLite as the sole database. All file storage is local filesystem. The entire stack runs from a single `docker-compose up`.

---

## 2. Goals & Non-Goals

### Goals

- Real-time collaborative editing with visible cursors
- Documents stored as plain `.md` files on disk at all times
- Fine-grained permissions: per-user, per-group, inherited down the document tree
- Complete document history with diff view and one-click restore
- Image and file attachments pasted directly into the editor
- Full-text search across all accessible documents
- MCP (Model Context Protocol) entry point for AI agent access
- 100% open-source stack, MIT/Apache/BSD licences only
- Single-server Docker Compose deployment, zero external services

### Non-Goals

- Multi-server / distributed deployment (single node only in v1)
- Object storage integration (S3, MinIO — local filesystem only)
- LDAP / SSO / OAuth login (email + password only in v1)
- Mobile native apps (responsive web only)
- Offline editing (online only)
- Comment threads on documents
- Real-time notification system beyond the tree-update SSE stream

---

## 3. Users & Personas

### Workspace Admin
Creates and manages the workspace. Invites users, creates groups and spaces, sets global roles. Has implicit admin permission on all documents.

### Editor
A regular team member with edit access to documents in their assigned spaces or pages. Can create pages, write content, upload attachments, and manage the documents they own.

### Viewer
A team member with read-only access to specific document trees. Can browse, search, and read documents but cannot modify content or settings.

### AI Agent (MCP client)
An external AI system (e.g. Claude Desktop, a CI pipeline, an automation script) that accesses the wiki through the MCP endpoint. Subject to the same permission model as human users — granted a scoped API token by an admin.

---

## 4. Functional Requirements

### 4.1 Authentication

| ID | Requirement |
|----|-------------|
| F-AUTH-01 | The system shall allow registration of the first admin account when no users exist. A dedicated `/setup` route shall be provided; the API also returns a special status when zero users exist. |
| F-AUTH-02 | Users shall authenticate with email and password and receive a JWT access token and a refresh token. |
| F-AUTH-03 | Access tokens shall expire after 15 minutes; refresh tokens after 7 days. |
| F-AUTH-04 | Refresh tokens shall be rotated on each use (old token invalidated). Refresh tokens are stored hashed in SQLite. |
| F-AUTH-05 | Admins shall be able to invite new users by email. An invite token is generated; the invitee sets their password on first login. The full invite URL is returned in the API response (no SMTP in v1 — admin shares manually). |
| F-AUTH-06 | Password hashing shall use bcrypt with cost factor 12 (configurable via `BCRYPT_COST` env var). |
| F-AUTH-07 | JWT secret shall be configurable via env var; if not set, auto-generated on first startup and persisted to `data/jwt.key`. |
| F-AUTH-08 | Auth endpoints shall be rate-limited using an in-memory token bucket per IP. |

### 4.2 Spaces

| ID | Requirement |
|----|-------------|
| F-SPC-01 | The workspace shall be organised into named Spaces, each containing a document tree. |
| F-SPC-02 | Workspace admins shall be able to create, rename, and delete spaces. |
| F-SPC-03 | Each space shall have a configurable default permission (none, viewer, editor) applied to users without an explicit grant. |
| F-SPC-04 | Each space shall have a per-space `mcp_write_enabled` flag that prevents AI agents from writing to the space regardless of token scope. |
| F-SPC-05 | Each space shall have a configurable `snapshot_retention_days` (default: unlimited) for automatic snapshot cleanup. |

### 4.3 Document Editing

| ID | Requirement |
|----|-------------|
| F-EDIT-01 | The editor shall be a rich WYSIWYG Markdown editor supporting headings, lists, code blocks, tables, images, and file links. |
| F-EDIT-02 | Multiple users shall be able to edit the same document simultaneously; all editors shall see each other's changes in real time with no conflicts. |
| F-EDIT-03 | Each active editor's cursor position and selection shall be visible to all other editors, labelled with their name. |
| F-EDIT-04 | All documents shall be persisted as plain `.md` files on the server filesystem. The files shall be readable without the application. |
| F-EDIT-05 | Document content shall be saved automatically after a quiet period of editing (debounce), on session end, and on a 5-minute interval timer. |

### 4.4 Document Tree

| ID | Requirement |
|----|-------------|
| F-TREE-01 | Documents shall be organised in a hierarchical tree within each space. |
| F-TREE-02 | The sidebar shall display the document tree with collapsible nodes; children shall be loaded lazily on expand. |
| F-TREE-03 | Users shall be able to reorder documents within a parent and move documents between parents via drag-and-drop. |
| F-TREE-04 | Users shall be able to create, rename, move, and delete documents from the sidebar context menu. |
| F-TREE-05 | The sidebar shall update in real time when another user creates, renames, moves, or deletes a document (Server-Sent Events). SSE uses `event_id` in each event, `Last-Event-ID` header on reconnect, and a ring buffer for replay of missed events. |
| F-TREE-06 | A breadcrumb trail shall display the current document's position in the tree. |

### 4.5 Permissions

| ID | Requirement |
|----|-------------|
| F-PERM-01 | Access to documents shall be controlled by permission grants with four levels: `none`, `viewer`, `editor`, `admin`. |
| F-PERM-02 | Grants shall be assignable to individual users or to groups. |
| F-PERM-03 | Permissions shall cascade from parent to child documents by default. |
| F-PERM-04 | A permission grant on a child document shall override the inherited grant from its parent. |
| F-PERM-05 | The effective permission for a user is the maximum (most permissive) grant found when walking up the ancestor chain, considering both direct user grants and all group memberships. |
| F-PERM-06 | The permission editor shall be accessible from the document context menu and shall be visible only to users with admin permission on that document. |
| F-PERM-07 | Workspace admins shall be able to create and manage groups and add or remove members. |

### 4.6 Document History

| ID | Requirement |
|----|-------------|
| F-HIST-01 | The system shall automatically save a versioned snapshot of each document after each debounced save, on session end, and on a 5-minute interval. |
| F-HIST-02 | Users shall be able to create a manually labelled snapshot at any time. |
| F-HIST-03 | Users shall be able to browse the version history of a document in a sidebar panel, showing author, timestamp, and optional label. |
| F-HIST-04 | Users shall be able to view a visual diff between any two snapshots, with additions highlighted green and deletions highlighted red. |
| F-HIST-05 | Users shall be able to restore a document to any historical snapshot. The restore shall be applied live — all editors currently viewing the document shall see the restored content immediately without reloading. |
| F-HIST-06 | Before applying a restore, the system shall automatically save a pre-restore snapshot so the action can be undone. |

### 4.7 Attachments

| ID | Requirement |
|----|-------------|
| F-ATT-01 | Users shall be able to paste or drag-and-drop images and files directly into the editor. |
| F-ATT-02 | Attachments shall be saved to the local server filesystem alongside the document's Markdown file. |
| F-ATT-03 | Attachments shall be served through the API with a permission check — only users with at least viewer access to the document may download an attachment. |
| F-ATT-04 | The maximum attachment file size shall be configurable (default 50 MB). |

### 4.8 Search

| ID | Requirement |
|----|-------------|
| F-SRCH-01 | The system shall provide full-text search across all documents the user has at least viewer access to. |
| F-SRCH-02 | Search results shall include a document title, an excerpt with matched terms highlighted, and a relevance score. |
| F-SRCH-03 | Search results shall be filterable by space. |

### 4.9 MCP Integration

| ID | Requirement |
|----|-------------|
| F-MCP-01 | The system shall expose a Model Context Protocol (MCP) endpoint at `POST /mcp` using the Streamable HTTP transport. |
| F-MCP-02 | AI agents shall authenticate with the same JWT bearer tokens as human users, with optional scope restrictions (`wiki:read`, `wiki:write`, `wiki:admin`, `wiki:history`). |
| F-MCP-03 | All MCP tool calls shall enforce the same permission model as the REST API — an AI agent cannot access documents a human user with the same token could not access. |
| F-MCP-04 | The system shall expose MCP resources for spaces, document trees, page content, page metadata, and page history. |
| F-MCP-05 | The system shall expose MCP tools for: search, read page, create page, update page, move page, delete page, restore snapshot, set permission, invite user, create group, and add group member. |
| F-MCP-06 | The system shall expose MCP prompts for: summarise page, draft page, generate changelog, and Q&A over the wiki. |
| F-MCP-07 | Every MCP tool call shall be logged to an audit table (tool name, user, parameters, result, timestamp). |
| F-MCP-08 | Workspace admins shall be able to disable AI write access per space via a `mcp_write_enabled` flag. |

---

## 5. Non-Functional Requirements

| ID | Category | Requirement |
|----|----------|-------------|
| N-01 | Backend language | All server-side business logic shall be implemented in Go using the standard library `net/http` server. No third-party HTTP framework. Global HTTP timeouts: 30s read, 60s write. Per-handler context propagation. Structured JSON logs to journald. |
| N-02 | Collaboration runtime | The Yjs WebSocket layer shall be implemented as a minimal Node.js sidecar (~150 lines) running Hocuspocus. The sidecar shall have no business logic; it calls Go API endpoints via Unix domain sockets for auth and persistence. Sidecar persists Yjs to SQLite via `@hocuspocus/extension-sqlite` and reloads on restart. |
| N-03 | Database | SQLite (CGO via mattn/go-sqlite3) shall be the sole database. WAL mode enabled, `busy_timeout = 5000 ms`. No separate database server. Migrations auto-run on startup via golang-migrate. Type-safe queries via sqlc. SQLite corruption requires manual intervention. |
| N-04 | File storage | All files (Markdown, attachments) shall be stored on the local server filesystem. No object storage. Bind mount to `./data` for easy debugging. |
| N-05 | Deployment | The complete stack shall start with `docker-compose up`. Four containers: `go-api`, `collab`, `nginx`, `litestream`. One shared data volume. Graceful shutdown via `http.Server.Shutdown()` with 30s timeout; Docker `stop_grace_period=30s`. |
| N-06 | Portability | All documents shall remain readable as plain `.md` files if the application is removed. |
| N-07 | Open source | Every dependency shall be MIT, Apache-2, or BSD-3 licensed. Dual-licensed or commercial dependencies are not permitted. |
| N-08 | Backup | SQLite shall be continuously replicated by Litestream to a configurable target (local volume or remote SFTP/S3). |
| N-09 | Extensibility | New features shall be added as Go packages inside `internal/` without restructuring the core. |
| N-10 | Security | JWT tokens shall be validated on every request. Permissions shall be enforced on every endpoint including file serving, the WebSocket connection, and MCP tool calls. The `data/` directory shall never be exposed as a static file server. Files served through Go API with JWT + permission check, `filepath.Clean()` + validation within attachments directory. |
| N-11 | Error handling | All errors shall use a structured `AppError` type (status/code/message) with a central JSON error handler. Circular page moves detected via recursive CTE (400 error). Sidecar down returns 503 on edit endpoints. |
| N-12 | Testing | Three-layer testing: table-driven unit tests with `httptest`, integration tests with real SQLite, Playwright E2E for critical flows. |
| N-13 | API versioning | No API versioning from the start. |

---

## 6.4 Go Package Structure

Flat domain-driven packages inside `internal/`:
- `internal/pages/` — handlers, services, repositories for page CRUD, tree, permissions
- `internal/auth/` — authentication, JWT, users, groups
- `internal/spaces/` — space CRUD
- `internal/snapshots/` — history, restore, retention
- `internal/attachments/` — file upload and serving
- `internal/mcp/` — MCP transport, tools, resources, prompts
- `internal/search/` — FTS5 queries
- `internal/sse/` — SSE hub, event broadcasting
- `internal/health/` — sidecar health polling, health endpoint

Shared utilities:
- `internal/httputil/` — HTTP helpers, middleware
- `internal/db/` — database connection, migrations
- `internal/middleware/` — auth, permission, rate limiting

---

## 6. System Architecture

### 6.1 Components

```
Browser (React + Tiptap + Yjs)
  ├── WebSocket  →  Nginx :443/collab/*  →  Node.js Hocuspocus :3001
  └── HTTPS      →  Nginx :443/api/*    →  Go API :3000

Go API :3000
  ├── REST API     (/api/*)
  ├── File serve   (/files/*)
  ├── MCP          (/mcp)
  ├── SSE stream   (/api/events)
  └── Internal     (/internal/auth, /internal/save, /internal/restore)

Shared Docker volume /data
  ├── wiki.db              SQLite database
  ├── docs/{space}/{id}.md  Markdown files
  └── attachments/{id}/     Uploaded files
```

### 6.2 Collaboration Architecture

There is no production-ready native Go implementation of the Yjs CRDT. The architecture therefore splits responsibilities:

- **Go API** (~95% of the codebase): all business logic — REST, auth, permissions, storage, MCP.
- **Node.js sidecar** (~150 lines): Hocuspocus only. Accepts WebSocket connections, calls `/internal/auth` to validate tokens and resolve permissions, calls `/internal/save` to deliver Markdown and Yjs snapshots to the Go API.

Communication between sidecar and Go API uses **Unix domain sockets** (not TCP) for lower latency and no network stack overhead.

The internal endpoints form the complete, stable contract between the two processes:

| Endpoint | Caller | Purpose |
|----------|--------|---------|
| `GET /internal/auth?token=…&docId=…` | Sidecar `onAuthenticate` | Validate JWT, resolve permission. Returns 200 + `{userId, permission}` or 403. Fail-fast — no retries. |
| `POST /internal/save` | Sidecar `onStoreDocument` / `onDisconnect` | Deliver `{docId, markdown, yjsSnapshot, authorId}`. Go writes to disk and inserts snapshot row. Retries 3× with exponential backoff; failures logged to `dead_letter_saves` table. |
| `POST /internal/restore` | Go API | Ask sidecar to broadcast a historical snapshot to all live editors. |
| `POST /internal/cleanup` | Go API | Notify sidecar to destroy in-memory Yjs doc for a deleted page. |
| `GET /health` | Go API (polls every 10s) | Sidecar health check. Go caches result; returns 503 on edit endpoints when sidecar is down. |

### 6.3 Technology Stack

| Layer | Technology | Licence |
|-------|-----------|---------|
| Frontend | React + TypeScript + Vite + Tailwind + Tiptap + HocuspocusProvider + @dnd-kit/core + Zustand | MIT |
| Collab sidecar | Node.js + Hocuspocus + @hocuspocus/extension-sqlite + Yjs + y-prosemirror + prosemirror-markdown | MIT |
| Go REST API | Go stdlib net/http + golang-jwt/jwt + mattn/go-sqlite3 (CGO) + sqlc + golang-migrate | BSD-3 / MIT |
| MCP | modelcontextprotocol/go-sdk | MIT |
| Database | SQLite (WAL mode, busy_timeout=5000) via mattn/go-sqlite3 | Public Domain |
| Backup | Litestream | Apache-2 |
| Reverse proxy | Nginx (config volume-mounted from repo) | BSD |
| Deployment | Docker Compose | Apache-2 |

---

## 7. Data Model

### 7.1 Schema

```sql
CREATE TABLE spaces (
  id TEXT PRIMARY KEY, workspace_id TEXT, name TEXT, slug TEXT,
  default_permission TEXT, mcp_write_enabled INTEGER DEFAULT 1,
  snapshot_retention_days INTEGER DEFAULT NULL
);

CREATE TABLE pages (
  id TEXT PRIMARY KEY, parent_id TEXT REFERENCES pages(id),
  space_id TEXT REFERENCES spaces(id), title TEXT, slug TEXT,
  position INTEGER, created_by TEXT, created_at INTEGER, updated_at INTEGER,
  is_directory BOOLEAN DEFAULT 0, icon TEXT
);

CREATE TABLE users (
  id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL,
  name TEXT, password_hash TEXT, global_role TEXT, created_at INTEGER
);

CREATE TABLE groups        (id TEXT PRIMARY KEY, workspace_id TEXT, name TEXT);
CREATE TABLE group_members (group_id TEXT, user_id TEXT, PRIMARY KEY (group_id, user_id));

CREATE TABLE page_permissions (
  id TEXT PRIMARY KEY, page_id TEXT REFERENCES pages(id),
  subject_type TEXT CHECK(subject_type IN ('user','group')),
  subject_id TEXT, permission TEXT, inherit INTEGER DEFAULT 1
);

CREATE TABLE permission_cache (
  page_id TEXT, user_id TEXT, permission TEXT,
  PRIMARY KEY (page_id, user_id)
);

CREATE TABLE page_snapshots (
  id TEXT PRIMARY KEY, page_id TEXT REFERENCES pages(id),
  yjs_snapshot BLOB, markdown TEXT,
  author_id TEXT, created_at INTEGER, label TEXT,
  is_compacted INTEGER DEFAULT 0
);

CREATE TABLE page_content (
  page_id TEXT PRIMARY KEY REFERENCES pages(id),
  title TEXT, content TEXT
);

CREATE TABLE refresh_tokens (
  id TEXT PRIMARY KEY, user_id TEXT,
  token_hash TEXT NOT NULL, expires_at INTEGER,
  created_at INTEGER
);

CREATE TABLE settings (
  key TEXT PRIMARY KEY, value TEXT
);

CREATE TABLE mcp_audit (
  id TEXT PRIMARY KEY, user_id TEXT, tool_name TEXT,
  params_json TEXT, result TEXT, created_at INTEGER
);

CREATE TABLE dead_letter_saves (
  id TEXT PRIMARY KEY, page_id TEXT, markdown TEXT,
  yjs_snapshot BLOB, error TEXT, created_at INTEGER
);

CREATE VIRTUAL TABLE pages_fts USING fts5(
  title, content, content='page_content', content_rowid='rowid'
);
```

### 7.2 File Layout

```
data/
├── wiki.db
├── docs/{space-slug}/{page-id}.md
└── attachments/{page-id}/{uuid}-{filename}
```

### 7.3 Dual-Format Invariant

The Yjs binary in SQLite (managed by Hocuspocus) is the source of truth for the collaboration layer. The `.md` files on disk are a derived export. **Markdown is always derived from Yjs; never the reverse.** The Go API writes the `.md` file only when called by the sidecar's `/internal/save` webhook, never by reading or parsing Markdown.

**Serialization pipeline:** Yjs doc → y-prosemirror → ProseMirror DOM → prosemirror-markdown serializer → Go API writes raw string to disk. The Go API performs no Markdown validation — it trusts the sidecar serialization per the dual-format invariant.

**Bootstrap:** When a new page is created, the Go API writes an empty Yjs snapshot and an empty `.md` file to initialize both stores.

### 7.4 Yjs BLOB Management

Each page maintains two Yjs BLOBs in `page_snapshots`:
- **Working BLOB** — accumulates live edits, grows unbounded
- **Compacted BLOB** — rebuilt from current state when working BLOB exceeds 500KB; old working BLOB is discarded

Sidecar persists Yjs state to its own SQLite via `@hocuspocus/extension-sqlite`. On restart, it reloads the latest working BLOB from `page_snapshots`.

### 7.5 Deleted Page Cleanup

When a page is deleted:
- Yjs document is destroyed in-memory via `POST /internal/cleanup`
- All `page_snapshots` rows for that page are deleted immediately (not kept for retention)
- WebSocket connections to that document are rejected with 404

---

## 8. API Surface

All endpoints are served by the Go API. Authentication is JWT bearer token unless marked `[Public]`. Full documentation (request bodies, response shapes, error codes, curl examples) is in the companion architecture document.

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Register first admin `[Public]` |
| POST | `/api/auth/login` | Obtain tokens `[Public]` |
| POST | `/api/auth/refresh` | Rotate refresh token `[Public]` |
| POST | `/api/auth/logout` | Invalidate refresh token |

### Spaces
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/spaces` | List visible spaces |
| POST | `/api/spaces` | Create space `[admin]` |
| GET | `/api/spaces/{id}` | Get space |
| PATCH | `/api/spaces/{id}` | Update space `[space admin]` |
| DELETE | `/api/spaces/{id}` | Delete space `[admin]` |

### Pages
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/spaces/{spaceId}/pages` | Root pages of a space |
| POST | `/api/pages` | Create page |
| GET | `/api/pages/{id}` | Get page + content |
| PATCH | `/api/pages/{id}` | Update title / position |
| DELETE | `/api/pages/{id}` | Delete page |
| POST | `/api/pages/{id}/move` | Move to new parent |
| GET | `/api/pages/{id}/children` | Lazy-load children |
| GET | `/api/pages/{id}/breadcrumb` | Ancestor chain |
| GET | `/api/pages/search` | Full-text search |

### Permissions
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/pages/{id}/permissions` | List grants |
| PUT | `/api/pages/{id}/permissions` | Set / upsert grant |
| DELETE | `/api/pages/{id}/permissions/{permId}` | Remove grant |

### History
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/pages/{id}/snapshots` | List snapshots |
| GET | `/api/snapshots/{id}` | Get snapshot + binary |
| POST | `/api/pages/{id}/snapshots` | Create manual snapshot |
| POST | `/api/pages/{id}/restore` | Restore snapshot |

### Attachments
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/attachments` | Upload file |
| GET | `/files/{pageId}/{filename}` | Serve file (auth-gated) |

### Users
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | List users `[admin]` |
| POST | `/api/users/invite` | Invite user `[admin]` |
| GET | `/api/users/{id}` | Get profile |
| PATCH | `/api/users/{id}` | Update name / role |
| DELETE | `/api/users/{id}` | Remove user `[admin]` |

### Groups
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/groups` | List groups `[admin]` |
| POST | `/api/groups` | Create group `[admin]` |
| GET | `/api/groups/{id}` | Get group + members `[admin]` |
| PATCH | `/api/groups/{id}` | Rename group `[admin]` |
| DELETE | `/api/groups/{id}` | Delete group `[admin]` |
| POST | `/api/groups/{id}/members` | Add member `[admin]` |
| DELETE | `/api/groups/{id}/members/{userId}` | Remove member `[admin]` |

### System
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/events` | SSE stream (tree updates) |
| POST | `/mcp` | MCP Streamable HTTP endpoint |
| GET | `/.well-known/oauth-protected-resource` | OAuth 2.1 discovery |
| GET | `/health` | Health check |

### Internal (sidecar ↔ Go API, Unix domain socket)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/internal/auth` | Validate token + resolve permission |
| POST | `/internal/save` | Receive Markdown + snapshot |
| POST | `/internal/restore` | Broadcast snapshot to live editors |
| POST | `/internal/cleanup` | Destroy in-memory Yjs doc for deleted page |
| GET | `/health` | Sidecar health check (polled every 10s by Go) |

---

## 9. Permissions Model

### 9.1 Permission Levels

| Level | Value | Can read | Can edit | Can manage permissions | Can delete |
|-------|-------|----------|----------|----------------------|------------|
| `none` | 0 | ✗ | ✗ | ✗ | ✗ |
| `viewer` | 1 | ✓ | ✗ | ✗ | ✗ |
| `editor` | 2 | ✓ | ✓ | ✗ | ✗ |
| `admin` | 3 | ✓ | ✓ | ✓ | ✓ |

### 9.2 Resolution Algorithm

For a given user and page, the effective permission is computed by:

1. Walk up the `parent_id` chain from the page to the space root using a recursive CTE.
2. For each ancestor (including the page itself), collect all `page_permissions` rows where `subject_type = 'user' AND subject_id = <userId>` OR `subject_type = 'group' AND subject_id IN <userGroups>` AND `inherit = 1` (or the row is on the target page itself).
3. Return the maximum (most permissive) permission found. If no grant is found, return the space's `default_permission`.

This is implemented as a single SQL query in Go and called identically from REST middleware, the `/internal/auth` endpoint, and MCP tool handlers. A `permission_cache` table stores resolved results and is invalidated on any permission write to avoid repeated recursive CTE execution.

### 9.3 Write Rules

- Creating a page requires `editor` on the parent.
- Editing content requires `editor` on the page (enforced by `/internal/auth` at WebSocket connect time).
- Moving a page requires `admin` on the source and `editor` on the target parent.
- Deleting a page requires `admin` on the page.
- Managing permissions requires `admin` on the page.

---

## 10. Document History

### 10.1 Snapshot Triggers

| Trigger | When | Label |
|---------|------|-------|
| Debounced save | After 2–5 s quiet period during editing | *(none)* |
| Session end | `onDisconnect` in Hocuspocus | *(none)* |
| Timer | Every 5 minutes while clients active | *(none)* |
| Manual | User clicks "Save version" | User-provided |
| Pre-restore | Automatically before any restore | `"Before restore YYYY-MM-DD HH:MM"` |

### 10.2 Storage

Each snapshot stores:
- `yjs_snapshot` — `Y.encodeSnapshot()` binary (base64 from sidecar, decoded and stored as BLOB)
- `markdown` — plain text of the document at that point (for search and lightweight diff)
- `author_id`, `created_at`, `label`

### 10.3 Restore Flow

1. User selects a snapshot and clicks Restore.
2. UI shows a warning if other editors are currently active on the document.
3. Go API takes a pre-restore snapshot.
4. Go API calls `POST /internal/restore` with the target snapshot binary.
5. Sidecar calls `Y.createDocFromSnapshot()` and broadcasts the result as a live Yjs update.
6. All connected editors receive the update and their editors update instantly.
7. Go API overwrites the `.md` file on disk.
8. Response includes the pre-restore snapshot ID so the action can be undone.

### 10.4 Retention

Each space has a configurable `snapshot_retention_days` (default: unlimited). A nightly Go cron job deletes snapshots older than the retention window, keeping the most recent snapshot per day as a daily summary. Manual snapshots with labels are never auto-deleted.

---

## 11. MCP Integration

### 11.1 Transport

Streamable HTTP at `POST /mcp`. Mounted directly in the Go `ServeMux` via `modelcontextprotocol/go-sdk`. Authentication via JWT bearer token in the `Authorization` header.

### 11.2 Token Scopes

| Scope | Grants |
|-------|--------|
| `wiki:read` | All read operations (resources, search, get page, list history) |
| `wiki:write` | Create, update, move, delete pages; restore snapshots |
| `wiki:admin` | User invite, group management, permission management |
| `wiki:history` | Access to snapshot binaries (required for diff rendering) |

### 11.3 Resources

| URI | Description |
|-----|-------------|
| `wiki://spaces` | List all accessible spaces |
| `wiki://{space}/tree` | Full document tree for a space |
| `wiki://{space}/{slug}` | Page content as `text/markdown` |
| `wiki://{space}/{slug}/meta` | Page metadata (title, dates, author, permission) |
| `wiki://{space}/{slug}/history` | Snapshot list for a page |

### 11.4 Tools

| Tool | Scope | Description |
|------|-------|-------------|
| `wiki_search` | read | Full-text search |
| `wiki_get_page` | read | Get page content and metadata |
| `wiki_create_page` | write | Create a new page |
| `wiki_update_page` | write | Update page content (write appears live to editors) |
| `wiki_move_page` | write | Move to new parent |
| `wiki_delete_page` | write | Delete a page *(destructive)* |
| `wiki_restore_snapshot` | write | Restore a historical version |
| `wiki_set_permission` | admin | Set a permission grant |
| `wiki_invite_user` | admin | Invite a user to the workspace |
| `wiki_create_group` | admin | Create a permission group |
| `wiki_add_group_member` | admin | Add a user to a group |

### 11.5 Prompts

| Prompt | Description |
|--------|-------------|
| `wiki/summarise` | Summarise a page or space |
| `wiki/draft-page` | Draft a new page given a topic and context |
| `wiki/changelog` | Generate a changelog from recent snapshot diffs |
| `wiki/qa` | Answer a question by searching and reading relevant pages |

---

## 12. Iterative Delivery Plan

The project is delivered in six iterations. Each iteration ends with a fully deployable, usable product. No iteration ends in a partial state.

> **Working principle:** Deploy to a real server at the end of each iteration and use the product for at least a few days before starting the next one.

---

### Iteration 1 — Collaborative Editor
**Weeks 1–3**

A single hardcoded page. Real-time editing with cursors. Markdown saved to disk. No auth, no tree.

**Delivers:**
- Open the app and edit a document
- Two browsers edit simultaneously with visible cursors
- Every save writes a plain `.md` file to disk
- `docker-compose up` starts everything

**What to build:**
- Go API skeleton: `ServeMux`, `/internal/auth` (stub — always allows), `/internal/save` (writes `.md` to disk)
- Node.js sidecar: Hocuspocus + SQLite extension, calls `/internal/auth` and `/internal/save`
- Frontend: React + Tiptap + HocuspocusProvider, awareness cursors
- SQLite: `pages` table with one seed row
- Docker Compose: `go-api` + `collab` + `nginx`

---

### Iteration 2 — Multi-Page Wiki
**Weeks 4–7**

Full document tree, sidebar, login/logout, create/move/delete pages.

**Delivers:**
- Register the first admin account
- Login and get a JWT
- Create, rename, move, delete pages
- Collapsible sidebar with drag-and-drop reorder
- Real-time sidebar updates via SSE when another user changes the tree

**What to build:**
- Auth: `users` table, bcrypt, JWT issue/validate, all four auth endpoints
- Spaces: full CRUD, seed one space on register
- Directories: self-referential adjacency list, recursive CTE ancestors, CRUD + move + children + breadcrumb
- Pages: `directory_id` foreign key, CRUD + move + breadcrumb via directory ancestors
- SSE: `GET /api/events`, `sync.Map` hub, broadcasts from page write handlers
- Frontend: sidebar tree, lazy-load, `@dnd-kit` drag-drop, login/logout UI
- `/internal/auth`: now validates the JWT for real

---

### Iteration 3 — Multi-User Access Control
**Weeks 8–11**

Invite colleagues, create groups, set read/write permissions per document tree.

**Delivers:**
- Invite users by email
- Create groups and add users to them
- Set `viewer` / `editor` / `admin` permission on any page for any user or group
- Permissions cascade down the tree, overridable per page
- Users only see pages they have permission to view
- WebSocket auth enforces the same permissions as the REST API

**What to build:**
- Schema: `groups`, `group_members`, `page_permissions` + migrations
- `ResolvePermission()` recursive CTE + `RequirePermission` middleware
- Update `/internal/auth` to call `ResolvePermission`
- Filter all list endpoints to the caller's accessible pages
- All Users and Groups API endpoints
- Permissions API endpoints
- Frontend: invite flow, group management, per-page permission editor

---

### Iteration 4 — History & Attachments
**Weeks 12–14**

The wiki is feature-complete for daily use.

**Delivers:**
- Automatic snapshot on every save and session end
- Browse version history in a sidebar panel
- Visual diff between any two versions (green/red inline)
- Restore any version — live editors see it instantly
- Paste or drop images and files into the editor
- Auth-gated file serving

**What to build:**
- Sidecar: `gc: false`; include `Y.encodeSnapshot()` in `/internal/save`; 5-min interval trigger
- Go `/internal/save`: decode `yjsSnapshot`, `INSERT INTO page_snapshots`
- All Snapshots API endpoints including restore
- Restore flow: pre-restore snapshot → `POST /internal/restore` → sidecar broadcasts → Go overwrites `.md`
- `POST /api/attachments`: multipart, save to `data/attachments/{pageId}/`
- `GET /files/{pageId}/{filename}`: auth-gated streaming
- Frontend: history panel, y-prosemirror diff view, restore confirmation, image paste/drop

---

### Iteration 5 — Search & Production Operations
**Weeks 15–17**

Make the wiki production-ready.

**Delivers:**
- Full-text search with highlighted excerpts
- Continuous SQLite replication via Litestream
- HTTPS with automatic certificate renewal
- Configurable snapshot retention per space
- Periodic Yjs binary compaction
- Health check endpoint
- Rate limiting on auth endpoints

**What to build:**
- SQLite FTS5 virtual table + trigger on snapshot insert
- `GET /api/pages/search` with permission-filtered FTS5 query
- Litestream container in Docker Compose
- Nginx TLS: Certbot container, HTTPS redirect, cron
- Snapshot retention: column on `spaces`, nightly cleanup cron in Go
- Yjs compaction job
- `GET /health`
- Token bucket rate limiter on `/api/auth/*`

---

### Iteration 6 — MCP Integration
**Weeks 18–20**

Open the wiki to AI agents.

**Delivers:**
- Connect any MCP-compatible client (Claude Desktop, VS Code Copilot, custom scripts)
- AI can read, search, create, and update pages
- AI writes appear live to human editors
- AI can manage users, groups, permissions with admin-scoped tokens
- Every AI action logged in `mcp_audit`
- Per-space AI write lock
- Narrow-scope API tokens per agent

**What to build:**
- `internal/mcp/` package, `modelcontextprotocol/go-sdk` dependency
- `mux.Handle("/mcp", AuthMiddleware(mcpTransport))`
- All MCP Resources, Tools, and Prompts
- `mcp_audit` INSERT in every tool handler
- Token scope middleware, `mcp_write_enabled` check, rate limiter
- `GET /.well-known/oauth-protected-resource`
- Frontend: `mcp_write_enabled` toggle per space, scoped token generation panel

---

## 13. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `/internal/save` fails — Markdown not written to disk | Low | High | Sidecar retries 3× with exponential backoff. Failed saves logged to `dead_letter_saves` table for manual recovery. |
| SQLite `SQLITE_BUSY` from concurrent writes | Very low | Medium | WAL mode + `busy_timeout = 5000 ms`. Contention is statistically rare — the two processes write to separate tables, rarely at the same instant. |
| Yjs binary grows unbounded (`gc: false`) | Medium | Low | Periodic compaction job rebuilds the BLOB from current state. Old snapshot BLOBs pruned per retention policy. |
| Restore overwrites a collaborator's unsaved work | Low | Medium | UI warns if other editors are currently active. Pre-restore snapshot taken automatically. |
| Markdown serializer edge cases corrupt files | Medium | High | Round-trip test suite in CI: serialize every Tiptap element type → parse back → assert equality. Tables, nested lists, and fenced code blocks are the known edge cases. |
| Node.js sidecar grows in scope | Low | Medium | Hard rule: the sidecar does exactly three things (WebSocket relay, `/internal/auth`, `/internal/save`). Any new requirement goes in Go, not the sidecar. |

---

## 14. Out of Scope

The following are explicitly excluded from version 1.0 and should not be designed for or implemented:

- **Multi-node deployment** — a single server is the target; horizontal scaling is a future concern
- **Object storage** (S3, MinIO, GCS) — local filesystem only; can be added later by replacing the storage package
- **SSO / LDAP / OAuth login** — email + password only; can be added by extending the auth module
- **Mobile native apps** — responsive web is sufficient
- **Offline editing** — requires a significantly more complex sync architecture
- **Comment threads** — not part of the core wiki use case
- **Page-level access logs** — `mcp_audit` covers AI actions; human read logs are not implemented
- **Webhooks** — outbound event notifications to external systems
- **Import from Confluence / Notion / other wikis** — manual migration only
- **Custom domain per space** — single domain for the whole instance

---

## 15. Resolved Design Decisions

*All decisions from the design grilling session, resolved and accepted.*

### Architecture

| # | Decision | Rationale |
|---|----------|-----------|
| A1 | CGO-based `mattn/go-sqlite3` over pure Go `modernc.org/sqlite` | Pure Go driver is 3-10x slower on writes; CGO acceptable for single-server deployment |
| A2 | Unix domain sockets for sidecar↔Go communication | Lower latency, no network stack overhead, localhost-only security |
| A3 | Go API trusts sidecar Markdown serialization without validation | Dual-format invariant: sidecar owns Yjs→ProseMirror→Markdown pipeline; Go is consumer only |
| A4 | Sidecar handles Yjs→ProseMirror→Markdown serialization | Keeps Go free of Yjs/ProseMirror dependencies; clean separation of concerns |
| A5 | Unlimited concurrent WebSocket connections per document | Single-server target; Yjs handles concurrent connections efficiently |
| A6 | No API versioning from start | v1 only; version when breaking changes needed |

### Database

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | `sqlc` for compile-time type-safe queries | Complex queries (permission CTEs, FTS5, recursive trees) benefit from type safety |
| D2 | `golang-migrate` for migrations, auto-run on startup | Standard tool; no manual migration step needed |
| D3 | `permission_cache` table invalidated on permission writes | Avoids repeated recursive CTE execution on every request |
| D4 | FTS5 on separate `page_content` table (one row per page) | Decouples FTS5 from snapshot lifecycle; simpler updates on save |
| D5 | Integer `position` column with renumbering on move | Avoids REAL precision exhaustion over many reorderings |
| D6 | `settings` key-value table for runtime config | Stores LLM config, retention settings, etc. without schema changes |
| D7 | `refresh_tokens` table with hashed tokens | Secure rotation; can invalidate specific sessions |
| D8 | `dead_letter_saves` table for failed `/internal/save` | Manual recovery path when retries exhausted |

### Auth

| # | Decision | Rationale |
|---|----------|-----------|
| AU1 | Dedicated `/setup` route for first admin + zero-user auto-detect | Clear UX for initial setup; API signals when setup needed |
| AU2 | Bcrypt cost factor 12, configurable via `BCRYPT_COST` | Balance security and performance; configurable for low-power servers |
| AU3 | JWT secret auto-generated to `data/jwt.key` if not in env | Zero-config default; persistent across restarts |
| AU4 | Invite URL returned in API response (no SMTP in v1) | Simpler v1; admin shares manually via Slack/email/etc |
| AU5 | In-memory token bucket rate limiter per IP on auth endpoints | Prevents brute force; no Redis dependency |

### Permissions

| # | Decision | Rationale |
|---|----------|-----------|
| P1 | Both client-side and server-side permission enforcement | Defense in depth; UI hides inaccessible pages, API blocks unauthorized requests |
| P2 | Viewers blocked from paste/upload in frontend | Client-side UX improvement; server-side still validates |

### Collaboration

| # | Decision | Rationale |
|---|----------|-----------|
| C1 | Sidecar persists Yjs to SQLite, reloads on restart | Survives sidecar crashes without losing in-progress edits |
| C2 | `POST /internal/cleanup` endpoint for deleted pages | Immediate in-memory cleanup; prevents ghost documents |
| C3 | Sidecar health polled every 10s by Go | Go can return 503 on edit endpoints when sidecar is down |
| C4 | JWT expiry handled by silent frontend reconnection | No user interruption; refresh token flow transparent |
| C5 | Deleted page snapshots removed immediately | No retention for deleted content; clean storage |
| C6 | Sidecar rejects WebSocket for deleted doc with 404 | Immediate feedback; no dangling connections |

### History

| # | Decision | Rationale |
|---|----------|-----------|
| H1 | Two BLOBs per page: working + compacted (>500KB threshold) | Controls BLOB growth; compaction rebuilds from current state |
| H2 | Warn-and-confirm restore when other editors active | Prevents accidental work loss; pre-restore snapshot enables undo |

### Frontend

| # | Decision | Rationale |
|---|----------|-----------|
| F1 | Zustand store with auth/tree/editor slices | Lightweight, no boilerplate; clear separation of concerns |
| F2 | Yjs doc in React ref, not state | Yjs manages its own reactivity; no need to re-render on every change |
| F3 | SSE with `event_id` + `Last-Event-ID` + ring buffer replay | Reliable reconnection; no missed events after network blips |
| F4 | Page deleted shows toast, no auto-redirect | User decides where to navigate; toast informs of state change |
| F5 | Inline emoji picker on sidebar hover | Minimal UI footprint; discoverable on interaction |
| F6 | Concurrent page title edits: last-write-wins + SSE instant update | Simple conflict resolution; sidebar always current |

### Files

| # | Decision | Rationale |
|---|----------|-----------|
| FI1 | Files served through Go API with JWT + permission check | Security: no direct filesystem access |
| FI2 | `filepath.Clean()` + validate within attachments directory | Prevents path traversal attacks |

### Directories

| # | Decision | Rationale |
|---|----------|-----------|
| DI1 | Split `directories` and `pages` into two tables | Prevents pages nesting inside pages; true filesystem model with directories as containers and pages as leaves |
| DI2 | Click expands/collapses only; content area unchanged | Clear UX distinction from pages |
| DI3 | Directories can set own permissions that override cascade | Same permission model as pages |
| DI4 | Cascade delete all children on directory deletion | Consistent with tree semantics |
| DI5 | Directory↔page conversion via PATCH toggle | Flexibility without delete/recreate |
| DI6 | Single grapheme cluster enforced for icon | Prevents abuse; consistent display |
| DI7 | `wiki_get_page` on directory returns child listing | MCP agents can navigate directory structure |

### MCP

| # | Decision | Rationale |
|---|----------|-----------|
| M1 | Go calls external OpenAI-compatible LLM API for prompts | No embedded LLM; supports OpenAI, LM Studio, llama.cpp, Ollama |
| M2 | LLM config in admin UI, stored in `settings` table | Runtime configuration without restarts |
| M3 | LLM API key encrypted with AES-GCM, key derived from JWT secret via HKDF | No new key file needed; leverages existing secret |

### Iteration 2 — Auth & Multi-Page Wiki

| # | Decision | Rationale |
|---|----------|-----------|
| I2-1 | JWT library: `golang-jwt/jwt/v5` | Standard, well-maintained, covers all needs (HS256, claims, expiry) |
| I2-2 | JWT claims: `sub`, `email`, `role`, `scopes`, `iat`, `exp` | `email` avoids DB hit for display; `scopes` empty for humans, populated for MCP |
| I2-3 | Auth middleware: validates JWT, attaches user to request context | Standard Go pattern; keeps handlers clean; works with `net/http` context |
| I2-4 | First admin: auto-detect zero users on `/api/auth/register` | No separate setup endpoint; register handler checks `COUNT(*) FROM users` |
| I2-5 | Default space auto-created on first registration | Avoids chicken-and-egg problem; seed page migrated into it |
| I2-6 | Page CRUD in single `internal/pages/` package | Tree ops are page ops; avoids cross-package dependencies |
| I2-7 | SSE hub: `sync.Map` of channels + in-memory ring buffer | Simple, performant for single-server; no DB overhead |
| I2-8 | SSE event types: `page_created`, `page_updated`, `page_moved`, `page_deleted` | Covers all tree mutations; more types added in later iterations |
| I2-9 | Slug generation: auto from title, incrementing suffix on collision | Matches wiki conventions; avoids manual slug entry |
| I2-10 | Sidebar tree: flat array with `children` field, lazy-loaded | Matches lazy-load pattern; no rebuild from flat data needed |
| I2-11 | Frontend routing: `react-router-dom` v7 | Routes: `/setup`, `/login`, `/register`, `/`, `/s/:spaceSlug/*` |
| I2-12 | Migrations: fully file-based `golang-migrate` | Clean history from start; `001_initial.up.sql` contains Iteration 1 DDL |
| I2-13 | Seed page: update in place with new `space_id` | Preserves continuity; file path already matches default space slug |
| I2-14 | Auth package: single `internal/auth/` with handler/service/repo/middleware | Flat domain-driven pattern per N-09 |
| I2-15 | Spaces: full CRUD in Iteration 2 | Admin needs to organize content; small amount of code |
| I2-16 | Drag-and-drop: `@dnd-kit/core` + `@dnd-kit/sortable` | Modern, accessible, MIT-licensed, already in PRD tech stack |
| I2-17 | sqlc: adopt now for all Iteration 2 queries | Learn early; codegen pipeline ready before complex permission CTEs |
| I2-18 | `/internal/auth`: query param token + docId, returns userId + permission | Matches PRD Section 6.2 spec; fine for internal Unix socket |
| I2-19 | Token storage: access in localStorage, refresh in httpOnly cookie | Standard secure pattern; XSS risk low for self-hosted wiki |
| I2-20 | Login cookie: `Set-Cookie` on login response | `httpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, 7-day max age |

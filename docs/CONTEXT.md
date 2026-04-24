# Project Context

> Living document capturing the domain model, terminology, conventions, and current state of the Mostdoc codebase. Update this file as decisions change.

## Domain Model

### Core Entities

**Workspace**
The top-level container. In v1 there is exactly one workspace per deployment. All spaces, users, and groups belong to it.

**Space**
A named organizational unit containing a document tree. Each space has:
- `name`, `slug` — display and URL identity
- `default_permission` — `none` | `viewer` | `editor` applied to users without explicit grants
- `mcp_write_enabled` — boolean flag to block AI writes regardless of token scope
- `snapshot_retention_days` — auto-cleanup window for page history (null = unlimited)

**Directory**
A folder in the tree. Self-referential adjacency list (`parent_id` references `directories(id)`).
- `name` — display name
- `slug` — URL-friendly identifier, auto-generated from name with collision resolution
- `position` — integer ordering within parent
- `icon` — single grapheme cluster emoji (optional)
- `space_id` — which space the directory belongs to
- Can nest inside other directories; cannot contain content

**Page**
A document (leaf) that lives inside a directory or at the space root.
- `title` — display name
- `slug` — URL-friendly identifier, auto-generated from title with collision resolution
- `position` — integer ordering within directory
- `icon` — single grapheme cluster emoji (optional)
- `directory_id` — which directory contains the page (null = root)
- `space_id` — which space the page belongs to

**Snapshot**
A point-in-time capture of a page's content. Contains:
- `yjs_snapshot` — binary Yjs document state (`Y.encodeStateAsUpdate()`)
- `markdown` — plain text export at that moment
- `author_id`, `created_at`, `label` — metadata
- `is_compacted` — whether this is a rebuilt compacted BLOB

**User**
A human account. Has `email`, `name`, `password_hash` (bcrypt), `global_role` (`admin` | `user`).

**Group**
A named collection of users for bulk permission grants.

**Permission Grant**
An access rule assigned to a user or group on a specific page. Levels: `none` < `viewer` < `editor` < `admin`. Grants cascade down the tree by default but can be overridden on child pages.

### Key Relationships

```
Workspace
  └── Space (many)
        ├── Directory (self-referential tree)
        │     └── Page (many) — leaves inside directory
        └── Page (many) — root-level leaves
              ├── Snapshot (many, chronological)
              ├── Permission Grant (many)
              └── Attachment (many) — planned

User (many)
  └── Group Membership (many-to-many)
```

## Terminology

| Term | Meaning |
|------|---------|
| **Sidecar** | The Node.js Hocuspocus process that handles WebSocket connections for collaborative editing |
| **Yjs** | The CRDT library that merges concurrent edits without conflicts |
| **Dual-format invariant** | Yjs binary is source of truth; Markdown is derived export only |
| **Working BLOB** | The accumulating Yjs state that grows unbounded during editing |
| **Compacted BLOB** | A rebuilt Yjs state from current document content, replacing the working BLOB when it exceeds 500KB |
| **SSE** | Server-Sent Events; one-way push from server to browser for tree updates |
| **slug** | URL-safe identifier derived from title (e.g., "Getting Started" → `getting-started`) |
| **adjacency list** | Tree representation where each row stores its own `parent_id` (used by directories) |

## Conventions

### Go

- **No third-party HTTP framework**: stdlib `net/http` + `http.ServeMux` only
- **Flat domain packages**: `internal/auth/`, `internal/pages/`, `internal/spaces/` — each has handler, service, repo, models
- **sqlc for queries**: All SQL lives in `.sql` files; generated code in `internal/db/query/`
- **Context propagation**: Every repository method takes `context.Context` first param
- **Structured errors**: `httputil.AppError` with `status`, `code`, `message`; JSON responses always wrap in `{"error": {...}}`
- **JWT middleware pattern**: Global middleware attaches user to context; `RequireAuth` gates individual handlers
- **Path values**: Go 1.22 `r.PathValue("id")` for route parameters

### Frontend

- **Zustand stores**: Separate stores per domain (`auth.ts`, `tree.ts`, `editor.ts`)
- **API normalization**: Backend returns camelCase; frontend normalizes to snake_case in `normalizePage()` / `normalizeSpace()`
- **Token refresh**: Centralized in `api/pages.ts` — any 401 triggers `/api/auth/refresh` with httpOnly cookie, then retries original request
- **SSE reconnection**: `EventSource` with `Last-Event-ID` header for replay

### Database

- **Integer timestamps**: Unix epoch seconds (`INTEGER DEFAULT (strftime('%s', 'now'))`)
- **Soft nulls**: Use `sql.NullString`, `sql.NullInt64`, `sql.NullBool` for nullable columns
- **UUID primary keys**: All entities use `TEXT PRIMARY KEY` with v4 UUIDs
- **Slug uniqueness**: `UNIQUE INDEX idx_pages_space_slug ON pages(space_id, slug)`

### Sidecar

- **Three responsibilities only**: WebSocket relay, `/internal/auth`, `/internal/save`
- **No business logic**: All decisions made in Go; sidecar is a thin adapter
- **Retry policy**: `/internal/save` retries 3x with exponential backoff

## Iteration Status

| Iteration | Scope | State | Notes |
|-----------|-------|-------|-------|
| 1 | Collaborative Editor | ✅ Done | Committed as `0285c8d`. Single page, stub auth, Hocuspocus, docker-compose |
| 2 | Multi-Page Wiki | 🚧 Implemented (uncommitted) | Auth, spaces, page CRUD, tree, sidebar, SSE, slug routing, refresh tokens |
| 3 | Multi-User Access Control | 📋 Planned | Permissions CTE, groups, invites, WebSocket enforcement |
| 4 | History & Attachments | 📋 Planned | Snapshots, diff, restore, file upload, auth-gated serving |
| 5 | Search & Production | 📋 Planned | FTS5, Litestream, HTTPS, rate limiting, health checks |
| 6 | MCP Integration | 📋 Planned | MCP transport, tools, resources, prompts, audit log |

## Decisions Log

### Accepted

| ID | Decision | Rationale | Iteration |
|----|----------|-----------|-----------|
| A1 | CGO `mattn/go-sqlite3` over pure Go | 3-10x faster writes; CGO acceptable for single-server | 1 |
| A2 | Unix domain sockets for sidecar↔Go | Lower latency, localhost-only security | 2 |
| A3 | Go trusts sidecar Markdown without validation | Dual-format invariant; clean separation | 1 |
| A4 | Sidecar owns Yjs→ProseMirror→Markdown | Keeps Go free of JS dependencies | 1 |
| D1 | `sqlc` for type-safe queries | Compile-time safety for complex queries | 2 |
| D2 | `golang-migrate` file-based migrations | Standard tool; auto-run on startup | 2 |
| I2-1 | `golang-jwt/jwt/v5` | Well-maintained; covers HS256, claims, expiry | 2 |
| I2-2 | JWT claims: `sub`, `email`, `name`, `role`, `iat`, `exp` | `email` avoids DB hit for display | 2 |
| I2-3 | Auth middleware validates JWT + attaches to context | Standard Go pattern; keeps handlers clean | 2 |
| I2-4 | First admin via `/api/auth/register` with zero-user check | No separate setup endpoint | 2 |
| I2-5 | Default space auto-created on first registration | Avoids chicken-and-egg | 2 |
| I2-6 | Page CRUD in single `internal/pages/` package | Tree ops are page ops; no cross-package deps | 2 |
| I2-7 | SSE hub: `sync.Map` + ring buffer | Simple, performant for single-server | 2 |
| I2-8 | SSE event types: `page_created`, `page_updated`, `page_moved`, `page_deleted` | Covers all tree mutations | 2 |
| I2-9 | Slug auto-generation with incrementing suffix collision | Matches wiki conventions | 2 |
| I2-10 | Sidebar tree: flat array with `children`, lazy-loaded | No rebuild from flat data needed | 2 |
| I2-11 | `react-router-dom` v7 | Routes: `/setup` omitted; `/register` handles first admin | 2 |
| I2-20 | Split directories and pages into two tables | Prevents pages nesting inside pages; true filesystem model | 2 |
| I2-12 | Migrations fully file-based from start | Clean history; `001_initial` contains Iteration 1 DDL | 2 |
| I2-19 | Token storage: access in localStorage, refresh in httpOnly cookie | Standard secure pattern | 2 |

### Deferred / Pending

| Topic | Current State | Target |
|-------|--------------|--------|
| Permission resolution | Stub: all authenticated users get `admin` | Iteration 3 |
| `/internal/auth` full permission check | Returns hardcoded `admin` | Iteration 3 |
| Groups & group_members tables | Not created | Iteration 3 |
| page_permissions table | Not created | Iteration 3 |
| permission_cache table | Not created | Iteration 3 |
| WebSocket permission enforcement | Stub | Iteration 3 |
| File attachments | No upload/serving endpoints | Iteration 4 |
| History panel | No snapshot endpoints beyond internal save | Iteration 4 |
| Search | No FTS5 table | Iteration 5 |
| Rate limiting | Not implemented | Iteration 5 |
| Litestream backup | Not in docker-compose | Iteration 5 |
| MCP endpoint | Not implemented | Iteration 6 |
| HTTPS / Certbot | HTTP only on :8081 | Iteration 5 |

## Known Issues & Gotchas

1. **Permission stub**: All authenticated users currently have `admin` permission. The `/internal/auth` endpoint and all REST endpoints enforce auth but not granular permissions.
2. **Space slug in markdown path**: `internal/pages/handler.go` Save handler has a TODO for looking up space slug from space ID; currently writes to `docs/default/`.
3. **Circular move check**: Only checks `parentId == id` immediate case. Full descendant check via CTE needed.
4. **No rate limiting**: Auth endpoints are not yet rate-limited.
5. **SSE CORS**: `Access-Control-Allow-Origin: *` is set broadly; should be restricted to frontend origin in production.
6. **Refresh cookie Secure flag**: Set to `false` for local dev; must be `true` behind HTTPS.
7. **Collab DB_PATH env var**: Sidecar no longer uses its own DB but still accepts `DB_PATH` for backwards compatibility.

## How Things Work

### Creating a Page

1. Frontend POST `/api/pages` with `spaceId`, `title`, `parentId`
2. Backend generates slug (collision resolution), assigns position (`max+1`), inserts row
3. Returns page object; frontend optimistically adds to tree store
4. Backend broadcasts `page_created` SSE event to all connected clients
5. Other clients receive SSE and update their tree

### Opening a Document for Editing

1. User navigates to `/s/:spaceSlug/:pageSlug`
2. Frontend resolves page by slug via `GET /api/spaces/{spaceId}/pages/{slug}`
3. `CollaborativeEditor` mounts with `docId = page.id`
4. HocuspocusProvider opens WebSocket to `/collab` with JWT token
5. Sidecar calls `/internal/auth` to validate token (stub permission = admin)
6. Sidecar calls `/internal/load` to fetch latest Yjs snapshot
7. Editor initializes with loaded state; awareness cursors activate

### Saving a Document

1. User edits in Tiptap; Yjs captures changes as CRDT updates
2. On debounce (Hocuspocus default ~5s quiet) or disconnect, sidecar calls `onStoreDocument`
3. Sidecar serializes: Yjs doc → y-prosemirror → ProseMirror DOM → prosemirror-markdown → string
4. Sidecar POSTs `/internal/save` with `{docId, markdown, yjsSnapshot, authorId}`
5. Go writes `.md` file to `data/docs/{space}/{id}.md` and inserts snapshot row

### Refreshing a Session

1. Access token expires after 15 minutes
2. Frontend API client receives 401 on any request
3. Client calls `POST /api/auth/refresh` with httpOnly cookie
4. Backend validates refresh token hash, issues new pair, sets new cookie
5. Client retries original request with new access token
6. If refresh fails, client logs out and redirects to `/login`

## File Inventory (Key Files)

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Wire all packages, start HTTP server |
| `internal/auth/service.go` | JWT issue/parse, bcrypt, refresh rotation |
| `internal/auth/handler.go` | Register, login, refresh, logout, me endpoints |
| `internal/auth/middleware.go` | JWT extraction, context attachment, RequireAuth |
| `internal/pages/rest.go` | Page CRUD, move, breadcrumb REST handlers |
| `internal/pages/handler.go` | Internal endpoints: auth, save, load, cleanup, health |
| `internal/pages/repo.go` | sqlc repository wrapper for page queries |
| `internal/directories/rest.go` | Directory CRUD, move, children, breadcrumb REST handlers |
| `internal/directories/repo.go` | sqlc repository wrapper for directory queries |
| `internal/spaces/handler.go` | Space CRUD REST handlers |
| `internal/sse/hub.go` | SSE subscriber management, ring buffer, broadcast |
| `internal/sse/handler.go` | SSE endpoint with Last-Event-ID replay |
| `internal/db/db.go` | SQLite connection setup (WAL, busy_timeout) |
| `internal/db/migrate.go` | golang-migrate runner |
| `frontend/src/App.tsx` | Router setup |
| `frontend/src/pages/SpacePage.tsx` | Main layout: header, sidebar, editor, breadcrumb |
| `frontend/src/editor/CollaborativeEditor.tsx` | Tiptap + HocuspocusProvider + awareness |
| `frontend/src/components/Sidebar/Sidebar.tsx` | Document tree with lazy-load |
| `frontend/src/api/pages.ts` | Fetch wrapper with auto-refresh on 401 |
| `frontend/src/api/sse.ts` | EventSource connection manager |
| `frontend/src/store/tree.ts` | Zustand tree state |
| `collab/server.js` | Hocuspocus server, internal API client |

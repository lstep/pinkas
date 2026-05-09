# Next Steps: Iteration 4 (History & Attachments)

## Completed Iterations

### Iteration 1 — Collaborative Editor ✅
- Single-page collaborative editor (Tiptap + Hocuspocus)
- JWT auth, SQLite, Docker Compose

### Iteration 2 — Multi-Page Wiki ✅
- Multi-page wiki with directories + pages
- Drag-drop reorder, SSE updates, emoji picker
- Full frontend refactor with Zustand tree store

### Iteration 3 — Multi-User Access Control ✅
- Schema: groups, group_members, permissions tables (migration 006)
- Permission resolver (CTE ancestor walking) in `internal/permissions/`
- Groups CRUD in `internal/groups/`
- Permission enforcement on all pages/directories/spaces handlers
- Frontend: Settings page with Users, Groups, Permissions tabs
- Frontend: `POST /api/users/invite` (admin-only, returns temp password)
- Frontend: Invite form in SettingsPage with temp password modal
- Frontend: ShareModal (Notion-style in-page permission editor)
- Sidebar read-only filtering, CollaborativeEditor read-only mode
- Frontend admin.ts API client for all operations

## What's Next

### Iteration 4 — History & Attachments ✅ Complete
**Phase A (backend): ✅ Complete**
- Migration 007: snapshot index (created_at)
- `internal/pages/history.go`: ListSnapshots, GetSnapshot, RestoreSnapshot handlers
- REST routes: GET/POST /api/pages/{id}/snapshots[/{snapshotId}], POST .../restore
- `collab/server.js`: gc:false, 5-min auto-save interval, POST /internal/restore endpoint
- Sidecar restore: decodes snapshot, applies to Y.Doc, broadcasts to clients, saves markdown
- Pre-restore snapshot taken automatically before restore
- `COLLAB_URL` env var (default http://localhost:3002) wired in main.go

**Phase B (frontend): ✅ Complete**
- HistoryPanel component: slide-out panel listing snapshots with timestamps
- DiffView component: modal showing snapshot content with line-numbered viewer
- Restore button with confirmation (auto pre-restore backup)
- Editor toolbar integration: "History" toggle button in status bar
- `go build ./...` + `npx tsc --noEmit` + `npm run build` all pass clean

**Phase C (attachments): ✅ Complete**
- POST /api/attachments (multipart upload, 10MB limit, editor+ access)
- GET /api/files/{pageId}/{filename} (auth-gated streaming, viewer+ access)
- Path traversal protection (filepath.Base), MIME type validation
- Frontend: Image extension, paste/drop handlers, toolbar upload button
- Images saved to `data/attachments/{pageId}/{uuid}-{filename}`
- `go build ./...` + `npx tsc --noEmit` pass clean

### Iteration 5 — Search & Production Operations

FTS5 search and health check are already built. Remaining work:

- [ ] **Snapshot retention cleanup** — Nightly cron job in Go that deletes snapshots older than `snapshot_retention_days` per space (column exists, logic unwired)
- [ ] **Yjs compaction job** — Periodic task to compact Yjs binary BLOBs when working snapshot exceeds 500KB
- [ ] **Rate limiting on auth endpoints** — In-memory token bucket per IP on `/api/auth/*`
- **Litestream backup** — Skipped (not needed)
- **HTTPS/TLS** — Skipped; you handle NGINX externally with your own cert setup
- **FTS5 search** — ✅ Already implemented (`internal/pages/repo.go`: `SearchPages`, `EnsureFTS5Index`, FTS5 triggers; `internal/pages/search.go`: handler; `GET /api/pages/search` route)
- **Health check** — ✅ Already implemented (`GET /health` in `internal/pages/handler.go`)

### Iteration 6 — MCP Integration
**Weeks 18–20**

- AI agent access via Model Context Protocol
- Scoped tokens, MCP audit logging
- Per-space AI write lock

## File Map

### Key Files for Iteration 5
- `internal/pages/repo.go` — FTS5 search, snapshot queries
- `internal/pages/search.go` — Search HTTP handler
- `docker-compose.yml` — Add litestream container
- `Dockerfile.api` — May need `litestream` binary or separate image
- `internal/pages/handler.go` — Register new routes
- `internal/auth/handler.go` — Add rate limiter middleware

### Reference Files
- `PRD.md` — Iteration 5 scope at lines 708-731
- `docs/ARCHITECTURE.md` — System architecture
- `docs/API.md` — API contract
- `docs/CONTEXT.md` — Decision log

## Known Issues / Gotchas
- `data/wiki.db` and `frontend/dist/` are in `.gitignore` but sometimes show as modified
- Design system refactor done: CSS custom properties + shared UI components (Button, Input, Modal, Card, Badge, Skeleton)
- **HTTPS**: You handle NGINX with your own cert externally — the project's nginx container in docker-compose is for dev/internal routing only

# Next Steps: Iteration 3 (Multi-User Access Control)

## What Was Done So Far

### Iteration 1 (v1)
- Single-page collaborative editor using Tiptap + Hocuspocus
- JWT-based authentication
- SQLite backend with `pages` table
- Hocuspocus sidecar for real-time collaboration
- Docker Compose setup

### Iteration 2 (v2)
- Multi-page wiki with directories and pages as separate tables
- Split `directories` and `pages` — prevents pages nesting inside pages
- Drag-and-drop reorder with persisted positions
- SSE for real-time page updates
- Emoji icon picker (24 emojis + remove) in sidebar context menu
- Editor width fix: removed `max-width: 900px` constraint
- Full frontend refactor with Zustand tree store
- Docker fixes: `sqlc.NullString` → `database/sql.NullString`
- Removed duplicate `GenerateID()` function

### Documentation
- `PRD.md` — Full product requirements, Iteration 3 scope at lines 659-680
- `docs/ARCHITECTURE.md` — System architecture
- `docs/API.md` — API contract
- `docs/SETUP.md` — Setup instructions
- `docs/CONTEXT.md` — Decision log

## Current State

### Done (Waiting for Plan Approval)
- Plan for Iteration 3 drafted with user, including:
  - Schema design: `users`, `groups`, `user_groups`, `permissions` (polymorphic `target_type` + `target_id`)
  - Permission levels: `admin`, `editor`, `viewer`
  - `default_permission` on `spaces` as fallback
  - `permission_cache` table for performance
  - Invite flow: admin calls `POST /api/users/invite` with random temp password, manual share
  - Admin UI: dedicated `/settings` route with tabs for Users, Groups, Permissions

### Blockers
- **Plan not yet approved by user**

## Key Technical Decisions

### DI1 (updated): Directories/Pages Split
Two separate tables instead of single polymorphic tree. Prevents pages nesting inside pages.

### Permission Model
- Polymorphic `permissions` table over separate tables — identical semantics, single resolution query, single cache
- Recursive CTE up directory ancestors for permission resolution
- Group membership expansion
- `permission_cache` table for fast repeated lookups
- Cache invalidation on permission write

### Authentication
- JWT in httpOnly cookies
- `UserContextKey` in context for handlers
- Sidecar `/internal/auth` endpoint must return real permission level for WebSocket auth

### API Flow
- `POST /api/users/invite` → creates user with random password → admin shares manually
- All list endpoints filter by ≥ `viewer` permission
- `default_permission` on spaces serves as fallback

## Known Issues / Gotchas

### Hardcoded Permissions
- `internal/pages/handler.go:Auth` returns hardcoded `"admin"` for all authenticated users
- Must be replaced with real permission resolution

### Files to Not Commit
- `data/wiki.db`
- `frontend/dist/`
- Both are in `.gitignore` but sometimes show as modified

## Next Steps (Post-Approval)

1. **Database**: Implement `006_permissions_groups` migration
2. **Backend - Permissions**: Create `internal/permissions/` package
   - Permission resolution CTE
   - Cache management
   - Middleware for route-level access control
3. **Backend - Groups**: Create `internal/groups/` package
4. **Backend - Auth**: Update `internal/auth/` with invite + user CRUD
5. **Backend - Sidecar**: Update `/internal/auth` endpoint for real permissions
6. **Backend - REST**: Add permission enforcement to all handlers
   - `internal/pages/rest.go`
   - `internal/directories/rest.go`
   - Space handlers
7. **Frontend - Settings**: Build `/settings` page
   - Users tab (invite, list, edit)
   - Groups tab (create, list, edit)
   - Permissions tab (grant/revoke)
8. **Frontend - Permission Awareness**:
   - Sidebar tree filtering by permission
   - Editor read-only mode for viewers
   - Conditional UI based on permission level

## File Map

### Critical Files to Modify
- `migrations/006_permissions_groups.up.sql` — new migration
- `migrations/006_permissions_groups.down.sql` — rollback
- `internal/permissions/` — new package (resolution, cache, middleware)
- `internal/groups/` — new package
- `internal/auth/handler.go` — add invite + user CRUD
- `internal/auth/middleware.go` — permission middleware
- `internal/pages/handler.go` — real `/internal/auth` endpoint
- `internal/pages/rest.go` — enforce permissions
- `internal/directories/rest.go` — enforce permissions
- `frontend/src/settings/` — new routes/pages
- `frontend/src/components/Sidebar/Sidebar.tsx` — permission filtering
- `frontend/src/editor/CollaborativeEditor.tsx` — read-only mode

### Reference Files
- `PRD.md` — requirements reference
- `docs/API.md` — API contract reference
- `docs/CONTEXT.md` — decision log reference
- `migrations/005_split_directories_pages.up.sql` — current schema reference

## Relevant Code Snippets

### Current Permission Stub (needs fixing)
```go
// internal/pages/handler.go:Auth
// Returns hardcoded "admin" for all authenticated users
// Must integrate with real permission resolution
```

### Context Key Pattern
```go
// internal/auth/middleware.go
// UserContextKey used to store UserInfo in request context
// UserInfo { UserID, Email, Name, IsAdmin }
```

### JWT Middleware
```go
// RequireAuth extracts JWT from cookie, verifies, stores UserInfo in context
```

## Session Notes
- Date: 2026-04-25
- Last action: User requested storing status to NEXT.md
- Waiting for: Plan approval for Iteration 3

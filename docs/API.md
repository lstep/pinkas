# API Reference

Base URL: `/api`

All authenticated endpoints require an `Authorization: Bearer <access_token>` header. The access token is returned at login/register and stored in `localStorage`. Refresh tokens are httpOnly cookies handled automatically by the frontend client.

Error responses follow the shape:

```json
{
  "error": {
    "code": "not_found",
    "message": "Page not found"
  }
}
```

---

## Auth

### POST /api/auth/register
Register the first admin account. Only succeeds when zero users exist.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "...",
  "name": "Admin"
}
```

**Response 201:**
```json
{
  "user": {
    "id": "...",
    "email": "admin@example.com",
    "name": "Admin",
    "role": "admin"
  },
  "token": {
    "accessToken": "...",
    "refreshToken": "..."
  }
}
```

**Response 403:** Registration is closed (users already exist).

### POST /api/auth/login
Authenticate and receive tokens.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "..."
}
```

**Response 200:** Same shape as register.

**Response 401:** Invalid email or password.

### POST /api/auth/refresh
Rotate refresh token. Reads `refresh_token` httpOnly cookie.

**Response 200:** Same shape as register (new access token + new refresh cookie).

**Response 401:** Invalid or expired refresh token.

### POST /api/auth/logout
Invalidate refresh token and clear cookie.

**Response 204**

### GET /api/auth/me
Return current authenticated user.

**Response 200:**
```json
{
  "user": {
    "id": "...",
    "email": "...",
    "name": "...",
    "role": "..."
  }
}
```

**Response 401:** Not authenticated.

---

## Spaces

### GET /api/spaces
List all spaces.

**Response 200:**
```json
{
  "spaces": [
    {
      "id": "default",
      "name": "Default Space",
      "slug": "default",
      "defaultPermission": "editor",
      "mcpWriteEnabled": true,
      "snapshotRetentionDays": null
    }
  ]
}
```

### POST /api/spaces
Create a new space.

**Request:**
```json
{
  "name": "Engineering",
  "defaultPermission": "viewer",
  "mcpWriteEnabled": true,
  "snapshotRetentionDays": 30
}
```

**Response 201:** The created space object.

### GET /api/spaces/{id}
Get a single space.

**Response 200:** Space object.

**Response 404:** Space not found.

### PATCH /api/spaces/{id}
Update a space.

**Request:**
```json
{
  "name": "New Name",
  "defaultPermission": "editor",
  "mcpWriteEnabled": false,
  "snapshotRetentionDays": 90
}
```

**Response 200:** Updated space object.

### DELETE /api/spaces/{id}
Delete a space.

**Response 204**

---

## Directories

### GET /api/spaces/{spaceId}/directories
List root directories for a space.

**Response 200:**
```json
{
  "directories": [
    {
      "id": "...",
      "spaceId": "default",
      "parentId": null,
      "name": "Getting Started",
      "slug": "getting-started",
      "position": 0,
      "icon": null,
      "createdBy": "...",
      "createdAt": 1713800000,
      "updatedAt": 1713800000
    }
  ]
}
```

### POST /api/directories
Create a new directory.

**Request:**
```json
{
  "spaceId": "default",
  "parentId": "parent-directory-id-or-null",
  "name": "My Folder",
  "icon": "📁"
}
```

**Response 201:** The created directory object. Slug is auto-generated from name with collision resolution.

### GET /api/directories/{id}
Get a single directory by ID.

**Response 200:** Directory object.

**Response 404:** Directory not found.

### GET /api/spaces/{spaceId}/directories/{slug}
Get a directory by space + slug.

**Response 200:** Directory object.

**Response 404:** Directory not found.

### PATCH /api/directories/{id}
Update a directory. Partial updates supported.

**Request:**
```json
{
  "name": "New Name",
  "icon": "📁",
  "position": 2
}
```

**Response 200:** Updated directory object. Name changes trigger slug regeneration with collision resolution.

### DELETE /api/directories/{id}
Delete a directory.

**Response 204**

### POST /api/directories/{id}/move
Move a directory to a new parent.

**Request:**
```json
{
  "parentId": "new-parent-directory-id-or-null"
}
```

**Response 204**

**Response 400:** Circular move (moving a directory into itself or its descendants).

### GET /api/directories/{id}/children
List direct child directories of a directory.

**Response 200:**
```json
{
  "directories": [ ... ]
}
```

### GET /api/directories/{id}/pages
List pages inside a directory.

**Response 200:**
```json
{
  "pages": [ ... ]
}
```

### GET /api/directories/{id}/breadcrumb
Return ancestor chain from root to this directory (not including the directory itself).

**Response 200:**
```json
{
  "ancestors": [ ... ]
}
```

---

## Pages

### GET /api/spaces/{spaceId}/pages
List root pages for a space (pages with no directory).

**Response 200:**
```json
{
  "pages": [
    {
      "id": "...",
      "spaceId": "default",
      "directoryId": null,
      "title": "Getting Started",
      "slug": "getting-started",
      "position": 0,
      "icon": null,
      "createdBy": "...",
      "createdAt": 1713800000,
      "updatedAt": 1713800000
    }
  ]
}
```

### POST /api/pages
Create a new page.

**Request:**
```json
{
  "spaceId": "default",
  "directoryId": "directory-id-or-null",
  "title": "My Page",
  "icon": ""
}
```

**Response 201:** The created page object. Slug is auto-generated from title with collision resolution (`-1`, `-2`, etc.).

### GET /api/pages/{id}
Get a single page by ID.

**Response 200:** Page object.

**Response 404:** Page not found.

### GET /api/spaces/{spaceId}/pages/{slug}
Get a page by space + slug. Used by frontend routing.

**Response 200:** Page object.

**Response 404:** Page not found.

### PATCH /api/pages/{id}
Update a page. Partial updates supported.

**Request:**
```json
{
  "title": "New Title",
  "icon": "📄",
  "position": 2
}
```

**Response 200:** Updated page object. Title changes trigger slug regeneration with collision resolution.

### DELETE /api/pages/{id}
Delete a page.

**Response 204**

### POST /api/pages/{id}/move
Move a page to a different directory.

**Request:**
```json
{
  "directoryId": "new-directory-id-or-null"
}
```

**Response 204**

---

## SSE

### GET /api/events
Server-Sent Events stream for real-time tree updates.

**Headers:**
- `Authorization: Bearer <token>` (required)
- `Last-Event-ID: <id>` (optional, for replay on reconnect)

**Event format:**
```
id: 42
event: page_created
data: {"id":"...","title":"...",...}

```

**Event types:**
- `page_created` — new page created
- `page_updated` — page title/slug/icon/position changed
- `page_moved` — page moved to new directory (`{id, directoryId}`)
- `page_deleted` — page deleted (`{id}`)
- `directory_created` — new directory created
- `directory_updated` — directory name/slug/icon/position changed
- `directory_moved` — directory moved to new parent (`{id, parentId}`)
- `directory_deleted` — directory deleted (`{id}`)

The connection sends `:keepalive` comments every 30 seconds and `retry: 5000` on connect.

---

## Internal Endpoints (Sidecar ↔ Go)

These are called by the Node.js Hocuspocus sidecar, not by the frontend.

### GET /internal/auth?token={jwt}&docId={docId}
Validate JWT and resolve permission for a document.

**Response 200:**
```json
{
  "userId": "...",
  "permission": "admin"
}
```

**Response 403:** Invalid token or insufficient permission.

### POST /internal/save
Receive Markdown + Yjs snapshot from sidecar on save/disconnect.

**Request:**
```json
{
  "docId": "...",
  "markdown": "# Hello\n\nWorld",
  "yjsSnapshot": "base64...",
  "authorId": "..."
}
```

**Response 200**

Sidecar retries 3x with exponential backoff on failure.

### GET /internal/load?docId={docId}
Return the latest Yjs snapshot for a document.

**Response 200:**
```json
{
  "yjsSnapshot": "base64..."
}
```

**Response 200 (empty):**
```json
{
  "yjsSnapshot": null
}
```

### POST /internal/restore
Broadcast a historical snapshot to all live editors. (Planned — Iteration 4)

### POST /internal/cleanup
Destroy in-memory Yjs doc for a deleted page. (Stub — full impl planned)

### GET /health
Health check. Returns 200 if healthy.

---

## Planned Endpoints (Iterations 3–6)

### Permissions
- `GET /api/pages/{id}/permissions` — list grants
- `PUT /api/pages/{id}/permissions` — set/upsert grant
- `DELETE /api/pages/{id}/permissions/{permId}` — remove grant

### History
- `GET /api/pages/{id}/snapshots` — list snapshots
- `GET /api/snapshots/{id}` — get snapshot + binary
- `POST /api/pages/{id}/snapshots` — create manual snapshot
- `POST /api/pages/{id}/restore` — restore snapshot

### Attachments
- `POST /api/attachments` — upload file
- `GET /files/{pageId}/{filename}` — serve file (auth-gated)

### Users
- `GET /api/users` — list users
- `POST /api/users/invite` — invite user
- `GET /api/users/{id}` — get profile
- `PATCH /api/users/{id}` — update name/role
- `DELETE /api/users/{id}` — remove user

### Groups
- `GET /api/groups` — list groups
- `POST /api/groups` — create group
- `GET /api/groups/{id}` — get group + members
- `PATCH /api/groups/{id}` — rename
- `DELETE /api/groups/{id}` — delete
- `POST /api/groups/{id}/members` — add member
- `DELETE /api/groups/{id}/members/{userId}` — remove member

### Search
- `GET /api/pages/search?q={query}&spaceId={spaceId}` — full-text search

### MCP
- `POST /mcp` — MCP Streamable HTTP endpoint

### System
- `GET /.well-known/oauth-protected-resource` — OAuth 2.1 discovery

---

## Auth Middleware Behavior

The global auth middleware (`auth.Middleware`) runs on every request:
1. Extracts `Bearer` token from `Authorization` header
2. Parses and validates JWT
3. Attaches `UserInfo` to request context
4. Continues regardless of auth status (endpoints use `RequireAuth` for gating)

`RequireAuth` checks context for user and returns 401 if missing.

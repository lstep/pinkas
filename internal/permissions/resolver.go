package permissions

import (
	"context"
	"log/slog"

	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Permission levels.
const (
	LevelNone   = 0
	LevelViewer = 1
	LevelEditor = 2
	LevelAdmin  = 3
)

// levelFromString converts a space default_permission string to a numeric level.
func levelFromString(s string) int {
	switch s {
	case "viewer":
		return LevelViewer
	case "editor":
		return LevelEditor
	case "admin":
		return LevelAdmin
	default:
		return LevelNone
	}
}

// Resolver resolves permission levels for resources.
type Resolver struct {
	permRepo  *Repository
	dirGet    func(ctx context.Context, id string) (sqlc.Directory, error)
	spaceGet  func(ctx context.Context, id string) (sqlc.Space, error)
	pageGet   func(ctx context.Context, id string) (sqlc.Page, error)
	listUG    func(ctx context.Context, userID string) ([]sqlc.Group, error)
	logger    *slog.Logger
}

// NewResolver creates a new permission resolver.
func NewResolver(permRepo *Repository, dirGet func(ctx context.Context, id string) (sqlc.Directory, error), spaceGet func(ctx context.Context, id string) (sqlc.Space, error), pageGet func(ctx context.Context, id string) (sqlc.Page, error), listUG func(ctx context.Context, userID string) ([]sqlc.Group, error), logger *slog.Logger) *Resolver {
	return &Resolver{
		permRepo: permRepo,
		dirGet:   dirGet,
		spaceGet: spaceGet,
		pageGet:  pageGet,
		listUG:   listUG,
		logger:   logger,
	}
}

// ResolveSpace returns the effective permission level for a user on a space.
// Checks: global admin -> direct user perm -> group perms -> space default.
func (r *Resolver) ResolveSpace(ctx context.Context, userID, spaceID string) int {
	// Quick check: get space default
	space, err := r.spaceGet(ctx, spaceID)
	if err != nil {
		r.logger.Warn("resolve space: failed to fetch space", "spaceID", spaceID, "error", err)
		return LevelNone
	}

	maxLevel := r.resolveTarget(ctx, userID, "space", spaceID)
	if maxLevel >= LevelAdmin {
		return maxLevel
	}

	// Fall back to space default
	defaultLevel := LevelNone
	if space.DefaultPermission.Valid {
		defaultLevel = levelFromString(space.DefaultPermission.String)
	}
	if defaultLevel > maxLevel {
		maxLevel = defaultLevel
	}

	return maxLevel
}

// ResolveDirectory returns the effective permission level for a user on a directory.
// Walks: directory -> parent directory chain -> space -> default.
func (r *Resolver) ResolveDirectory(ctx context.Context, userID, directoryID string) int {
	dir, err := r.dirGet(ctx, directoryID)
	if err != nil {
		r.logger.Warn("resolve directory: failed to fetch", "directoryID", directoryID, "error", err)
		return LevelNone
	}

	maxLevel := r.resolveTarget(ctx, userID, "directory", directoryID)
	if maxLevel >= LevelAdmin {
		return maxLevel
	}

	// Walk parent chain
	spaceID := ""
	if dir.SpaceID.Valid {
		spaceID = dir.SpaceID.String
	}
	currentID := directoryID
	visited := make(map[string]bool)

	for {
		if visited[currentID] {
			break
		}
		visited[currentID] = true

		currentDir, err := r.dirGet(ctx, currentID)
		if err != nil {
			break
		}

		if !currentDir.ParentID.Valid || currentDir.ParentID.String == "" {
			break
		}

		parentID := currentDir.ParentID.String
		parentLevel := r.resolveTarget(ctx, userID, "directory", parentID)
		if parentLevel > maxLevel {
			maxLevel = parentLevel
		}
		if maxLevel >= LevelAdmin {
			return maxLevel
		}
		currentID = parentID
	}

	// Check space level
	if spaceID != "" {
		spaceLevel := r.ResolveSpace(ctx, userID, spaceID)
		if spaceLevel > maxLevel {
			maxLevel = spaceLevel
		}
	}

	return maxLevel
}

// ResolvePage returns the effective permission level for a user on a page.
// Walks: page -> directory chain -> space -> default.
func (r *Resolver) ResolvePage(ctx context.Context, userID, pageID string) int {
	maxLevel := r.resolveTarget(ctx, userID, "page", pageID)
	if maxLevel >= LevelAdmin {
		return maxLevel
	}

	page, err := r.pageGet(ctx, pageID)
	if err != nil {
		r.logger.Warn("resolve page: failed to fetch page", "pageID", pageID, "error", err)
		return maxLevel
	}

	// If page has a directory, resolve via directory chain
	if page.DirectoryID.Valid && page.DirectoryID.String != "" {
		dirLevel := r.ResolveDirectory(ctx, userID, page.DirectoryID.String)
		if dirLevel > maxLevel {
			maxLevel = dirLevel
		}
		return maxLevel
	}

	// No directory, resolve via space
	if page.SpaceID.Valid && page.SpaceID.String != "" {
		spaceLevel := r.ResolveSpace(ctx, userID, page.SpaceID.String)
		if spaceLevel > maxLevel {
			maxLevel = spaceLevel
		}
	}

	return maxLevel
}

// Resolve dispatches permission resolution by target type.
func (r *Resolver) Resolve(ctx context.Context, userID, targetType, targetID string) int {
	switch targetType {
	case "space":
		return r.ResolveSpace(ctx, userID, targetID)
	case "directory":
		return r.ResolveDirectory(ctx, userID, targetID)
	case "page":
		return r.ResolvePage(ctx, userID, targetID)
	default:
		return r.resolveTarget(ctx, userID, targetType, targetID)
	}
}

// HasAccess returns true if the user's level meets minLevel. Global admins bypass all checks.
func HasAccess(role string, level, minLevel int) bool {
	if role == "admin" {
		return true
	}
	return level >= minLevel
}

// resolveTarget checks direct user and group permissions for a specific target.
func (r *Resolver) resolveTarget(ctx context.Context, userID, targetType, targetID string) int {
	maxLevel := LevelNone

	// Check direct user permission
	perm, err := r.permRepo.GetPermission(ctx, targetType, targetID, "user", userID)
	if err == nil {
		if int(perm.Level) > maxLevel {
			maxLevel = int(perm.Level)
		}
	}

	// Check group permissions
	groups, err := r.listUG(ctx, userID)
	if err == nil {
		for _, g := range groups {
			gPerm, err := r.permRepo.GetPermission(ctx, targetType, targetID, "group", g.ID)
			if err == nil {
				if int(gPerm.Level) > maxLevel {
					maxLevel = int(gPerm.Level)
				}
			}
		}
	}

	return maxLevel
}

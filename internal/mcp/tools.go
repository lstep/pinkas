package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
	"github.com/mostdoc/mostdoc/internal/mcptokens"
	"github.com/mostdoc/mostdoc/internal/pages"
	"github.com/mostdoc/mostdoc/internal/permissions"
)

func (s *Server) registerTools() {
	s.mcpServer.AddTool(s.defineWikiSearch(), s.handleWikiSearch)
	s.mcpServer.AddTool(s.defineWikiGetPage(), s.handleWikiGetPage)
	s.mcpServer.AddTool(s.defineWikiCreatePage(), s.handleWikiCreatePage)
	s.mcpServer.AddTool(s.defineWikiUpdatePage(), s.handleWikiUpdatePage)
	s.mcpServer.AddTool(s.defineWikiListSpaces(), s.handleWikiListSpaces)
	s.mcpServer.AddTool(s.defineWikiGetSpaceBySlug(), s.handleWikiGetSpaceBySlug)
}

// ─── Tool definitions ───────────────────────────────────────────────────────

func (s *Server) defineWikiSearch() mcp.Tool {
	return mcp.NewTool("wiki_search",
		mcp.WithDescription("Search wiki pages by query string"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query text"),
			mcp.MinLength(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default 10)"),
			mcp.DefaultNumber(10),
			mcp.Min(1),
			mcp.Max(50),
		),
	)
}

func (s *Server) defineWikiGetPage() mcp.Tool {
	return mcp.NewTool("wiki_get_page",
		mcp.WithDescription("Get a wiki page by ID or slug"),
		mcp.WithString("identifier",
			mcp.Required(),
			mcp.Description("Page ID (UUID) or 'space-slug/page-slug'"),
		),
	)
}

func (s *Server) defineWikiCreatePage() mcp.Tool {
	return mcp.NewTool("wiki_create_page",
		mcp.WithDescription("Create a new wiki page"),
		mcp.WithString("space_id",
			mcp.Required(),
			mcp.Description("Space ID (UUID) to create the page in"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Page title"),
			mcp.MinLength(1),
		),
		mcp.WithString("content",
			mcp.Description("Page content (markdown)"),
		),
		mcp.WithString("icon",
			mcp.Description("Optional icon emoji for the page"),
		),
	)
}

func (s *Server) defineWikiUpdatePage() mcp.Tool {
	return mcp.NewTool("wiki_update_page",
		mcp.WithDescription("Update an existing wiki page"),
		mcp.WithString("page_id",
			mcp.Required(),
			mcp.Description("Page ID (UUID) to update"),
		),
		mcp.WithString("title",
			mcp.Description("New title"),
		),
		mcp.WithString("content",
			mcp.Description("New content (markdown)"),
		),
	)
}

func (s *Server) defineWikiListSpaces() mcp.Tool {
	return mcp.NewTool("wiki_list_spaces",
		mcp.WithDescription("List all available wiki spaces"),
	)
}

func (s *Server) defineWikiGetSpaceBySlug() mcp.Tool {
	return mcp.NewTool("wiki_get_space_by_slug",
		mcp.WithDescription("Get space details by slug"),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("Space slug (e.g. 'my-space')"),
		),
	)
}

// ─── Tool handlers ──────────────────────────────────────────────────────────

func (s *Server) handleWikiSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, _ := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required. Provide a valid MCP token via Authorization header or ?token= query parameter."), nil
	}

	query := req.GetString("query", "")
	limit := req.GetInt("limit", 10)

	results, err := s.pagesRepo.SearchPages(ctx, query, limit)
	if err != nil {
		s.logger.Error("wiki_search failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	filtered := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		if s.canAccessSpace(token, r.Page.SpaceID.String, permissions.LevelViewer) {
			item := map[string]interface{}{
				"id":       r.Page.ID,
				"title":    r.Page.Title.String,
				"slug":     r.Page.Slug.String,
				"space_id": r.Page.SpaceID.String,
				"excerpt":  truncate(r.Markdown, 200),
			}
			filtered = append(filtered, item)
		}
	}

	result := map[string]interface{}{
		"query":   query,
		"total":   len(filtered),
		"results": filtered,
	}

	return jsonResult(result)
}

func (s *Server) handleWikiGetPage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, _ := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required."), nil
	}

	identifier := req.GetString("identifier", "")

	var p sqlc.Page
	var err error

	// Try as page ID first
	p, err = s.pagesRepo.GetPage(ctx, identifier)
	if err != nil {
		// Try as space-slug/page-slug
		parts := splitIdentifier(identifier)
		if len(parts) != 2 {
			return mcp.NewToolResultError("Page not found. Use a page ID (UUID) or format: space-slug/page-slug"), nil
		}
		space, spaceErr := s.spacesRepo.GetBySlug(ctx, parts[0])
		if spaceErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Space not found: %s", parts[0])), nil
		}
		if !s.canAccessSpace(token, space.ID, permissions.LevelViewer) {
			return mcp.NewToolResultError("Access denied"), nil
		}
		p, err = s.pagesRepo.GetPageBySlug(ctx, space.ID, parts[1])
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Page not found: %s", identifier)), nil
		}
	} else {
		if !s.canAccessSpace(token, p.SpaceID.String, permissions.LevelViewer) {
			return mcp.NewToolResultError("Access denied"), nil
		}
	}

	// Get latest snapshot for content
	var markdown string
	snapshot, snapErr := s.pagesRepo.GetLatestSnapshot(ctx, p.ID)
	if snapErr == nil && snapshot != nil {
		markdown = snapshot.Markdown
	}

	result := map[string]interface{}{
		"id":       p.ID,
		"space_id": p.SpaceID.String,
		"title":    p.Title.String,
		"slug":     p.Slug.String,
		"content":  markdown,
		"icon":     p.Icon.String,
		"position": p.Position.Int64,
	}

	return jsonResult(result)
}

func (s *Server) handleWikiCreatePage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, isAdmin := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required."), nil
	}
	if err := requireWrite(token); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	spaceID, _ := req.RequireString("space_id")
	title, _ := req.RequireString("title")
	content := req.GetString("content", "")
	icon := req.GetString("icon", "")

	// Check permission on the space
	if !isAdmin && s.permRes.ResolveSpace(ctx, userID, spaceID) < permissions.LevelEditor {
		return mcp.NewToolResultError("Access denied: editor permission required in this space"), nil
	}

	// Acquire write lock for this space
	if err := s.writeLock.Lock(spaceID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.writeLock.Unlock(spaceID)

	// Generate slug
	slug := slugify(title)
	existing, slugErr := s.pagesRepo.GetPagesBySlugPrefix(ctx, spaceID, slug)
	if slugErr == nil && len(existing) > 0 {
		slug = resolveSlugCollision(slug, existing)
	}

	maxPos, _ := s.pagesRepo.GetMaxPosition(ctx, nil)
	position := maxPos + 1

	id := pages.GenerateID()
	if err := s.pagesRepo.CreatePage(ctx, id, spaceID, title, slug, position, nil, userID, icon); err != nil {
		s.logger.Error("mcp create page failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create page: %v", err)), nil
	}

	// Save initial snapshot
	if content != "" {
		if err := s.pagesRepo.SaveSnapshot(ctx, id, content, nil, userID); err != nil {
			s.logger.Warn("mcp create page: failed to save snapshot", "error", err)
		}
	}

	result := map[string]interface{}{
		"id":       id,
		"title":    title,
		"slug":     slug,
		"space_id": spaceID,
	}

	return jsonResult(result)
}

func (s *Server) handleWikiUpdatePage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, isAdmin := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required."), nil
	}
	if err := requireWrite(token); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pageID, _ := req.RequireString("page_id")
	newTitle := req.GetString("title", "")
	newContent := req.GetString("content", "")

	if newTitle == "" && newContent == "" {
		return mcp.NewToolResultError("Provide at least one of: title or content"), nil
	}

	// Verify page exists and check access
	existingPage, err := s.pagesRepo.GetPage(ctx, pageID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Page not found: %s", pageID)), nil
	}
	if !isAdmin && s.permRes.ResolvePage(ctx, userID, pageID) < permissions.LevelEditor {
		return mcp.NewToolResultError("Access denied: editor permission required"), nil
	}

	// Acquire write lock for the page's space
	if err := s.writeLock.Lock(existingPage.SpaceID.String); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.writeLock.Unlock(existingPage.SpaceID.String)

	if newTitle != "" {
		slug := slugify(newTitle)
		existing, slugErr := s.pagesRepo.GetPagesBySlugPrefix(ctx, existingPage.SpaceID.String, slug)
		if slugErr == nil && len(existing) > 0 {
			slug = resolveSlugCollision(slug, existing)
		}
		if err := s.pagesRepo.UpdatePage(ctx, pageID, newTitle, slug); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page: %v", err)), nil
		}
	}

	if newContent != "" {
		if err := s.pagesRepo.SaveSnapshot(ctx, pageID, newContent, nil, userID); err != nil {
			s.logger.Warn("mcp update page: failed to save snapshot", "error", err)
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf("Page updated: %s", pageID)), nil
}

func (s *Server) handleWikiListSpaces(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, _ := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required."), nil
	}

	allSpaces, err := s.spacesRepo.List(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list spaces: %v", err)), nil
	}

	filtered := make([]map[string]interface{}, 0, len(allSpaces))
	for _, sp := range allSpaces {
		if s.canAccessSpace(token, sp.ID, permissions.LevelViewer) {
			item := map[string]interface{}{
				"id":   sp.ID,
				"name": sp.Name,
				"slug": sp.Slug,
			}
			filtered = append(filtered, item)
		}
	}

	return jsonResult(map[string]interface{}{
		"total":  len(filtered),
		"spaces": filtered,
	})
}

func (s *Server) handleWikiGetSpaceBySlug(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, token, _ := mcpUserFromContext(ctx)
	if userID == "" {
		return mcp.NewToolResultError("Authentication required."), nil
	}

	slug := req.GetString("slug", "")
	space, err := s.spacesRepo.GetBySlug(ctx, slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Space not found: %s", slug)), nil
	}

	if !s.canAccessSpace(token, space.ID, permissions.LevelViewer) {
		return mcp.NewToolResultError("Access denied"), nil
	}

	result := map[string]interface{}{
		"id":                 space.ID,
		"name":               space.Name,
		"slug":               space.Slug,
		"default_permission": space.DefaultPermission,
	}

	return jsonResult(result)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (s *Server) canAccessSpace(token *mcptokens.MCPToken, spaceID string, minLevel int) bool {
	if token == nil {
		return false
	}

	// If token has a space_id restriction, check it
	if !mcptokens.HasAccessToSpace(token, spaceID) {
		return false
	}

	// Admin-scoped tokens can access any space they are restricted to
	if hasScope(token, mcptokens.ScopeAdmin) {
		return true
	}

	return true // scope-level permission is sufficient for read within allowed space
}

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultJSON(v)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func splitIdentifier(identifier string) []string {
	parts := make([]string, 0, 2)
	current := ""
	for _, ch := range identifier {
		if ch == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// slugify produces a URL-safe slug from a title.
func slugify(s string) string {
	var result strings.Builder
	lastWasDash := false
	for _, ch := range strings.ToLower(strings.TrimSpace(s)) {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			result.WriteRune(ch)
			lastWasDash = false
		} else if ch == '-' || ch == '_' {
			if !lastWasDash {
				result.WriteRune('-')
				lastWasDash = true
			}
		} else {
			if !lastWasDash {
				result.WriteRune('-')
				lastWasDash = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}

// resolveSlugCollision finds a unique slug by appending a counter.
func resolveSlugCollision(baseSlug string, existing []sqlc.Page) string {
	taken := make(map[string]bool)
	for _, p := range existing {
		taken[p.Slug.String] = true
	}
	if !taken[baseSlug] {
		return baseSlug
	}
	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", baseSlug, i)
		if !taken[candidate] {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", baseSlug, time.Now().Unix())
}

package pages

import (
	"net/http"
	"strconv"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
	"github.com/mostdoc/mostdoc/internal/permissions"
)

// Search handles GET /api/pages/search
func (h *RESTHandler) Search(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	// Handle empty query
	if query == "" {
		httputil.JSON(w, http.StatusOK, map[string]interface{}{
			"results": []interface{}{},
		})
		return
	}

	// Search pages
	results, err := h.repo.SearchPages(r.Context(), query, limit)
	if err != nil {
		h.logger.Error("search pages failed", "error", err, "query", query)
		httputil.WriteError(w, http.StatusInternalServerError, "search_failed", "Search failed")
		return
	}

	// Filter by permissions for non-admin users
	var filteredResults []map[string]interface{}
	for _, result := range results {
		// Check permission
		level := h.permResolver.ResolvePage(r.Context(), user.ID, result.Page.ID)
		if !permissions.HasAccess(user.Role, level, permissions.LevelViewer) {
			continue
		}

		filteredResults = append(filteredResults, map[string]interface{}{
			"id":        result.Page.ID,
			"spaceId":   result.Page.SpaceID.String,
			"title":     result.Page.Title.String,
			"slug":      result.Page.Slug.String,
			"markdown":  result.Markdown,
			"directoryId": result.Page.DirectoryID.String,
			"icon":      result.Page.Icon.String,
		})
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{
		"results": filteredResults,
	})
}

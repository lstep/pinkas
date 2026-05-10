package permissions

import (
	"net/http"

	"github.com/pinkas/pinkas/internal/auth"
	"github.com/pinkas/pinkas/internal/httputil"
)

// TargetResolver extracts target type and ID from a request.
type TargetResolver func(r *http.Request) (targetType string, targetID string)

// SpaceTarget returns a TargetResolver for a space ID from path value.
func SpaceTarget(pathValue string) TargetResolver {
	return func(r *http.Request) (string, string) {
		return "space", r.PathValue(pathValue)
	}
}

// Require returns middleware that enforces a minimum permission level for a target.
// During Iteration 3, direct admin global_role bypasses permission checks.
// Non-admin users get their permission resolved through the resolver.
// Returns the middleware handler.
func Require(resolver *Resolver, minLevel int, targetResolver TargetResolver) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
				return
			}

			// Admin global role bypasses all permission checks
			if user.Role == "admin" {
				next(w, r)
				return
			}

			targetType, targetID := targetResolver(r)
			if targetID == "" {
				httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Target ID is required")
				return
			}

			level := resolver.Resolve(r.Context(), user.ID, targetType, targetID)
			if level < minLevel {
				httputil.WriteError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
				return
			}

			next(w, r)
		}
	}
}

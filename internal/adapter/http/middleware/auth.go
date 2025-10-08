package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

// --- base auth middleware ---

// Auth validates JWT, loads user and injects it into context.
// If token is invalid/missing, returns 401.
func (h *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		header := r.Header.Get("Authorization")
		// if no header, treat as anonymous user
		// anonymous user can access only public endpoints
		// protected endpoints should return 401
		if header == "" {
			r = r.WithContext(models.WithUser(ctx, models.AnonymousUser()))
			next.ServeHTTP(w, r)
			return
		}

		token, err := extractBearerToken(header)
		if err != nil {
			errorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Validate token & fetch user via domain service
		user, err := h.auth.RoleCheck(ctx, token)
		if err != nil || user == nil {
			h.log.Error(wrap.ErrorCtx(ctx, err), "failed to authenticate user", err)
			errorResponse(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		next.ServeHTTP(w, r.WithContext(models.WithUser(ctx, user)))
	})
}

// RequireRoles wraps a handler and allows only users with one of the given roles.
// Usage: mux.Handle("/admin", h.RequireRoles(h.AuthMiddleware(adminHandler), types.RoleAdmin.String()))
func (h *Middleware) RequireRoles(next http.HandlerFunc, allowedRoles ...types.UserRole) http.Handler {
	allowed := make(map[types.UserRole]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := models.UserFromContext(r.Context())
		if user == nil || user.IsAnonymous() {
			errorResponse(w, http.StatusUnauthorized, "authorization required")
			return
		}
		if len(allowed) > 0 {
			if _, ok := allowed[types.UserRole(user.Role)]; !ok {
				errorResponse(w, http.StatusForbidden, "forbidden: insufficient role")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// --- header parser ---
func extractBearerToken(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	return parts[1], nil
}

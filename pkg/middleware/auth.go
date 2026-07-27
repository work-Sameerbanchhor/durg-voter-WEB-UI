package middleware

import (
	"context"
	"net/http"
	"strings"

	"durg-voter-api/pkg/models"
)

const UserContextKey contextKey = "authenticated_user"

// Valid tokens mapping
var ValidTokens = map[string]models.User{
	"admin-token": {
		Username:    "admin",
		Role:        models.RoleAdmin,
		Permissions: []string{"read", "search", "filter", "group_by", "geo", "execute_sql"},
	},
	"admin-token-secret-key-12345": {
		Username:    "admin",
		Role:        models.RoleAdmin,
		Permissions: []string{"read", "search", "filter", "group_by", "geo", "execute_sql"},
	},
	"guest-token": {
		Username:    "guest",
		Role:        models.RoleGuest,
		Permissions: []string{"read", "search", "filter", "group_by", "geo"},
	},
	"guest-token-secret-key-67890": {
		Username:    "guest",
		Role:        models.RoleGuest,
		Permissions: []string{"read", "search", "filter", "group_by", "geo"},
	},
}

// Authenticate extracts authentication token and attaches user context
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)

		var user models.User
		if token != "" {
			if u, exists := ValidTokens[token]; exists {
				user = u
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":"Unauthorized: Invalid authentication token"}`))
				return
			}
		} else {
			// Default fallback for unauthenticated requests is guest role
			user = models.User{
				Username:    "guest_anonymous",
				Role:        models.RoleGuest,
				Permissions: []string{"read", "search", "filter", "group_by", "geo"},
			}
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware enforces role requirement (e.g., Admin only)
func RequireRole(requiredRole models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":"Unauthorized: Authentication required"}`))
				return
			}

			if user.Role != requiredRole {
				reqID := GetRequestID(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success":false,"error":"Forbidden: ` + string(requiredRole) + ` privileges required to perform this action","request_id":"` + reqID + `"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext retrieves authenticated user from request context
func GetUserFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(models.User)
	return user, ok
}

// Helper to extract token from Header or Query
func extractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 2. X-API-Key or X-Token header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return strings.TrimSpace(apiKey)
	}
	if xToken := r.Header.Get("X-Token"); xToken != "" {
		return strings.TrimSpace(xToken)
	}

	// 3. Query parameter ?token=...
	if queryToken := r.URL.Query().Get("token"); queryToken != "" {
		return strings.TrimSpace(queryToken)
	}

	return ""
}

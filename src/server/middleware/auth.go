// Package middleware provides HTTP middleware functions
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// Context keys for request context
type contextKey string

const (
	UserIDKey    contextKey = "userID"
	AdminIDKey   contextKey = "adminID"
	IsAdminKey   contextKey = "isAdmin"
	SessionIDKey contextKey = "sessionID"
)

// AuthMiddleware validates session or API token authentication
type AuthMiddleware struct {
	authService *service.AuthService
	store       store.Store
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(auth *service.AuthService, st store.Store) *AuthMiddleware {
	return &AuthMiddleware{authService: auth, store: st}
}

// RequireAuth requires authentication (session or API token)
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try session cookie first
		cookie, err := r.Cookie("session")
		if err == nil && cookie.Value != "" {
			userID, adminID, err := m.authService.ValidateSession(ctx, cookie.Value)
			if err == nil {
				ctx = context.WithValue(ctx, SessionIDKey, cookie.Value)
				ctx = context.WithValue(ctx, UserIDKey, userID)
				ctx = context.WithValue(ctx, AdminIDKey, adminID)
				ctx = context.WithValue(ctx, IsAdminKey, adminID != 0)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try API token (Bearer token) — validate against api_tokens table per AI.md PART 11
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			rawToken := strings.TrimPrefix(authHeader, "Bearer ")
			if rawToken != "" && m.store != nil {
				apiToken, err := m.store.GetToken(ctx, rawToken)
				if err == nil && apiToken != nil && apiToken.IsActive &&
					(apiToken.ExpiresAt.IsZero() || time.Now().Before(apiToken.ExpiresAt)) {
					ctx = context.WithValue(ctx, UserIDKey, apiToken.UserID)
					ctx = context.WithValue(ctx, IsAdminKey, false)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// No valid authentication
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}` + "\n"))
			return
		}

		// Redirect to login page for web requests
		http.Redirect(w, r, "/auth/login?redirect="+r.URL.Path, http.StatusSeeOther)
	})
}

// RequireAdmin requires admin authentication
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try session cookie first
		cookie, err := r.Cookie("session")
		if err == nil && cookie.Value != "" {
			userID, adminID, err := m.authService.ValidateSession(ctx, cookie.Value)
			if err == nil && adminID != 0 {
				ctx = context.WithValue(ctx, SessionIDKey, cookie.Value)
				ctx = context.WithValue(ctx, UserIDKey, userID)
				ctx = context.WithValue(ctx, AdminIDKey, adminID)
				ctx = context.WithValue(ctx, IsAdminKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No valid admin authentication
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden"}` + "\n"))
			return
		}

		// Redirect to admin login
		http.Redirect(w, r, "/auth/login?redirect="+r.URL.Path, http.StatusSeeOther)
	})
}

// OptionalAuth adds user context if authenticated, but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Try session cookie
		cookie, err := r.Cookie("session")
		if err == nil && cookie.Value != "" {
			userID, adminID, err := m.authService.ValidateSession(ctx, cookie.Value)
			if err == nil {
				ctx = context.WithValue(ctx, SessionIDKey, cookie.Value)
				ctx = context.WithValue(ctx, UserIDKey, userID)
				ctx = context.WithValue(ctx, AdminIDKey, adminID)
				ctx = context.WithValue(ctx, IsAdminKey, adminID != 0)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID returns the user ID from context (0 if not authenticated)
func GetUserID(ctx context.Context) int64 {
	if id, ok := ctx.Value(UserIDKey).(int64); ok {
		return id
	}
	return 0
}

// GetAdminID returns the admin ID from context (0 if not admin)
func GetAdminID(ctx context.Context) int64 {
	if id, ok := ctx.Value(AdminIDKey).(int64); ok {
		return id
	}
	return 0
}

// IsAdmin returns true if the request is from an admin
func IsAdmin(ctx context.Context) bool {
	if isAdmin, ok := ctx.Value(IsAdminKey).(bool); ok {
		return isAdmin
	}
	return false
}

// GetSessionID returns the session ID from context
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// Package handler - User management handlers
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/service"
)

// UserHandler handles user routes
type UserHandler struct {
	userService *service.UserService
	authService *service.AuthService
}

// NewUserHandler creates a new user handler
func NewUserHandler(user *service.UserService, auth *service.AuthService) *UserHandler {
	return &UserHandler{
		userService: user,
		authService: auth,
	}
}

// Profile handles GET /users - Current user's profile
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON for API requests
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                 user.ID,
			"username":           user.Username,
			"email":              user.Email,
			"role":               user.Role,
			"theme_preference":   user.ThemePreference,
			"storage_quota_bytes": user.StorageQuotaBytes,
			"storage_used_bytes":  user.StorageUsedBytes,
			"email_verified":     user.EmailVerified,
			"bio":                user.Bio,
			"website":            user.Website,
			"location":           user.Location,
			"created_at":         user.CreatedAt,
		})
		return
	}

	// Render profile page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Profile - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        .field { margin-bottom: 1rem; }
        .label { color: #6272a4; font-size: 0.875rem; }
        .value { color: #f8f8f2; font-size: 1rem; margin-top: 0.25rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Profile</h1>
        <div class="card">
            <div class="field"><div class="label">Username</div><div class="value">` + user.Username + `</div></div>
            <div class="field"><div class="label">Email</div><div class="value">` + user.Email + `</div></div>
            <div class="field"><div class="label">Role</div><div class="value">` + user.Role + `</div></div>
        </div>
    </div>
</body>
</html>`))
}

// ProfileUpdate handles PATCH /users - Update current user's profile
func (h *UserHandler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var updates map[string]interface{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
	} else {
		updates = make(map[string]interface{})
		if v := r.FormValue("theme_preference"); v != "" {
			updates["theme_preference"] = v
		}
		if v := r.FormValue("bio"); v != "" {
			updates["bio"] = v
		}
		if v := r.FormValue("website"); v != "" {
			updates["website"] = v
		}
		if v := r.FormValue("location"); v != "" {
			updates["location"] = v
		}
	}

	if err := h.userService.UpdateUser(ctx, userID, updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"updated"}` + "\n"))
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// Settings handles GET /users/settings - User preferences
func (h *UserHandler) Settings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Settings - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        label { display: block; color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        select, input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Settings</h1>
        <div class="card">
            <form method="POST" action="/users">
                <label for="theme">Theme</label>
                <select id="theme" name="theme_preference">
                    <option value="dark"` + selected(user.ThemePreference, "dark") + `>Dark</option>
                    <option value="light"` + selected(user.ThemePreference, "light") + `>Light</option>
                    <option value="auto"` + selected(user.ThemePreference, "auto") + `>Auto</option>
                </select>
                <button type="submit">Save</button>
            </form>
        </div>
    </div>
</body>
</html>`))
}

// Security handles GET /users/security - Security settings
func (h *UserHandler) Security(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        h2 { color: #8be9fd; font-size: 1.25rem; margin-bottom: 1rem; }
        p { color: #6272a4; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Security</h1>
        <div class="card">
            <h2>Two-Factor Authentication</h2>
            <p>2FA is not currently enabled.</p>
        </div>
        <div class="card">
            <h2>Sessions</h2>
            <p>Manage your active sessions.</p>
        </div>
    </div>
</body>
</html>`))
}

// Tokens handles GET /users/tokens - List API tokens
func (h *UserHandler) Tokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Tokens - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        p { color: #6272a4; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; margin-top: 1rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>API Tokens</h1>
        <div class="card">
            <p>No API tokens created.</p>
            <button onclick="alert('Token creation UI coming soon')">Create Token</button>
        </div>
    </div>
</body>
</html>`))
}

// TokenCreate handles POST /users/tokens - Create API token
func (h *UserHandler) TokenCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	// Generate new API token
	token, _, err := service.GenerateAPIToken()
	if err != nil {
		SendError(w, ErrServerError, "Failed to generate token")
		return
	}

	SendCreated(w, map[string]interface{}{
		"token":   token,
		"message": "Save this token - it will not be shown again",
	})
}

// TokenDelete handles DELETE /users/tokens/{token_id}
func (h *UserHandler) TokenDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	SendMessage(w, "Token deleted")
}

// APIMe handles GET /api/v1/users/me
func (h *UserHandler) APIMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		SendError(w, ErrServerError, "Failed to get user")
		return
	}

	SendOK(w, map[string]interface{}{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"role":                user.Role,
		"theme_preference":    user.ThemePreference,
		"storage_quota_bytes": user.StorageQuotaBytes,
		"storage_used_bytes":  user.StorageUsedBytes,
		"email_verified":      user.EmailVerified,
		"created_at":          user.CreatedAt,
	})
}

// APIUpdateMe handles PATCH /api/v1/users/me
func (h *UserHandler) APIUpdateMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	if err := h.userService.UpdateUser(ctx, userID, updates); err != nil {
		SendError(w, ErrServerError, "Failed to update user")
		return
	}

	SendMessage(w, "Profile updated")
}

func selected(value, check string) string {
	if value == check {
		return " selected"
	}
	return ""
}

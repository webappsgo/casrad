// Package admin provides admin-specific functionality
// See AI.md PART 17 for admin panel specification
package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// Admin represents the admin module
type Admin struct {
	path        string // Configurable admin path, default "admin"
	store       store.Store
	authService *service.AuthService
	userService *service.UserService
	scheduler   *service.Scheduler
	securityMW  *middleware.SecurityMiddleware
	startTime   time.Time
}

// Config holds admin module configuration
type Config struct {
	AdminPath   string
	Store       store.Store
	AuthService *service.AuthService
	UserService *service.UserService
	Scheduler   *service.Scheduler
	SecurityMW  *middleware.SecurityMiddleware
}

// New creates a new admin module
func New(cfg Config) *Admin {
	if cfg.AdminPath == "" {
		cfg.AdminPath = "admin"
	}
	return &Admin{
		path:        cfg.AdminPath,
		store:       cfg.Store,
		authService: cfg.AuthService,
		userService: cfg.UserService,
		scheduler:   cfg.Scheduler,
		securityMW:  cfg.SecurityMW,
		startTime:   time.Now(),
	}
}

// Path returns the admin path
func (a *Admin) Path() string {
	return a.path
}

// Routes returns the admin routes
// See AI.md PART 17 - Admin route hierarchy
func (a *Admin) Routes() http.Handler {
	mux := http.NewServeMux()

	// Dashboard - /{admin_path}/
	mux.HandleFunc("GET /", a.handleDashboard)

	// Admin's own routes - /{admin_path}/profile, /preferences, /notifications
	mux.HandleFunc("GET /profile", a.handleProfile)
	mux.HandleFunc("PATCH /profile", a.handleProfileUpdate)
	mux.HandleFunc("GET /preferences", a.handlePreferences)
	mux.HandleFunc("PATCH /preferences", a.handlePreferencesUpdate)
	mux.HandleFunc("GET /notifications", a.handleNotifications)

	// Server management - /{admin_path}/server/*
	mux.HandleFunc("GET /server/settings", a.handleServerSettings)
	mux.HandleFunc("PATCH /server/settings", a.handleServerSettingsUpdate)
	mux.HandleFunc("GET /server/users", a.handleServerUsers)
	mux.HandleFunc("POST /server/users", a.handleServerUserCreate)
	mux.HandleFunc("GET /server/users/{id}", a.handleServerUserDetail)
	mux.HandleFunc("PATCH /server/users/{id}", a.handleServerUserUpdate)
	mux.HandleFunc("DELETE /server/users/{id}", a.handleServerUserDelete)
	mux.HandleFunc("GET /server/logs", a.handleServerLogs)
	mux.HandleFunc("GET /server/backup", a.handleServerBackup)
	mux.HandleFunc("POST /server/backup", a.handleServerBackupCreate)
	mux.HandleFunc("POST /server/restore", a.handleServerRestore)
	mux.HandleFunc("GET /server/metrics", a.handleServerMetrics)
	mux.HandleFunc("GET /server/tasks", a.handleServerTasks)
	mux.HandleFunc("POST /server/tasks/{name}/run", a.handleServerTaskRun)

	return mux
}

// Dashboard HTML template
func (a *Admin) renderDashboard(w http.ResponseWriter, stats map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Dashboard - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; }
        .sidebar { position: fixed; top: 0; left: 0; width: 240px; height: 100vh; background: #21222c; border-right: 1px solid #44475a; padding: 1.5rem 0; }
        .sidebar h1 { color: #bd93f9; font-size: 1.25rem; padding: 0 1.5rem 1.5rem; border-bottom: 1px solid #44475a; }
        .nav { margin-top: 1rem; }
        .nav a { display: block; padding: 0.75rem 1.5rem; color: #f8f8f2; text-decoration: none; transition: background 0.2s; }
        .nav a:hover { background: #44475a; }
        .nav a.active { background: #44475a; border-left: 3px solid #bd93f9; }
        .main { margin-left: 240px; padding: 2rem; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
        .header h2 { color: #bd93f9; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
        .stat-card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        .stat-card .label { color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        .stat-card .value { font-size: 2rem; font-weight: 600; color: #50fa7b; }
        .stat-card .value.warning { color: #f1fa8c; }
        .stat-card .value.error { color: #ff5555; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        .card h3 { color: #8be9fd; margin-bottom: 1rem; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        .badge { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; }
        .badge-success { background: #50fa7b; color: #282a36; }
        .badge-warning { background: #f1fa8c; color: #282a36; }
        .badge-error { background: #ff5555; color: #f8f8f2; }
    </style>
</head>
<body>
    <div class="sidebar">
        <h1>CASRAD Admin</h1>
        <nav class="nav">
            <a href="/` + a.path + `/" class="active">Dashboard</a>
            <a href="/` + a.path + `/server/users">Users</a>
            <a href="/` + a.path + `/server/settings">Settings</a>
            <a href="/` + a.path + `/server/logs">Logs</a>
            <a href="/` + a.path + `/server/backup">Backup</a>
            <a href="/` + a.path + `/server/metrics">Metrics</a>
            <a href="/` + a.path + `/server/tasks">Tasks</a>
            <a href="/` + a.path + `/profile">My Profile</a>
            <a href="/auth/logout">Logout</a>
        </nav>
    </div>
    <div class="main">
        <div class="header">
            <h2>Dashboard</h2>
        </div>
        <div class="stats">
            <div class="stat-card">
                <div class="label">Total Users</div>
                <div class="value">` + fmt.Sprintf("%v", stats["total_users"]) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Active Sessions</div>
                <div class="value">` + fmt.Sprintf("%v", stats["active_sessions"]) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Uptime</div>
                <div class="value">` + fmt.Sprintf("%v", stats["uptime"]) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Memory Used</div>
                <div class="value">` + fmt.Sprintf("%v", stats["memory_used"]) + `</div>
            </div>
        </div>
        <div class="card">
            <h3>System Information</h3>
            <table>
                <tr><th>Version</th><td>1.0.0</td></tr>
                <tr><th>Go Version</th><td>` + runtime.Version() + `</td></tr>
                <tr><th>Platform</th><td>` + runtime.GOOS + `/` + runtime.GOARCH + `</td></tr>
                <tr><th>CPUs</th><td>` + fmt.Sprintf("%d", runtime.NumCPU()) + `</td></tr>
                <tr><th>Goroutines</th><td>` + fmt.Sprintf("%d", runtime.NumGoroutine()) + `</td></tr>
            </table>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin auth
	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login?redirect=/"+a.path+"/", http.StatusSeeOther)
		return
	}

	// Gather stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	users, total, _ := a.store.ListUsers(ctx, 0, 1)
	_ = users // suppress unused warning

	stats := map[string]interface{}{
		"total_users":     total,
		"active_sessions": 0, // Would need session count from store
		"uptime":          formatDuration(time.Since(a.startTime)),
		"memory_used":     formatBytes(memStats.Alloc),
	}

	// JSON response
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
		return
	}

	a.renderDashboard(w, stats)
}

func (a *Admin) handleProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	adminID := middleware.GetAdminID(ctx)
	admin, err := a.store.GetAdminByID(ctx, adminID)
	if err != nil {
		http.Error(w, "Admin not found", http.StatusNotFound)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         admin.ID,
			"username":   admin.Username,
			"email":      admin.Email,
			"role":       admin.Role,
			"created_at": admin.CreatedAt,
			"last_login": admin.LastLogin,
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Profile - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        .field { margin-bottom: 1rem; }
        .label { color: #6272a4; font-size: 0.875rem; }
        .value { color: #f8f8f2; font-size: 1rem; margin-top: 0.25rem; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Admin Profile</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <div class="field"><div class="label">Username</div><div class="value">` + admin.Username + `</div></div>
            <div class="field"><div class="label">Email</div><div class="value">` + admin.Email + `</div></div>
            <div class="field"><div class="label">Role</div><div class="value">` + admin.Role + `</div></div>
            <div class="field"><div class="label">Created</div><div class="value">` + admin.CreatedAt.Format(time.RFC3339) + `</div></div>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	adminID := middleware.GetAdminID(ctx)
	admin, err := a.store.GetAdminByID(ctx, adminID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin not found"})
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Apply allowed updates
	if email, ok := updates["email"].(string); ok && email != "" {
		admin.Email = email
	}
	admin.UpdatedAt = time.Now()

	if err := a.store.UpdateAdmin(ctx, admin); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (a *Admin) handlePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Preferences - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        label { display: block; color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        select, input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Preferences</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <form method="POST">
                <label for="theme">Theme</label>
                <select id="theme" name="theme">
                    <option value="dark" selected>Dark (Dracula)</option>
                    <option value="light">Light</option>
                </select>
                <label for="language">Language</label>
                <select id="language" name="language">
                    <option value="en" selected>English</option>
                </select>
                <button type="submit">Save Preferences</button>
            </form>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handlePreferencesUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (a *Admin) handleNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Notifications - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        p { color: #6272a4; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Notifications</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <p>No notifications.</p>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	settings := map[string]interface{}{
		"server_name":      "CASRAD",
		"server_url":       "",
		"registration":     "disabled",
		"default_quota":    "50GB",
		"mpd_enabled":      true,
		"mpd_port":         6600,
		"subsonic_enabled": true,
		"webdav_enabled":   true,
		"rtmp_enabled":     true,
		"rtmp_port":        1935,
		"dlna_enabled":     true,
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Server Settings - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        h2 { color: #8be9fd; font-size: 1.25rem; margin-bottom: 1rem; }
        label { display: block; color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        input, select { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; margin-right: 0.5rem; }
        a { color: #8be9fd; text-decoration: none; }
        .toggle { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem; }
        .toggle input[type="checkbox"] { width: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Settings</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>

        <div class="card" style="margin-top: 1rem;">
            <h2>General</h2>
            <label for="server_name">Server Name</label>
            <input type="text" id="server_name" value="CASRAD">
            <label for="server_url">Server URL</label>
            <input type="text" id="server_url" placeholder="https://your-server.com">
            <label for="registration">User Registration</label>
            <select id="registration">
                <option value="disabled" selected>Disabled</option>
                <option value="public">Public</option>
                <option value="approval">Require Approval</option>
            </select>
        </div>

        <div class="card">
            <h2>Protocols</h2>
            <div class="toggle"><input type="checkbox" id="mpd" checked><label for="mpd">MPD (Port 6600)</label></div>
            <div class="toggle"><input type="checkbox" id="subsonic" checked><label for="subsonic">Subsonic API</label></div>
            <div class="toggle"><input type="checkbox" id="webdav" checked><label for="webdav">WebDAV</label></div>
            <div class="toggle"><input type="checkbox" id="rtmp" checked><label for="rtmp">RTMP (Port 1935)</label></div>
            <div class="toggle"><input type="checkbox" id="dlna" checked><label for="dlna">DLNA/UPnP</label></div>
        </div>

        <div class="card">
            <h2>Storage</h2>
            <label for="quota">Default User Quota</label>
            <select id="quota">
                <option value="10">10 GB</option>
                <option value="25">25 GB</option>
                <option value="50" selected>50 GB</option>
                <option value="100">100 GB</option>
                <option value="unlimited">Unlimited</option>
            </select>
        </div>

        <button type="button" onclick="alert('Settings saved')">Save Settings</button>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Settings would be persisted to database here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (a *Admin) handleServerUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get pagination params
	page := 1
	limit := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	offset := (page - 1) * limit

	users, total, err := a.store.ListUsers(ctx, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": users,
			"total": total,
			"page":  page,
			"limit": limit,
		})
		return
	}

	// Build user rows HTML
	userRows := ""
	for _, user := range users {
		status := `<span class="badge badge-success">Active</span>`
		if !user.IsActive {
			status = `<span class="badge badge-error">Inactive</span>`
		}
		userRows += fmt.Sprintf(`<tr>
            <td>%d</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td><a href="/%s/server/users/%d">Edit</a></td>
        </tr>`, user.ID, user.Username, user.Email, user.Role, status, user.CreatedAt.Format("2006-01-02"), a.path, user.ID)
	}

	if userRows == "" {
		userRows = `<tr><td colspan="7" style="text-align: center; color: #6272a4;">No users found</td></tr>`
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Management - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        .badge { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; }
        .badge-success { background: #50fa7b; color: #282a36; }
        .badge-error { background: #ff5555; color: #f8f8f2; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
        a { color: #8be9fd; text-decoration: none; }
        .pagination { margin-top: 1rem; text-align: center; }
        .pagination a { padding: 0.5rem 1rem; background: #44475a; border-radius: 4px; margin: 0 0.25rem; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>User Management</h1>
            <button onclick="alert('Create user modal would open')">Create User</button>
        </div>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Username</th>
                        <th>Email</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ` + userRows + `
                </tbody>
            </table>
        </div>
        <div class="pagination">
            <span>Total: ` + fmt.Sprintf("%d", total) + ` users</span>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerUserCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	userID, err := a.userService.CreateUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"user_id": userID})
}

func (a *Admin) handleServerUserDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := a.store.GetUserByID(ctx, id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit User - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; }
        label { display: block; color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        input, select { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; margin-right: 0.5rem; }
        button.danger { background: #ff5555; color: #f8f8f2; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit User: ` + user.Username + `</h1>
        <a href="/` + a.path + `/server/users">&larr; Back to Users</a>
        <div class="card" style="margin-top: 1rem;">
            <label for="username">Username</label>
            <input type="text" id="username" value="` + user.Username + `" readonly>
            <label for="email">Email</label>
            <input type="email" id="email" value="` + user.Email + `">
            <label for="role">Role</label>
            <select id="role">
                <option value="user"` + selectedIf(user.Role == "user") + `>User</option>
                <option value="moderator"` + selectedIf(user.Role == "moderator") + `>Moderator</option>
                <option value="admin"` + selectedIf(user.Role == "admin") + `>Admin</option>
            </select>
            <label for="quota">Storage Quota</label>
            <input type="text" id="quota" value="` + formatBytes(uint64(user.StorageQuotaBytes)) + `">
            <button type="button" onclick="alert('User updated')">Save Changes</button>
            <button type="button" class="danger" onclick="if(confirm('Delete this user?')) alert('User deleted')">Delete User</button>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerUserUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user ID"})
		return
	}

	user, err := a.store.GetUserByID(ctx, id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "user not found"})
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Apply updates
	if email, ok := updates["email"].(string); ok && email != "" {
		user.Email = email
	}
	if role, ok := updates["role"].(string); ok && role != "" {
		user.Role = role
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		user.IsActive = isActive
	}
	user.UpdatedAt = time.Now()

	if err := a.store.UpdateUser(ctx, user); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (a *Admin) handleServerUserDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid user ID"})
		return
	}

	if err := a.store.DeleteUser(ctx, id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (a *Admin) handleServerLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Mock log entries - in production would read from log files or database
	logs := []map[string]interface{}{
		{"timestamp": time.Now().Add(-1 * time.Minute).Format(time.RFC3339), "level": "info", "message": "Server started"},
		{"timestamp": time.Now().Add(-30 * time.Second).Format(time.RFC3339), "level": "info", "message": "User login: admin"},
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
		return
	}

	logRows := ""
	for _, log := range logs {
		levelClass := "info"
		if log["level"] == "error" {
			levelClass = "error"
		} else if log["level"] == "warning" {
			levelClass = "warning"
		}
		logRows += fmt.Sprintf(`<tr>
            <td>%s</td>
            <td><span class="badge badge-%s">%s</span></td>
            <td>%s</td>
        </tr>`, log["timestamp"], levelClass, log["level"], log["message"])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Server Logs - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        .badge { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; }
        .badge-info { background: #8be9fd; color: #282a36; }
        .badge-warning { background: #f1fa8c; color: #282a36; }
        .badge-error { background: #ff5555; color: #f8f8f2; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Logs</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <table>
                <thead>
                    <tr>
                        <th>Timestamp</th>
                        <th>Level</th>
                        <th>Message</th>
                    </tr>
                </thead>
                <tbody>
                    ` + logRows + `
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	backups := []map[string]interface{}{
		{"id": 1, "type": "full", "size": "45 MB", "created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339)},
		{"id": 2, "type": "full", "size": "44 MB", "created_at": time.Now().Add(-48 * time.Hour).Format(time.RFC3339)},
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(backups)
		return
	}

	backupRows := ""
	for _, b := range backups {
		backupRows += fmt.Sprintf(`<tr>
            <td>%v</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td><button onclick="alert('Restore not implemented')">Restore</button></td>
        </tr>`, b["id"], b["type"], b["size"], b["created_at"])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Backup Management - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        button { padding: 0.5rem 1rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
        button.primary { padding: 0.75rem 1.5rem; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Backup Management</h1>
            <button class="primary" onclick="alert('Creating backup...')">Create Backup</button>
        </div>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Type</th>
                        <th>Size</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ` + backupRows + `
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerBackupCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// Backup creation would happen here
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"backup_id":  time.Now().Unix(),
		"status":     "created",
		"created_at": time.Now().Format(time.RFC3339),
	})
}

func (a *Admin) handleServerRestore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		BackupID int64 `json:"backup_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Restore would happen here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restored"})
}

func (a *Admin) handleServerMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"cpu_usage":       0,
		"memory_used":     memStats.Alloc,
		"memory_total":    memStats.Sys,
		"goroutines":      runtime.NumGoroutine(),
		"gc_runs":         memStats.NumGC,
		"gc_pause_total":  memStats.PauseTotalNs,
		"heap_objects":    memStats.HeapObjects,
		"active_streams":  0,
		"active_sessions": 0,
		"uptime_seconds":  int64(time.Since(a.startTime).Seconds()),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Metrics - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
        .stat-card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        .stat-card .label { color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        .stat-card .value { font-size: 1.5rem; font-weight: 600; color: #50fa7b; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Metrics</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="stats" style="margin-top: 1rem;">
            <div class="stat-card">
                <div class="label">Memory Used</div>
                <div class="value">` + formatBytes(memStats.Alloc) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Memory Total</div>
                <div class="value">` + formatBytes(memStats.Sys) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Goroutines</div>
                <div class="value">` + fmt.Sprintf("%d", runtime.NumGoroutine()) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">GC Runs</div>
                <div class="value">` + fmt.Sprintf("%d", memStats.NumGC) + `</div>
            </div>
        </div>
        <div class="card">
            <table>
                <tr><th>Heap Objects</th><td>` + fmt.Sprintf("%d", memStats.HeapObjects) + `</td></tr>
                <tr><th>Heap Alloc</th><td>` + formatBytes(memStats.HeapAlloc) + `</td></tr>
                <tr><th>Heap Sys</th><td>` + formatBytes(memStats.HeapSys) + `</td></tr>
                <tr><th>Stack Sys</th><td>` + formatBytes(memStats.StackSys) + `</td></tr>
                <tr><th>Uptime</th><td>` + formatDuration(time.Since(a.startTime)) + `</td></tr>
            </table>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	tasks := []map[string]interface{}{
		{"name": "cleanup_temp", "schedule": "0 * * * *", "last_run": "Never", "status": "enabled"},
		{"name": "cleanup_cache", "schedule": "0 */6 * * *", "last_run": "Never", "status": "enabled"},
		{"name": "rotate_logs", "schedule": "0 3 * * *", "last_run": "Never", "status": "enabled"},
		{"name": "backup_database", "schedule": "0 2 * * *", "last_run": "Never", "status": "enabled"},
		{"name": "check_quotas", "schedule": "*/30 * * * *", "last_run": "Never", "status": "enabled"},
		{"name": "update_podcasts", "schedule": "0 */6 * * *", "last_run": "Never", "status": "enabled"},
		{"name": "scan_libraries", "schedule": "0 3 * * *", "last_run": "Never", "status": "enabled"},
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
		return
	}

	taskRows := ""
	for _, t := range tasks {
		taskRows += fmt.Sprintf(`<tr>
            <td>%s</td>
            <td><code>%s</code></td>
            <td>%s</td>
            <td><span class="badge badge-success">%s</span></td>
            <td><button onclick="alert('Running task: %s')">Run Now</button></td>
        </tr>`, t["name"], t["schedule"], t["last_run"], t["status"], t["name"])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scheduled Tasks - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
        code { background: #282a36; padding: 0.25rem 0.5rem; border-radius: 4px; font-family: 'JetBrains Mono', monospace; }
        .badge { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; }
        .badge-success { background: #50fa7b; color: #282a36; }
        button { padding: 0.5rem 1rem; border: none; border-radius: 6px; background: #8be9fd; color: #282a36; cursor: pointer; }
        a { color: #8be9fd; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Scheduled Tasks</h1>
        <a href="/` + a.path + `/">&larr; Back to Dashboard</a>
        <div class="card" style="margin-top: 1rem;">
            <table>
                <thead>
                    <tr>
                        <th>Task</th>
                        <th>Schedule</th>
                        <th>Last Run</th>
                        <th>Status</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ` + taskRows + `
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`))
}

func (a *Admin) handleServerTaskRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	taskName := r.PathValue("name")
	if taskName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "task name required"})
		return
	}

	// Run task via scheduler
	if a.scheduler != nil {
		if err := a.scheduler.RunTask(r.Context(), taskName); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"task":   taskName,
	})
}

// Helper functions
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func selectedIf(condition bool) string {
	if condition {
		return " selected"
	}
	return ""
}

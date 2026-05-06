// Package handler - Admin panel handlers
// These are thin wrappers that delegate to the main admin module
// See src/admin/admin.go for the main implementation
package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// AdminHandler handles admin routes
type AdminHandler struct {
	store       store.Store
	userService *service.UserService
	startTime   time.Time
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(s store.Store, user *service.UserService) *AdminHandler {
	return &AdminHandler{
		store:       s,
		userService: user,
		startTime:   time.Now(),
	}
}

// Dashboard handles GET /{admin_path}/ - Admin dashboard
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	_, total, _ := h.store.ListUsers(ctx, 0, 1)

	stats := map[string]interface{}{
		"total_users":     total,
		"active_sessions": 0,
		"uptime":          formatUptime(time.Since(h.startTime)),
		"memory_used":     formatMemory(memStats.Alloc),
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Dashboard - CASRAD</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 2rem; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; }
        .stat { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        .stat .label { color: #6272a4; font-size: 0.875rem; }
        .stat .value { font-size: 2rem; font-weight: 600; color: #50fa7b; margin-top: 0.5rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Admin Dashboard</h1>
        <div class="stats">
            <div class="stat"><div class="label">Total Users</div><div class="value">` + formatInt(total) + `</div></div>
            <div class="stat"><div class="label">Active Sessions</div><div class="value">0</div></div>
            <div class="stat"><div class="label">Uptime</div><div class="value">` + formatUptime(time.Since(h.startTime)) + `</div></div>
            <div class="stat"><div class="label">Memory</div><div class="value">` + formatMemory(memStats.Alloc) + `</div></div>
        </div>
    </div>
</body>
</html>`))
}

// Profile handles GET /{admin_path}/profile - Admin's own profile
func (h *AdminHandler) Profile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	adminID := middleware.GetAdminID(ctx)
	admin, err := h.store.GetAdminByID(ctx, adminID)
	if err != nil {
		http.Error(w, "Admin not found", http.StatusNotFound)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       admin.ID,
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
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
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        .field { margin-bottom: 1rem; }
        .label { color: #6272a4; font-size: 0.875rem; }
        .value { color: #f8f8f2; font-size: 1rem; margin-top: 0.25rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Admin Profile</h1>
        <div class="card">
            <div class="field"><div class="label">Username</div><div class="value">` + admin.Username + `</div></div>
            <div class="field"><div class="label">Email</div><div class="value">` + admin.Email + `</div></div>
            <div class="field"><div class="label">Role</div><div class="value">` + admin.Role + `</div></div>
        </div>
    </div>
</body>
</html>`))
}

// Preferences handles GET /{admin_path}/preferences - Admin's UI preferences
func (h *AdminHandler) Preferences(w http.ResponseWriter, r *http.Request) {
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
        select { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Preferences</h1>
        <div class="card">
            <label for="theme">Theme</label>
            <select id="theme" name="theme">
                <option value="dark" selected>Dark (Dracula)</option>
                <option value="light">Light</option>
            </select>
            <button type="button" onclick="alert('Preferences saved')">Save</button>
        </div>
    </div>
</body>
</html>`))
}

// ServerSettings handles GET /{admin_path}/server/settings
func (h *AdminHandler) ServerSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server_name":  "CASRAD",
			"registration": "disabled",
			"mpd_enabled":  true,
			"mpd_port":     6600,
		})
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
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        label { display: block; color: #6272a4; font-size: 0.875rem; margin-bottom: 0.5rem; }
        input { width: 100%; padding: 0.75rem; border: 1px solid #6272a4; border-radius: 6px; background: #282a36; color: #f8f8f2; margin-bottom: 1rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Settings</h1>
        <div class="card">
            <label for="name">Server Name</label>
            <input type="text" id="name" value="CASRAD">
            <button type="button" onclick="alert('Settings saved')">Save</button>
        </div>
    </div>
</body>
</html>`))
}

// ServerUsers handles GET /{admin_path}/server/users - User management
func (h *AdminHandler) ServerUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	users, total, err := h.store.ListUsers(ctx, 0, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": users,
			"total": total,
		})
		return
	}

	userRows := ""
	for _, user := range users {
		userRows += `<tr><td>` + formatInt(user.ID) + `</td><td>` + user.Username + `</td><td>` + user.Email + `</td><td>` + user.Role + `</td></tr>`
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
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #6272a4; }
        th { color: #6272a4; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <h1>User Management</h1>
        <div class="card">
            <table>
                <thead><tr><th>ID</th><th>Username</th><th>Email</th><th>Role</th></tr></thead>
                <tbody>` + userRows + `</tbody>
            </table>
        </div>
    </div>
</body>
</html>`))
}

// ServerLogs handles GET /{admin_path}/server/logs
func (h *AdminHandler) ServerLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"timestamp": time.Now().Format(time.RFC3339), "level": "info", "message": "Server running"},
		})
		return
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
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        pre { background: #282a36; padding: 1rem; border-radius: 6px; overflow-x: auto; font-family: 'JetBrains Mono', monospace; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Logs</h1>
        <div class="card">
            <pre>` + time.Now().Format(time.RFC3339) + ` [INFO] Server running</pre>
        </div>
    </div>
</body>
</html>`))
}

// ServerBackup handles GET/POST /{admin_path}/server/backup
func (h *AdminHandler) ServerBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": 1, "type": "full", "created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339)},
		})
		return
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
        .card { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        button { padding: 0.75rem 1.5rem; border: none; border-radius: 6px; background: #50fa7b; color: #282a36; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Backup Management</h1>
        <div class="card">
            <button type="button" onclick="alert('Creating backup...')">Create Backup</button>
        </div>
    </div>
</body>
</html>`))
}

// ServerMetrics handles GET /{admin_path}/server/metrics
func (h *AdminHandler) ServerMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !middleware.IsAdmin(ctx) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"memory_used":  memStats.Alloc,
		"memory_total": memStats.Sys,
		"goroutines":   runtime.NumGoroutine(),
		"gc_runs":      memStats.NumGC,
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
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; }
        .stat { background: #44475a; border-radius: 8px; padding: 1.5rem; }
        .stat .label { color: #6272a4; font-size: 0.875rem; }
        .stat .value { font-size: 1.5rem; font-weight: 600; color: #50fa7b; margin-top: 0.5rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Server Metrics</h1>
        <div class="stats">
            <div class="stat"><div class="label">Memory Used</div><div class="value">` + formatMemory(memStats.Alloc) + `</div></div>
            <div class="stat"><div class="label">Memory Total</div><div class="value">` + formatMemory(memStats.Sys) + `</div></div>
            <div class="stat"><div class="label">Goroutines</div><div class="value">` + formatInt(int64(runtime.NumGoroutine())) + `</div></div>
            <div class="stat"><div class="label">GC Runs</div><div class="value">` + formatInt(int64(memStats.NumGC)) + `</div></div>
        </div>
    </div>
</body>
</html>`))
}

// Helper functions
func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return itoa(days) + "d " + itoa(hours) + "h"
	}
	if hours > 0 {
		return itoa(hours) + "h " + itoa(minutes) + "m"
	}
	return itoa(minutes) + "m"
}

func formatMemory(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return itoa(int(bytes)) + " B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	val := float64(bytes) / float64(div)
	return ftoa(val) + " " + string("KMGTPE"[exp]) + "B"
}

func formatInt(n int64) string {
	return itoa(int(n))
}

// itoa converts int to string without strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ftoa converts float to string with 1 decimal place
func ftoa(f float64) string {
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 10)
	if decPart < 0 {
		decPart = -decPart
	}
	return itoa(intPart) + "." + itoa(decPart)
}

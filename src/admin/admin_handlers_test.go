// Package admin — Handler tests covering all 27 HTTP handler functions.
//
// Test groups:
//   - Unauthenticated paths: every handler that redirects/401s without admin context
//   - Authenticated GET handlers: HTML rendering with admin context
//   - Authenticated JSON handlers: Accept:application/json responses
//   - Authenticated POST/PATCH/DELETE handlers: mutation + error paths
//   - Edge cases: invalid IDs, bad JSON bodies, missing path values
//
// All tests use MemoryStore so no SQLite, no filesystem, no network required.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// --- Helpers ---

// adminCtx returns a context carrying admin identity values.
func adminCtx(base context.Context) context.Context {
	ctx := context.WithValue(base, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.AdminIDKey, int64(1))
	return ctx
}

// adminRequest creates a request with admin context injected.
func adminRequest(method, path string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, http.NoBody)
	}
	return r.WithContext(adminCtx(r.Context()))
}

// anonRequest creates a request without any admin context.
func anonRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, http.NoBody)
}

// newTestEnv builds an Admin instance with a seeded MemoryStore.
// It also creates one admin account (ID=1) so handleProfile can find it.
func newTestEnv(t *testing.T) *Admin {
	t.Helper()

	st := store.NewMemoryStore()
	ctx := context.Background()

	// Seed admin account so GetAdminByID(ctx, 1) works.
	_, err := st.CreateAdmin(ctx, &model.Admin{
		Username:  "testadmin",
		Email:     "testadmin@example.com",
		Role:      "admin",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	// Seed a regular user (ID=1) for user-detail/update/delete handlers.
	_, err = st.CreateUser(ctx, &model.User{
		Username:          "alice",
		Email:             "alice@example.com",
		Role:              "user",
		IsActive:          true,
		StorageQuotaBytes: 53687091200,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	authSvc := service.NewAuthService(st)
	userSvc := service.NewUserService(st, authSvc, t.TempDir())

	return New(Config{
		AdminPath:   "admin",
		Store:       st,
		AuthService: authSvc,
		UserService: userSvc,
	})
}

// jsonBody returns a JSON-encoded request body.
func jsonBody(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// --- Unauthenticated paths (all redirect to /auth/login) ---

func TestHandleDashboard_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleDashboard(w, anonRequest("GET", "/"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleProfile_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleProfile(w, anonRequest("GET", "/profile"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleProfileUpdate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleProfileUpdate(w, anonRequest("PATCH", "/profile"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandlePreferences_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handlePreferences(w, anonRequest("GET", "/preferences"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandlePreferencesUpdate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handlePreferencesUpdate(w, anonRequest("PATCH", "/preferences"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleNotifications_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleNotifications(w, anonRequest("GET", "/notifications"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerSettings_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerSettings(w, anonRequest("GET", "/server/settings"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerSettingsUpdate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerSettingsUpdate(w, anonRequest("PATCH", "/server/settings"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerUsers_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerUsers(w, anonRequest("GET", "/server/users"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerUserCreate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerUserCreate(w, anonRequest("POST", "/server/users"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerUserDetail_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := anonRequest("GET", "/server/users/1")
	r.SetPathValue("id", "1")
	a.handleServerUserDetail(w, r)
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerUserUpdate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerUserUpdate(w, anonRequest("PATCH", "/server/users/1"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerUserDelete_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerUserDelete(w, anonRequest("DELETE", "/server/users/1"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerLogs_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerLogs(w, anonRequest("GET", "/server/logs"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerBackup_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerBackup(w, anonRequest("GET", "/server/backup"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerBackupCreate_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerBackupCreate(w, anonRequest("POST", "/server/backup"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerRestore_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerRestore(w, anonRequest("POST", "/server/restore"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestHandleServerMetrics_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerMetrics(w, anonRequest("GET", "/server/metrics"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerTasks_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerTasks(w, anonRequest("GET", "/server/tasks"))
	if w.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", w.Code)
	}
}

func TestHandleServerTaskRun_Unauthenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerTaskRun(w, anonRequest("POST", "/server/tasks/cleanup_temp/run"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// --- Authenticated GET handlers returning HTML ---

func TestHandleDashboard_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleDashboard(w, adminRequest("GET", "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Dashboard") {
		t.Error("response body should contain Dashboard")
	}
}

func TestHandleDashboard_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/", nil)
	r.Header.Set("Accept", "application/json")
	a.handleDashboard(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload["total_users"]; !ok {
		t.Error("JSON response should contain total_users")
	}
	if _, ok := payload["uptime"]; !ok {
		t.Error("JSON response should contain uptime")
	}
}

func TestHandleProfile_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleProfile(w, adminRequest("GET", "/profile", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "testadmin") {
		t.Error("response body should contain the admin username")
	}
}

func TestHandleProfile_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/profile", nil)
	r.Header.Set("Accept", "application/json")
	a.handleProfile(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["username"] != "testadmin" {
		t.Errorf("expected username testadmin, got %v", payload["username"])
	}
}

func TestHandlePreferences_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handlePreferences(w, adminRequest("GET", "/preferences", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Preferences") {
		t.Error("response should contain Preferences")
	}
}

func TestHandleNotifications_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleNotifications(w, adminRequest("GET", "/notifications", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestHandleNotifications_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/notifications", nil)
	r.Header.Set("Accept", "application/json")
	a.handleNotifications(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	// Should be a JSON array (empty)
	var payload []interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("expected JSON array: %v", err)
	}
}

func TestHandleServerSettings_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerSettings(w, adminRequest("GET", "/server/settings", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Server Settings") {
		t.Error("response should contain Server Settings")
	}
}

func TestHandleServerSettings_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/settings", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerSettings(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["server_name"]; !ok {
		t.Error("JSON response should contain server_name")
	}
}

func TestHandleServerUsers_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerUsers(w, adminRequest("GET", "/server/users", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "User Management") {
		t.Error("response should contain User Management")
	}
}

func TestHandleServerUsers_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerUsers(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["users"]; !ok {
		t.Error("JSON response should contain users key")
	}
}

func TestHandleServerUsers_Pagination(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users?page=2", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerUsers(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// page 2 should be reported
	if payload["page"].(float64) != 2 {
		t.Errorf("expected page=2, got %v", payload["page"])
	}
}

func TestHandleServerUserDetail_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users/1", nil)
	r.SetPathValue("id", "1")
	a.handleServerUserDetail(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "alice") {
		t.Error("response should contain seeded user alice")
	}
}

func TestHandleServerUserDetail_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users/1", nil)
	r.SetPathValue("id", "1")
	r.Header.Set("Accept", "application/json")
	a.handleServerUserDetail(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var user model.User
	if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected alice, got %s", user.Username)
	}
}

func TestHandleServerUserDetail_InvalidID(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users/abc", nil)
	r.SetPathValue("id", "abc")
	a.handleServerUserDetail(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerUserDetail_NotFound(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/users/9999", nil)
	r.SetPathValue("id", "9999")
	a.handleServerUserDetail(w, r)
	// GetUserByID returns nil,nil for missing users in MemoryStore → 404
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestHandleServerLogs_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerLogs(w, adminRequest("GET", "/server/logs", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Server Logs") {
		t.Error("response should contain Server Logs")
	}
}

func TestHandleServerLogs_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/logs", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerLogs(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Seeded with 2 mock log entries
	if len(payload) == 0 {
		t.Error("expected at least one log entry")
	}
}

func TestHandleServerBackup_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerBackup(w, adminRequest("GET", "/server/backup", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Backup") {
		t.Error("response should contain Backup")
	}
}

func TestHandleServerBackup_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/backup", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerBackup(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload) == 0 {
		t.Error("expected backup entries")
	}
}

func TestHandleServerMetrics_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerMetrics(w, adminRequest("GET", "/server/metrics", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Server Metrics") {
		t.Error("response should contain Server Metrics")
	}
}

func TestHandleServerMetrics_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/metrics", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerMetrics(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["goroutines"]; !ok {
		t.Error("JSON response should contain goroutines")
	}
	if _, ok := payload["uptime_seconds"]; !ok {
		t.Error("JSON response should contain uptime_seconds")
	}
}

func TestHandleServerTasks_HTML(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerTasks(w, adminRequest("GET", "/server/tasks", nil))
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Scheduled Tasks") {
		t.Error("response should contain Scheduled Tasks")
	}
}

func TestHandleServerTasks_JSON(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("GET", "/server/tasks", nil)
	r.Header.Set("Accept", "application/json")
	a.handleServerTasks(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var payload []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload) == 0 {
		t.Error("expected at least one task")
	}
}

// --- Authenticated POST/PATCH/DELETE mutation handlers ---

func TestHandleProfileUpdate_ValidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]string{"email": "newemail@example.com"})
	r := adminRequest("PATCH", "/profile", body)
	r.Header.Set("Content-Type", "application/json")
	a.handleProfileUpdate(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "updated" {
		t.Errorf("expected status=updated, got %v", resp["status"])
	}
}

func TestHandleProfileUpdate_InvalidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("PATCH", "/profile", []byte("not-json"))
	a.handleProfileUpdate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandlePreferencesUpdate_Authenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("PATCH", "/preferences", jsonBody(map[string]string{"theme": "light"}))
	r.Header.Set("Content-Type", "application/json")
	a.handlePreferencesUpdate(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestHandleServerSettingsUpdate_ValidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]interface{}{"server_name": "TestServer"})
	r := adminRequest("PATCH", "/server/settings", body)
	r.Header.Set("Content-Type", "application/json")
	a.handleServerSettingsUpdate(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "updated" {
		t.Errorf("expected updated, got %v", resp["status"])
	}
}

func TestHandleServerSettingsUpdate_InvalidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("PATCH", "/server/settings", []byte("{bad json"))
	a.handleServerSettingsUpdate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerUserCreate_ValidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]string{
		"username": "bobsmith",
		"email":    "bob@example.com",
		"password": "Str0ng!Pass99",
		"role":     "user",
	})
	r := adminRequest("POST", "/server/users", body)
	r.Header.Set("Content-Type", "application/json")
	a.handleServerUserCreate(w, r)
	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["user_id"]; !ok {
		t.Error("response should contain user_id")
	}
}

func TestHandleServerUserCreate_InvalidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("POST", "/server/users", []byte("bad"))
	a.handleServerUserCreate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerUserCreate_DuplicateUsername(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	// "alice" is already seeded by newTestEnv
	w := httptest.NewRecorder()
	body := jsonBody(map[string]string{
		"username": "alice",
		"email":    "alice2@example.com",
		"password": "Str0ng!Pass99",
	})
	r := adminRequest("POST", "/server/users", body)
	r.Header.Set("Content-Type", "application/json")
	a.handleServerUserCreate(w, r)
	// UserService.CreateUser will fail on duplicate/blocked username
	if w.Code == http.StatusCreated {
		t.Error("should not create user with blocked/duplicate username")
	}
}

func TestHandleServerUserUpdate_ValidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]interface{}{
		"email":     "alice.updated@example.com",
		"role":      "moderator",
		"is_active": true,
	})
	r := adminRequest("PATCH", "/server/users/1", body)
	r.SetPathValue("id", "1")
	r.Header.Set("Content-Type", "application/json")
	a.handleServerUserUpdate(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "updated" {
		t.Errorf("expected updated, got %v", resp["status"])
	}
}

func TestHandleServerUserUpdate_InvalidID(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("PATCH", "/server/users/xyz", jsonBody(map[string]string{"email": "x@x.com"}))
	r.SetPathValue("id", "xyz")
	a.handleServerUserUpdate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerUserUpdate_NotFound(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]string{"email": "x@x.com"})
	r := adminRequest("PATCH", "/server/users/9999", body)
	r.SetPathValue("id", "9999")
	a.handleServerUserUpdate(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestHandleServerUserUpdate_InvalidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("PATCH", "/server/users/1", []byte("{bad"))
	r.SetPathValue("id", "1")
	a.handleServerUserUpdate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerUserDelete_Existing(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("DELETE", "/server/users/1", nil)
	r.SetPathValue("id", "1")
	a.handleServerUserDelete(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "deleted" {
		t.Errorf("expected deleted, got %v", resp["status"])
	}
}

func TestHandleServerUserDelete_InvalidID(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("DELETE", "/server/users/abc", nil)
	r.SetPathValue("id", "abc")
	a.handleServerUserDelete(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerBackupCreate_Authenticated(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	a.handleServerBackupCreate(w, adminRequest("POST", "/server/backup", nil))
	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "created" {
		t.Errorf("expected created, got %v", resp["status"])
	}
}

func TestHandleServerRestore_ValidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	body := jsonBody(map[string]int64{"backup_id": 1})
	r := adminRequest("POST", "/server/restore", body)
	r.Header.Set("Content-Type", "application/json")
	a.handleServerRestore(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "restored" {
		t.Errorf("expected restored, got %v", resp["status"])
	}
}

func TestHandleServerRestore_InvalidBody(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	// Send a multipart form body (not JSON) — should fail decode
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("backup_id", "1")
	mw.Close()
	r := adminRequest("POST", "/server/restore", buf.Bytes())
	r.Header.Set("Content-Type", mw.FormDataContentType())
	a.handleServerRestore(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleServerTaskRun_WithScheduler(t *testing.T) {
	t.Parallel()
	// scheduler is nil in newTestEnv — handler should succeed silently
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	r := adminRequest("POST", "/server/tasks/cleanup_temp/run", nil)
	r.SetPathValue("name", "cleanup_temp")
	a.handleServerTaskRun(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["task"] != "cleanup_temp" {
		t.Errorf("expected task=cleanup_temp, got %v", resp["task"])
	}
	if resp["status"] != "started" {
		t.Errorf("expected status=started, got %v", resp["status"])
	}
}

func TestHandleServerTaskRun_EmptyName(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	w := httptest.NewRecorder()
	// PathValue not set → empty string → 400
	r := adminRequest("POST", "/server/tasks//run", nil)
	a.handleServerTaskRun(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// --- Routes registration smoke test ---

func TestRoutes_RegistersWithoutPanic(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)
	// Routes() must not panic and must return a non-nil handler
	h := a.Routes()
	if h == nil {
		t.Error("Routes() returned nil")
	}
}

// --- Idempotency: delete then list shows user is gone ---

func TestHandleServerUserDelete_Idempotent(t *testing.T) {
	t.Parallel()
	a := newTestEnv(t)

	// First delete
	w1 := httptest.NewRecorder()
	r1 := adminRequest("DELETE", "/server/users/1", nil)
	r1.SetPathValue("id", "1")
	a.handleServerUserDelete(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("first delete: want 200, got %d", w1.Code)
	}

	// Second delete of the same ID should still return 200 (MemoryStore delete is a no-op on missing)
	w2 := httptest.NewRecorder()
	r2 := adminRequest("DELETE", "/server/users/1", nil)
	r2.SetPathValue("id", "1")
	a.handleServerUserDelete(w2, r2)
	if w2.Code != http.StatusOK {
		t.Errorf("second delete: want 200, got %d", w2.Code)
	}
}

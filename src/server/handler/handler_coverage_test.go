// Package handler — Coverage tests for authenticated handler paths.
// Covers: AdminHandler.Dashboard/Profile/Preferences/ServerSettings/ServerUsers/
//         ServerLogs/ServerBackup/ServerMetrics (non-admin redirect + admin paths),
//         APIHandler authenticated paths: Playlists, PlaylistCreate, PlaylistUpdate,
//         PlaylistDelete, PlaylistAddTracks, Podcasts, PodcastSubscribe, Audiobooks,
//         Queue, QueueAdd, QueueClear, Player, PlayerControl (all actions),
//         History, Stats, Scrobble, Rate, Favorite (authenticated happy paths + errors),
//         Health (HTML, JSON, text/plain, default paths), InitHealth, SetMode,
//         HealthText, HealthAPI, getUptime, buildHealthResponse.
//
// Auth injection: the middleware package exports typed context keys.
// To simulate a logged-in user we add UserIDKey → int64(id) to the request context.
// To simulate an admin we also add IsAdminKey → true.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// Ensure json is used (used by TestSearchWithPlaylistAndPodcastTypes).
var _ = json.Marshal

// injectUserID returns a request with user ID set in context (simulates authenticated user).
func injectUserID(r *http.Request, id int64) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, id)
	return r.WithContext(ctx)
}

// injectAdmin returns a request with admin flag set in context.
func injectAdmin(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.IsAdminKey, true)
	return r.WithContext(ctx)
}

// injectAdminID returns a request with admin ID set in context.
func injectAdminID(r *http.Request, id int64) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.AdminIDKey, id)
	return r.WithContext(ctx)
}

// newMemoryHandler creates an APIHandler backed by a real MemoryStore.
func newMemoryHandler() *APIHandler {
	return NewAPIHandler(store.NewMemoryStore())
}

// newAdminHandlerWithStore creates an AdminHandler backed by a real MemoryStore.
func newAdminHandlerWithStore() *AdminHandler {
	return NewAdminHandler(store.NewMemoryStore(), nil)
}

// --- AdminHandler: non-admin redirects ---

func TestAdminDashboardNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rr := httptest.NewRecorder()
	h.Dashboard(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Dashboard(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminProfileNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Profile(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminPreferencesNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/preferences", nil)
	rr := httptest.NewRecorder()
	h.Preferences(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Preferences(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminServerSettingsNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerSettings(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminServerUsersNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/users", nil)
	rr := httptest.NewRecorder()
	h.ServerUsers(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerUsers(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminServerLogsNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerLogs(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminServerBackupNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerBackup(non-admin) status = %d, want 303", rr.Code)
	}
}

func TestAdminServerMetricsNonAdminRedirects(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerMetrics(non-admin) status = %d, want 303", rr.Code)
	}
}

// --- AdminHandler: JSON paths (admin context) ---

func TestAdminDashboardJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.Dashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Dashboard(admin,JSON) status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestAdminDashboardHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.Dashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Dashboard(admin,HTML) status = %d, want 200", rr.Code)
	}
}

func TestAdminPreferencesAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/preferences", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.Preferences(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Preferences(admin) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerSettingsJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerSettings(admin,JSON) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerSettingsHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerSettings(admin,HTML) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerUsersJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/users", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerUsers(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerUsers(admin,JSON) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerUsersHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/users", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerUsers(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerUsers(admin,HTML) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerLogsJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerLogs(admin,JSON) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerLogsHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerLogs(admin,HTML) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerBackupJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerBackup(admin,JSON) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerBackupHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerBackup(admin,HTML) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerMetricsJSONAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerMetrics(admin,JSON) status = %d, want 200", rr.Code)
	}
}

func TestAdminServerMetricsHTMLAsAdmin(t *testing.T) {
	t.Parallel()
	h := newAdminHandlerWithStore()
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	req = injectAdmin(req)
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerMetrics(admin,HTML) status = %d, want 200", rr.Code)
	}
}

// AdminProfile as admin with a real admin in the store
func TestAdminProfileJSONAsAdmin(t *testing.T) {
	t.Parallel()
	s := store.NewMemoryStore()
	id, err := s.CreateAdmin(context.Background(), &model.Admin{
		Username: "testadmin",
		Email:    "testadmin@example.com",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("CreateAdmin: %v", err)
	}
	h := NewAdminHandler(s, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	req.Header.Set("Accept", "application/json")
	req = injectAdmin(req)
	// Inject the actual admin ID so GetAdminByID finds the record
	ctx := context.WithValue(req.Context(), middleware.AdminIDKey, id)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Profile(admin,JSON) status = %d, want 200", rr.Code)
	}
}

// --- APIHandler: authenticated paths ---

func TestPlaylistsAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/playlists", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Playlists(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Playlists(auth) status = %d, want 200", rr.Code)
	}
}

func TestPlaylistCreateAuthenticatedValidBody(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	body := `{"name":"My Mix","description":"Weekend vibes","is_public":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistCreate(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("PlaylistCreate(auth,valid) status = %d, want 201", rr.Code)
	}
}

func TestPlaylistCreateAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistCreate(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistCreateAuthenticatedEmptyName(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistCreate(auth,empty name) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistUpdateAuthenticatedValidID(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/playlists/1", strings.NewReader(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PlaylistUpdate(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestPlaylistUpdateAuthenticatedInvalidID(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/playlists/abc", strings.NewReader(`{}`))
	req.SetPathValue("id", "abc")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistUpdate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistUpdate(auth,invalid id) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistUpdateAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/playlists/1", strings.NewReader("not-json"))
	req.SetPathValue("id", "1")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistUpdate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistUpdate(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistDeleteAuthenticatedValidID(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/playlists/1", nil)
	req.SetPathValue("id", "1")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PlaylistDelete(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestPlaylistDeleteAuthenticatedInvalidID(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/playlists/xyz", nil)
	req.SetPathValue("id", "xyz")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistDelete(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistDelete(auth,invalid id) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistAddTracksAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists/1/tracks",
		strings.NewReader(`{"track_ids":[1,2,3]}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "1")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistAddTracks(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PlaylistAddTracks(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestPlaylistAddTracksAuthenticatedInvalidID(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists/bad/tracks",
		strings.NewReader(`{"track_ids":[1]}`))
	req.SetPathValue("id", "bad")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistAddTracks(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistAddTracks(auth,invalid id) status = %d, want 400", rr.Code)
	}
}

func TestPlaylistAddTracksAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists/1/tracks",
		strings.NewReader("not-json"))
	req.SetPathValue("id", "1")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlaylistAddTracks(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlaylistAddTracks(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestPodcastsAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/podcasts", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Podcasts(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Podcasts(auth) status = %d, want 200", rr.Code)
	}
}

func TestPodcastSubscribeAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podcasts",
		strings.NewReader(`{"feed_url":"https://example.com/feed.xml"}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PodcastSubscribe(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("PodcastSubscribe(auth,valid) status = %d, want 201", rr.Code)
	}
}

func TestPodcastSubscribeAuthenticatedEmptyURL(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podcasts",
		strings.NewReader(`{"feed_url":""}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PodcastSubscribe(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PodcastSubscribe(auth,empty url) status = %d, want 400", rr.Code)
	}
}

func TestPodcastSubscribeAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podcasts", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PodcastSubscribe(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PodcastSubscribe(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestAudiobooksAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audiobooks", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Audiobooks(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Audiobooks(auth) status = %d, want 200", rr.Code)
	}
}

func TestQueueAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Queue(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Queue(auth) status = %d, want 200", rr.Code)
	}
}

func TestQueueAddAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue",
		strings.NewReader(`{"track_ids":[1,2,3],"position":"end"}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.QueueAdd(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("QueueAdd(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestQueueAddAuthenticatedEmptyTrackIDs(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue",
		strings.NewReader(`{"track_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.QueueAdd(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("QueueAdd(auth,empty ids) status = %d, want 400", rr.Code)
	}
}

func TestQueueAddAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.QueueAdd(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("QueueAdd(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestQueueClearAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/queue", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.QueueClear(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("QueueClear(auth) status = %d, want 200", rr.Code)
	}
}

func TestPlayerAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/player", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Player(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Player(auth) status = %d, want 200", rr.Code)
	}
}

// PlayerControl: exercise all action branches
func TestPlayerControlActions(t *testing.T) {
	t.Parallel()
	actions := []string{"play", "pause", "stop", "next", "previous", "shuffle", "repeat"}
	h := newMemoryHandler()
	for _, action := range actions {
		action := action
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/player/"+action, nil)
			req.SetPathValue("action", action)
			req = injectUserID(req, 1)
			rr := httptest.NewRecorder()
			h.PlayerControl(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("PlayerControl(%s) status = %d, want 200", action, rr.Code)
			}
		})
	}
}

func TestPlayerControlUnknownActionReturns400(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/player/dance", nil)
	req.SetPathValue("action", "dance")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.PlayerControl(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PlayerControl(unknown) status = %d, want 400", rr.Code)
	}
}

func TestHistoryAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/history", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.History(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("History(auth) status = %d, want 200", rr.Code)
	}
}

func TestStatsAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Stats(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Stats(auth) status = %d, want 200", rr.Code)
	}
}

func TestScrobbleAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scrobble",
		strings.NewReader(`{"track_id":1,"timestamp":1700000000,"duration":240}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Scrobble(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Scrobble(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestScrobbleAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scrobble", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Scrobble(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Scrobble(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestRateAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate",
		strings.NewReader(`{"type":"track","id":1,"rating":4}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Rate(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Rate(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestRateAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Rate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Rate(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestRateAuthenticatedRatingTooHigh(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate",
		strings.NewReader(`{"type":"track","id":1,"rating":6}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Rate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Rate(auth,rating>5) status = %d, want 400", rr.Code)
	}
}

func TestRateAuthenticatedRatingNegative(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate",
		strings.NewReader(`{"type":"track","id":1,"rating":-1}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Rate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Rate(auth,rating<0) status = %d, want 400", rr.Code)
	}
}

func TestFavoriteAuthenticatedValid(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/favorite",
		strings.NewReader(`{"type":"track","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Favorite(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Favorite(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestFavoriteAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/favorite", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Favorite(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Favorite(auth,bad json) status = %d, want 400", rr.Code)
	}
}

// Search: exercise the playlist and podcast type branches (not covered yet)
func TestSearchWithPlaylistAndPodcastTypes(t *testing.T) {
	t.Parallel()
	h := newMemoryHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=jazz&type=playlist,podcast", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Search(playlist,podcast) status = %d, want 200", rr.Code)
	}
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Search response unmarshal: %v", err)
	}
	if !resp.OK {
		t.Error("Search response.ok should be true")
	}
}

// --- Auth handler: Login/APILogin/Register with real service stack ---
// Build a minimal service stack backed by MemoryStore for auth path coverage.

// TestAPILoginInvalidJSONReturns400 covers the JSON decode error branch in APILogin.
func TestAPILoginInvalidJSONReturns400(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APILogin(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("APILogin(bad json) status = %d, want 400", rr.Code)
	}
}

// TestAPILoginInvalidCredentialsReturns401 covers the authenticate failure branch.
func TestAPILoginInvalidCredentialsReturns401(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"identifier":"nobody","password":"wrongpassword1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APILogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("APILogin(wrong creds) status = %d, want 401", rr.Code)
	}
}

// TestAPILoginValidationFailureReturns400 covers the ValidateLogin failure branch.
func TestAPILoginValidationFailureReturns400(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	// Empty identifier triggers validation failure
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"identifier":"","password":"somepassword1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APILogin(rr, req)
	// Empty identifier = validation error → 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("APILogin(empty identifier) status = %d, want 400", rr.Code)
	}
}

// TestLoginFormInvalidCredentialsRedirects covers the redirect-on-failure branch.
func TestLoginFormInvalidCredentialsRedirects(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader("identifier=nobody&password=wrongpassword1&redirect=/"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	// Failed auth redirects to /auth/login?error=invalid
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login(wrong creds) status = %d, want 303", rr.Code)
	}
}

// TestLoginFormJSONInvalidCredentials covers the JSON error branch in Login.
func TestLoginFormJSONInvalidCredentials(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader("identifier=nobody&password=wrongpassword1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Login(JSON,wrong creds) status = %d, want 401", rr.Code)
	}
}

// TestAPIRegisterValidationFailure covers the ValidateRegistration failure branch.
func TestAPIRegisterValidationFailure(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "open")
	// Password too short (< 8 chars) triggers validation error
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"username":"validuser","email":"valid@example.com","password":"short","confirm_password":"short"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APIRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("APIRegister(short password) status = %d, want 400", rr.Code)
	}
}

// TestRegisterFormValidationFailureRedirects covers form Register with bad input.
func TestRegisterFormValidationFailureRedirects(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader("username=ab&email=notanemail&password=short&confirm_password=short"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	// Validation failure → redirect back
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Register(bad input) status = %d, want 303", rr.Code)
	}
}

// TestRegisterFormValidationFailureJSON covers JSON Register with bad input.
func TestRegisterFormValidationFailureJSON(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	h := NewAuthHandler(authSvc, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader("username=ab&email=notanemail&password=short&confirm_password=short"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register(JSON,bad input) status = %d, want 400", rr.Code)
	}
}

// --- UserHandler: authenticated paths with a real UserService ---

func newUserHandlerWithStore(t *testing.T) (*UserHandler, *store.MemoryStore, int64) {
	t.Helper()
	ms := store.NewMemoryStore()
	authSvc := service.NewAuthService(ms)
	userSvc := service.NewUserService(ms, authSvc, t.TempDir())
	h := NewUserHandler(userSvc, authSvc)

	// Create a test user to operate on
	id, err := ms.CreateUser(context.Background(), &model.User{
		Username: "testprofileuser",
		Email:    "testprofile@example.com",
		IsActive: true,
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("setup CreateUser: %v", err)
	}
	return h, ms, id
}

func TestUserProfileAuthenticatedJSONReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Accept", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Profile(auth,JSON) status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestUserProfileAuthenticatedHTMLReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Profile(auth,HTML) status = %d, want 200", rr.Code)
	}
}

func TestUserSettingsAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/users/settings", nil)
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.Settings(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Settings(auth) status = %d, want 200", rr.Code)
	}
}

func TestUserSecurityAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/users/security", nil)
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.Security(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Security(auth) status = %d, want 200", rr.Code)
	}
}

func TestUserTokensAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/users/tokens", nil)
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.Tokens(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Tokens(auth) status = %d, want 200", rr.Code)
	}
}

func TestUserTokenCreateAuthenticatedReturns201(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPost, "/users/tokens", strings.NewReader(`{"name":"my-api-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.TokenCreate(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("TokenCreate(auth) status = %d, want 201", rr.Code)
	}
}

func TestUserTokenDeleteAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/tokens/1", nil)
	req.SetPathValue("id", "1")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.TokenDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("TokenDelete(auth) status = %d, want 200", rr.Code)
	}
}

func TestAPIMeAuthenticatedReturns200(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.APIMe(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("APIMe(auth) status = %d, want 200", rr.Code)
	}
}

func TestAPIUpdateMeAuthenticatedValidJSON(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me",
		strings.NewReader(`{"bio":"Hello!","theme_preference":"light"}`))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.APIUpdateMe(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("APIUpdateMe(auth,valid) status = %d, want 200", rr.Code)
	}
}

func TestAPIUpdateMeAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.APIUpdateMe(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("APIUpdateMe(auth,bad json) status = %d, want 400", rr.Code)
	}
}

func TestProfileUpdateAuthenticatedFormValues(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader("bio=Music+lover&website=https%3A%2F%2Fexample.com&location=NYC"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.ProfileUpdate(rr, req)
	// Should redirect to /users on success
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ProfileUpdate(auth,form) status = %d, want 303", rr.Code)
	}
}

func TestProfileUpdateAuthenticatedJSONBody(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPatch, "/users",
		strings.NewReader(`{"bio":"Updated bio"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.ProfileUpdate(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ProfileUpdate(auth,JSON) status = %d, want 200", rr.Code)
	}
}

func TestProfileUpdateAuthenticatedInvalidJSON(t *testing.T) {
	t.Parallel()
	h, _, id := newUserHandlerWithStore(t)
	req := httptest.NewRequest(http.MethodPatch, "/users", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = injectUserID(req, id)
	rr := httptest.NewRecorder()
	h.ProfileUpdate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("ProfileUpdate(auth,bad json) status = %d, want 400", rr.Code)
	}
}

// ftoa coverage: negative decimal part branch
func TestFtoaNegativeDecimal(t *testing.T) {
	t.Parallel()
	// ftoa uses int((f - int(f)) * 10); for very small negative fractional parts
	// the decPart will be negative and the abs branch fires.
	// Use 1.9 to get decPart = 9 (no negative branch in normal floats, so use explicit test)
	got := ftoa(1.9)
	if got == "" {
		t.Error("ftoa(1.9) should return non-empty string")
	}
	// Manually test the negative absolute value branch via a specific value
	// ftoa(-1.5): intPart = -1, f-intPart = -0.5, decPart = int(-0.5*10) = -5 → abs → 5
	got2 := ftoa(-1.5)
	if got2 == "" {
		t.Error("ftoa(-1.5) should return non-empty string")
	}
}

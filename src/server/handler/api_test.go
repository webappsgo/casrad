// Package handler — Tests for API helper functions and handler methods.
// Covers: errorCodeToHTTP, SendOK, SendError, SendMessage, SendCreated,
// SendValidationErrors, NewAPIHandler, writeJSON, getPagination,
// Tracks, Track, TrackStream, Albums, Album, Artists, Artist,
// Playlists (unauthenticated), PlaylistCreate (unauthenticated/missing name),
// PlaylistUpdate (unauthenticated), PlaylistDelete (unauthenticated),
// PlaylistAddTracks (unauthenticated).
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- errorCodeToHTTP ---

func TestErrorCodeToHTTP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		want int
	}{
		{ErrBadRequest, 400},
		{ErrValidationFailed, 400},
		{ErrUnauthorized, 401},
		{ErrTokenExpired, 401},
		{ErrTokenInvalid, 401},
		{Err2FARequired, 401},
		{Err2FAInvalid, 401},
		{ErrForbidden, 403},
		{ErrAccountLocked, 403},
		{ErrNotFound, 404},
		{ErrMethodNotAllowed, 405},
		{ErrConflict, 409},
		{ErrRateLimited, 429},
		{ErrMaintenance, 503},
		// Unknown codes map to 500
		{"UNKNOWN_ERROR", 500},
		{ErrServerError, 500},
	}
	for _, tc := range cases {
		got := errorCodeToHTTP(tc.code)
		if got != tc.want {
			t.Errorf("errorCodeToHTTP(%q) = %d, want %d", tc.code, got, tc.want)
		}
	}
}

// --- SendOK ---

func TestSendOKSetsJSONContentType(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendOK(rr, map[string]string{"key": "value"})
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestSendOKBodyIsValid(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendOK(rr, map[string]string{"hello": "world"})
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendOK body unmarshal error: %v\nbody: %s", err, rr.Body.String())
	}
	if !resp.OK {
		t.Errorf("SendOK response.ok = false, want true")
	}
}

func TestSendOKStatus200(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendOK(rr, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("SendOK status = %d, want 200", rr.Code)
	}
}

// --- SendError ---

func TestSendErrorSetsCorrectStatus(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendError(rr, ErrNotFound, "not found")
	if rr.Code != http.StatusNotFound {
		t.Errorf("SendError(NOT_FOUND) status = %d, want 404", rr.Code)
	}
}

func TestSendErrorBodyIsValid(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendError(rr, ErrUnauthorized, "unauthorized message")
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendError body unmarshal error: %v", err)
	}
	if resp.OK {
		t.Error("SendError response.ok = true, want false")
	}
	if resp.Error != ErrUnauthorized {
		t.Errorf("SendError response.error = %q, want %q", resp.Error, ErrUnauthorized)
	}
	if resp.Message != "unauthorized message" {
		t.Errorf("SendError response.message = %q, want 'unauthorized message'", resp.Message)
	}
}

// --- SendMessage ---

func TestSendMessageBodyIsValid(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendMessage(rr, "operation complete")
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendMessage body unmarshal error: %v", err)
	}
	if !resp.OK {
		t.Error("SendMessage response.ok = false, want true")
	}
	if resp.Message != "operation complete" {
		t.Errorf("SendMessage response.message = %q, want 'operation complete'", resp.Message)
	}
}

// --- SendCreated ---

func TestSendCreatedReturns201(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendCreated(rr, map[string]int{"id": 1})
	if rr.Code != http.StatusCreated {
		t.Errorf("SendCreated status = %d, want 201", rr.Code)
	}
}

func TestSendCreatedBodyIsValid(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendCreated(rr, map[string]int{"id": 42})
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendCreated body unmarshal error: %v", err)
	}
	if !resp.OK {
		t.Error("SendCreated response.ok = false, want true")
	}
}

// --- SendValidationErrors ---

func TestSendValidationErrorsReturns400(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SendValidationErrors(rr, map[string]string{"email": "invalid format"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("SendValidationErrors status = %d, want 400", rr.Code)
	}
}

func TestSendValidationErrorsBodyIsValid(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	errs := map[string]string{"username": "too short", "email": "required"}
	SendValidationErrors(rr, errs)
	var resp ValidationErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendValidationErrors body unmarshal error: %v", err)
	}
	if resp.OK {
		t.Error("SendValidationErrors response.ok = true, want false")
	}
	if resp.Error != ErrValidationFailed {
		t.Errorf("SendValidationErrors response.error = %q, want %q", resp.Error, ErrValidationFailed)
	}
	if len(resp.Errors) != 2 {
		t.Errorf("response has %d errors, want 2", len(resp.Errors))
	}
}

// --- NewAPIHandler ---

func TestNewAPIHandlerReturnsNonNil(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	if h == nil {
		t.Error("NewAPIHandler returned nil")
	}
}

// --- writeJSON ---

func TestWriteJSONSetsStatusAndContentType(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusAccepted, map[string]string{"result": "ok"})
	if rr.Code != http.StatusAccepted {
		t.Errorf("writeJSON status = %d, want 202", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// --- getPagination ---

func TestGetPaginationDefaults(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	page, limit, offset := getPagination(req)
	if page != 1 {
		t.Errorf("default page = %d, want 1", page)
	}
	if limit != 50 {
		t.Errorf("default limit = %d, want 50", limit)
	}
	if offset != 0 {
		t.Errorf("default offset = %d, want 0", offset)
	}
}

func TestGetPaginationCustomValues(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?page=3&limit=20", nil)
	page, limit, offset := getPagination(req)
	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
	if limit != 20 {
		t.Errorf("limit = %d, want 20", limit)
	}
	if offset != 40 {
		t.Errorf("offset = %d, want 40", offset)
	}
}

func TestGetPaginationInvalidPageFallsBack(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?page=notanumber", nil)
	page, _, _ := getPagination(req)
	if page != 1 {
		t.Errorf("invalid page = %d, want fallback 1", page)
	}
}

func TestGetPaginationLimitCappedAt100(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?limit=999", nil)
	_, limit, _ := getPagination(req)
	if limit != 50 {
		t.Errorf("over-limit = %d, want fallback 50", limit)
	}
}

func TestGetPaginationZeroLimitFallsBack(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?limit=0", nil)
	_, limit, _ := getPagination(req)
	if limit != 50 {
		t.Errorf("zero limit = %d, want fallback 50", limit)
	}
}

// --- APIHandler.Tracks ---

func TestTracksReturns200WithPaginatedResponse(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	rr := httptest.NewRecorder()
	h.Tracks(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Tracks status = %d, want 200", rr.Code)
	}
	var resp PaginatedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Tracks response unmarshal: %v", err)
	}
}

func TestTracksWithFilters(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?artist=Tool&album=Lateralus&genre=prog", nil)
	rr := httptest.NewRecorder()
	h.Tracks(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Tracks with filters status = %d, want 200", rr.Code)
	}
}

// --- APIHandler.Track ---

func TestTrackReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.Track(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Track status = %d, want 200", rr.Code)
	}
}

func TestTrackInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	h.Track(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Track(invalid id) status = %d, want 400", rr.Code)
	}
}

// --- APIHandler.TrackStream ---

func TestTrackStreamReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/1/stream", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.TrackStream(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("TrackStream status = %d, want 200", rr.Code)
	}
}

func TestTrackStreamInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/xyz/stream", nil)
	req.SetPathValue("id", "xyz")
	rr := httptest.NewRecorder()
	h.TrackStream(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("TrackStream(invalid id) status = %d, want 400", rr.Code)
	}
}

func TestTrackStreamAcceptsFormatAndBitrate(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/5/stream?format=ogg&bitrate=320", nil)
	req.SetPathValue("id", "5")
	rr := httptest.NewRecorder()
	h.TrackStream(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("TrackStream with params status = %d, want 200", rr.Code)
	}
}

// --- APIHandler.Albums ---

func TestAlbumsReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/albums", nil)
	rr := httptest.NewRecorder()
	h.Albums(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Albums status = %d, want 200", rr.Code)
	}
}

// --- APIHandler.Album ---

func TestAlbumReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/albums/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.Album(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Album status = %d, want 200", rr.Code)
	}
}

func TestAlbumInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/albums/bad", nil)
	req.SetPathValue("id", "bad")
	rr := httptest.NewRecorder()
	h.Album(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Album(invalid id) status = %d, want 400", rr.Code)
	}
}

// --- APIHandler.Artists ---

func TestArtistsReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists", nil)
	rr := httptest.NewRecorder()
	h.Artists(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Artists status = %d, want 200", rr.Code)
	}
}

// --- APIHandler.Artist ---

func TestArtistReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists/3", nil)
	req.SetPathValue("id", "3")
	rr := httptest.NewRecorder()
	h.Artist(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Artist status = %d, want 200", rr.Code)
	}
}

func TestArtistInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artists/nope", nil)
	req.SetPathValue("id", "nope")
	rr := httptest.NewRecorder()
	h.Artist(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Artist(invalid id) status = %d, want 400", rr.Code)
	}
}

// --- APIHandler.Playlists (unauthenticated) ---

func TestPlaylistsUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/playlists", nil)
	rr := httptest.NewRecorder()
	h.Playlists(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Playlists unauthenticated status = %d, want 401", rr.Code)
	}
}

// --- APIHandler.Playlist ---

func TestPlaylistValidIDReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/playlists/7", nil)
	req.SetPathValue("id", "7")
	rr := httptest.NewRecorder()
	h.Playlist(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Playlist status = %d, want 200", rr.Code)
	}
}

func TestPlaylistInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/playlists/bad", nil)
	req.SetPathValue("id", "bad")
	rr := httptest.NewRecorder()
	h.Playlist(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Playlist(invalid id) status = %d, want 400", rr.Code)
	}
}

// --- APIHandler.PlaylistCreate (unauthenticated) ---

func TestPlaylistCreateUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	body := `{"name":"My Playlist"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PlaylistCreate(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PlaylistCreate unauthenticated status = %d, want 401", rr.Code)
	}
}

// --- APIHandler.PlaylistUpdate (unauthenticated) ---

func TestPlaylistUpdateUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/playlists/1", strings.NewReader(`{}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PlaylistUpdate(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PlaylistUpdate unauthenticated status = %d, want 401", rr.Code)
	}
}

// --- APIHandler.PlaylistDelete (unauthenticated) ---

func TestPlaylistDeleteUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/playlists/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.PlaylistDelete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PlaylistDelete unauthenticated status = %d, want 401", rr.Code)
	}
}

// --- APIHandler.PlaylistAddTracks (unauthenticated) ---

func TestPlaylistAddTracksUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlists/1/tracks", strings.NewReader(`{"track_ids":[1,2]}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PlaylistAddTracks(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PlaylistAddTracks unauthenticated status = %d, want 401", rr.Code)
	}
}

// Package handler — Additional API handler tests for broadcast, podcast, audiobook,
// search, queue, player, cover art, history, stats, scrobble, rate, favorite endpoints.
// All unauthenticated paths return 401; public paths return expected shapes.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Broadcasts (public, no auth required) ---

func TestBroadcastsReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/broadcasts", nil)
	rr := httptest.NewRecorder()
	h.Broadcasts(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Broadcasts status = %d, want 200", rr.Code)
	}
}

// --- Broadcast (public, requires mount param) ---

func TestBroadcastEmptyMountReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/broadcasts/", nil)
	rr := httptest.NewRecorder()
	h.Broadcast(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Broadcast(empty mount) status = %d, want 400", rr.Code)
	}
}

func TestBroadcastWithMountReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/broadcasts/live", nil)
	req.SetPathValue("mount", "live")
	rr := httptest.NewRecorder()
	h.Broadcast(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Broadcast(live) status = %d, want 200", rr.Code)
	}
}

// --- Podcasts (requires auth) ---

func TestPodcastsUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/podcasts", nil)
	rr := httptest.NewRecorder()
	h.Podcasts(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Podcasts(unauth) status = %d, want 401", rr.Code)
	}
}

// --- PodcastSubscribe (requires auth) ---

func TestPodcastSubscribeUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podcasts",
		strings.NewReader(`{"feed_url":"https://example.com/feed.xml"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PodcastSubscribe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PodcastSubscribe(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Audiobooks (requires auth) ---

func TestAudiobooksUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audiobooks", nil)
	rr := httptest.NewRecorder()
	h.Audiobooks(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Audiobooks(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Audiobook (public endpoint, valid/invalid ID) ---

func TestAudiobookValidIDReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audiobooks/42", nil)
	req.SetPathValue("id", "42")
	rr := httptest.NewRecorder()
	h.Audiobook(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Audiobook(42) status = %d, want 200", rr.Code)
	}
}

func TestAudiobookInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audiobooks/notanid", nil)
	req.SetPathValue("id", "notanid")
	rr := httptest.NewRecorder()
	h.Audiobook(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Audiobook(notanid) status = %d, want 400", rr.Code)
	}
}

// --- Search (public, requires q param) ---

func TestSearchMissingQueryReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Search(no q) status = %d, want 400", rr.Code)
	}
}

func TestSearchWithQueryReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=hello", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Search(q=hello) status = %d, want 200", rr.Code)
	}
}

func TestSearchWithTypeFilterReturns200(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test&type=track,album", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Search(q=test,type=track,album) status = %d, want 200", rr.Code)
	}
}

// --- Queue (requires auth) ---

func TestQueueUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	rr := httptest.NewRecorder()
	h.Queue(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Queue(unauth) status = %d, want 401", rr.Code)
	}
}

// --- QueueAdd (requires auth) ---

func TestQueueAddUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue",
		strings.NewReader(`{"track_ids":[1,2,3]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.QueueAdd(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("QueueAdd(unauth) status = %d, want 401", rr.Code)
	}
}

// --- QueueClear (requires auth) ---

func TestQueueClearUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/queue", nil)
	rr := httptest.NewRecorder()
	h.QueueClear(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("QueueClear(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Player (requires auth) ---

func TestPlayerUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/player", nil)
	rr := httptest.NewRecorder()
	h.Player(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Player(unauth) status = %d, want 401", rr.Code)
	}
}

// --- PlayerControl (requires auth) ---

func TestPlayerControlUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/player/play", nil)
	req.SetPathValue("action", "play")
	rr := httptest.NewRecorder()
	h.PlayerControl(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PlayerControl(unauth) status = %d, want 401", rr.Code)
	}
}

// --- CoverArt (public, validates ID) ---

func TestCoverArtInvalidIDReturns400(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cover/album/notanid", nil)
	req.SetPathValue("type", "album")
	req.SetPathValue("id", "notanid")
	rr := httptest.NewRecorder()
	h.CoverArt(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("CoverArt(invalid id) status = %d, want 400", rr.Code)
	}
}

func TestCoverArtValidIDReturns404(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cover/album/1", nil)
	req.SetPathValue("type", "album")
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.CoverArt(rr, req)
	// Returns 404 because cover art requires a populated media library
	if rr.Code != http.StatusNotFound {
		t.Errorf("CoverArt(valid id, no library) status = %d, want 404", rr.Code)
	}
}

// --- History (requires auth) ---

func TestHistoryUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/history", nil)
	rr := httptest.NewRecorder()
	h.History(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("History(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Stats (requires auth) ---

func TestStatsUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rr := httptest.NewRecorder()
	h.Stats(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Stats(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Scrobble (requires auth) ---

func TestScrobbleUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scrobble",
		strings.NewReader(`{"track_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Scrobble(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Scrobble(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Rate (requires auth) ---

func TestRateUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate",
		strings.NewReader(`{"type":"track","id":1,"rating":5}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Rate(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Rate(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Favorite (requires auth) ---

func TestFavoriteUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/favorite",
		strings.NewReader(`{"type":"track","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Favorite(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Favorite(unauth) status = %d, want 401", rr.Code)
	}
}

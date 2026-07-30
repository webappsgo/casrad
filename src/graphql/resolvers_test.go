// Package graphql — Tests for Resolver methods and the resolve dispatch functions.
// Covers: every resolver function (Health, Me, Tracks, Track, Albums, Album,
// Artists, Artist, Playlists, Playlist, Broadcasts, Broadcast, Search, Login,
// Logout, CreatePlaylist, UpdatePlaylist, DeletePlaylist, AddToPlaylist, UpdateProfile)
// via direct calls AND via the HTTP Handler (which exercises resolveQueryField and
// resolveMutationField and executeMutation code paths).
package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/store"
)

// --- helpers ---

// newResolverWithStore returns a Resolver backed by an empty MemoryStore.
func newResolverWithStore() *Resolver {
	return NewResolver(store.NewMemoryStore())
}

// ctxWithUser returns a context carrying the given user ID, satisfying
// middleware.GetUserID so that auth-gated resolvers allow access.
func ctxWithUser(userID int64) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID)
}

// --- Health ---

func TestHealthReturnsStatus(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Health() returned nil map")
	}
	if _, ok := result["status"]; !ok {
		t.Errorf("Health() result missing 'status' key: %v", result)
	}
	if result["status"] != "ok" {
		t.Errorf("Health() status = %v, want ok", result["status"])
	}
}

// --- Me ---

func TestMeUnauthenticatedReturnsNilNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Me(context.Background())
	if err != nil {
		t.Fatalf("Me(unauthenticated) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Me(unauthenticated) = %v, want nil", result)
	}
}

func TestMeAuthenticatedUnknownUserReturnsNilNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	// User ID 99 does not exist in the empty store — store returns nil, nil
	ctx := ctxWithUser(99)
	result, err := r.Me(ctx)
	// MemoryStore.GetUserByID returns nil, nil for missing IDs → resolver returns nil, nil
	if err != nil {
		t.Fatalf("Me(missing user) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Me(missing user) = %v, want nil", result)
	}
}

// --- Tracks ---

func TestTracksEmptyListNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Tracks(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("Tracks() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Tracks() returned nil")
	}
	nodes, ok := result["nodes"]
	if !ok {
		t.Errorf("Tracks() missing 'nodes' key: %v", result)
	}
	if nodes == nil {
		t.Error("Tracks() nodes is nil")
	}
}

func TestTracksWithOffsetAndLimit(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Tracks(context.Background(), 10, 5)
	if err != nil {
		t.Fatalf("Tracks(offset=10,limit=5) unexpected error: %v", err)
	}
}

// --- Track ---

func TestTrackEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Track(context.Background(), "")
	if err == nil {
		t.Error("Track(\"\") should return error for empty ID")
	}
}

func TestTrackNonExistentIDReturnsNilNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Track(context.Background(), "999")
	if err != nil {
		t.Fatalf("Track(999) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Track(999) = %v, want nil", result)
	}
}

// --- Albums ---

func TestAlbumsEmptyListNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Albums(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("Albums() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Albums() returned nil")
	}
	if _, ok := result["nodes"]; !ok {
		t.Error("Albums() missing 'nodes' key")
	}
}

// --- Album ---

func TestAlbumEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Album(context.Background(), "")
	if err == nil {
		t.Error("Album(\"\") should return error")
	}
}

func TestAlbumNonExistentIDReturnsNil(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Album(context.Background(), "999")
	if err != nil {
		t.Fatalf("Album(999) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Album(999) = %v, want nil", result)
	}
}

// --- Artists ---

func TestArtistsEmptyListNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Artists(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("Artists() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Artists() returned nil")
	}
	if _, ok := result["nodes"]; !ok {
		t.Error("Artists() missing 'nodes' key")
	}
}

// --- Artist ---

func TestArtistEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Artist(context.Background(), "")
	if err == nil {
		t.Error("Artist(\"\") should return error")
	}
}

func TestArtistNonExistentIDReturnsNil(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Artist(context.Background(), "999")
	if err != nil {
		t.Fatalf("Artist(999) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Artist(999) = %v, want nil", result)
	}
}

// --- Playlists ---

func TestPlaylistsUnauthenticatedReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Playlists(context.Background())
	if err != nil {
		t.Fatalf("Playlists(unauthenticated) unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Playlists() should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("Playlists(unauthenticated) len = %d, want 0", len(result))
	}
}

func TestPlaylistsAuthenticatedReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Playlists(ctxWithUser(1))
	if err != nil {
		t.Fatalf("Playlists(authenticated) unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Playlists() should return non-nil slice")
	}
}

// --- Playlist ---

func TestPlaylistEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Playlist(context.Background(), "")
	if err == nil {
		t.Error("Playlist(\"\") should return error")
	}
}

func TestPlaylistNonExistentIDReturnsNil(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Playlist(context.Background(), "999")
	if err != nil {
		t.Fatalf("Playlist(999) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Playlist(999) = %v, want nil", result)
	}
}

// --- Broadcasts ---

func TestBroadcastsReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Broadcasts(context.Background())
	if err != nil {
		t.Fatalf("Broadcasts() unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Broadcasts() should return non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("Broadcasts() len = %d, want 0", len(result))
	}
}

// --- Broadcast ---

func TestBroadcastEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Broadcast(context.Background(), "")
	if err == nil {
		t.Error("Broadcast(\"\") should return error")
	}
}

func TestBroadcastNonExistentIDReturnsNil(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Broadcast(context.Background(), "999")
	if err != nil {
		t.Fatalf("Broadcast(999) unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Broadcast(999) = %v, want nil", result)
	}
}

// --- Search ---

func TestSearchWithQueryReturnsMap(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search(test) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Search() returned nil")
	}
	for _, key := range []string{"tracks", "albums", "artists"} {
		if _, ok := result[key]; !ok {
			t.Errorf("Search() result missing %q key: %v", key, result)
		}
	}
}

func TestSearchEmptyQueryReturnsEmptyResults(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	result, err := r.Search(context.Background(), "")
	if err != nil {
		t.Fatalf("Search(\"\") unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Search(\"\") returned nil")
	}
}

// --- Login ---

func TestLoginEmptyIdentifierReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Login(context.Background(), "", "pass")
	if err == nil {
		t.Error("Login(\"\", pass) should return error")
	}
}

func TestLoginEmptyPasswordReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.Login(context.Background(), "user", "")
	if err == nil {
		t.Error("Login(user, \"\") should return error")
	}
}

func TestLoginInvalidCredentialsReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	// No users in store → lookup fails → "invalid credentials"
	_, err := r.Login(context.Background(), "nonexistent", "wrongpass")
	if err == nil {
		t.Error("Login(nonexistent, wrongpass) should return error")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("Login() error = %q, want 'invalid credentials'", err.Error())
	}
}

// --- Logout ---

func TestLogoutReturnsTrueNoError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ok, err := r.Logout(context.Background())
	if err != nil {
		t.Fatalf("Logout() unexpected error: %v", err)
	}
	if !ok {
		t.Error("Logout() = false, want true")
	}
}

// --- CreatePlaylist ---

func TestCreatePlaylistUnauthenticatedReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.CreatePlaylist(context.Background(), map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("CreatePlaylist(unauthenticated) should return error")
	}
}

func TestCreatePlaylistEmptyNameReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	_, err := r.CreatePlaylist(ctx, map[string]interface{}{"name": ""})
	if err == nil {
		t.Error("CreatePlaylist(empty name) should return error")
	}
}

func TestCreatePlaylistAuthenticatedReturnsPlaylist(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	result, err := r.CreatePlaylist(ctx, map[string]interface{}{
		"name":        "My Playlist",
		"description": "A test playlist",
		"isPublic":    true,
	})
	if err != nil {
		t.Fatalf("CreatePlaylist(authenticated) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("CreatePlaylist() returned nil")
	}
	if result["name"] != "My Playlist" {
		t.Errorf("CreatePlaylist() name = %v, want 'My Playlist'", result["name"])
	}
}

// --- UpdatePlaylist ---

func TestUpdatePlaylistUnauthenticatedReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.UpdatePlaylist(context.Background(), "1", map[string]interface{}{})
	if err == nil {
		t.Error("UpdatePlaylist(unauthenticated) should return error")
	}
}

func TestUpdatePlaylistEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	_, err := r.UpdatePlaylist(ctx, "", map[string]interface{}{})
	if err == nil {
		t.Error("UpdatePlaylist(\"\") should return error")
	}
}

func TestUpdatePlaylistAuthenticatedReturnsResult(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	result, err := r.UpdatePlaylist(ctx, "999", map[string]interface{}{"name": "Updated"})
	if err != nil {
		t.Fatalf("UpdatePlaylist(authenticated) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("UpdatePlaylist() returned nil")
	}
	if result["id"] != "999" {
		t.Errorf("UpdatePlaylist() id = %v, want '999'", result["id"])
	}
}

// --- DeletePlaylist ---

func TestDeletePlaylistUnauthenticatedReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.DeletePlaylist(context.Background(), "1")
	if err == nil {
		t.Error("DeletePlaylist(unauthenticated) should return error")
	}
}

func TestDeletePlaylistEmptyIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	_, err := r.DeletePlaylist(ctx, "")
	if err == nil {
		t.Error("DeletePlaylist(\"\") should return error")
	}
}

func TestDeletePlaylistAuthenticatedReturnsTrue(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	ok, err := r.DeletePlaylist(ctx, "999")
	if err != nil {
		t.Fatalf("DeletePlaylist(authenticated) unexpected error: %v", err)
	}
	if !ok {
		t.Error("DeletePlaylist() = false, want true")
	}
}

// --- AddToPlaylist ---

func TestAddToPlaylistUnauthenticatedReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.AddToPlaylist(context.Background(), "1", []string{"a"})
	if err == nil {
		t.Error("AddToPlaylist(unauthenticated) should return error")
	}
}

func TestAddToPlaylistEmptyPlaylistIDReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	_, err := r.AddToPlaylist(ctx, "", []string{"track1"})
	if err == nil {
		t.Error("AddToPlaylist(empty playlistID) should return error")
	}
}

func TestAddToPlaylistAuthenticatedReturnsResult(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	ctx := ctxWithUser(1)
	result, err := r.AddToPlaylist(ctx, "42", []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("AddToPlaylist(authenticated) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("AddToPlaylist() returned nil")
	}
	if result["trackCount"] != 2 {
		t.Errorf("AddToPlaylist() trackCount = %v, want 2", result["trackCount"])
	}
}

// --- UpdateProfile ---

func TestUpdateProfileUnauthenticatedReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	_, err := r.UpdateProfile(context.Background(), map[string]interface{}{"email": "new@example.com"})
	if err == nil {
		t.Error("UpdateProfile(unauthenticated) should return error")
	}
}

func TestUpdateProfileMissingUserReturnsError(t *testing.T) {
	t.Parallel()
	r := newResolverWithStore()
	// User ID 999 does not exist — GetUserByID returns nil, nil
	// The resolver must return an error, not panic.
	ctx := ctxWithUser(999)
	_, err := r.UpdateProfile(ctx, map[string]interface{}{"email": "x@example.com"})
	if err == nil {
		t.Error("UpdateProfile(missing user) should return error, not nil")
	}
}

// --- HTTP path: resolveQueryField via Handler ---
// These tests exercise the resolveQueryField switch branches that were 0% covered.

func TestHTTPHandlerQueryHealth(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={health{status}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query health status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("query health body = %q, want 'ok'", rr.Body.String())
	}
}

func TestHTTPHandlerQueryMe(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={me{username}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query me status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryTracks(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={tracks{nodes{title}}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query tracks status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryTrackByID(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={track(id:"1"){title}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query track status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryAlbums(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={albums{nodes{title}}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query albums status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryAlbumByID(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={album(id:"1"){title}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query album status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryArtists(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={artists{nodes{name}}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query artists status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryArtistByID(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={artist(id:"1"){name}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query artist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryPlaylists(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={playlists{name}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query playlists status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryPlaylistByID(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={playlist(id:"1"){name}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query playlist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryBroadcasts(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={broadcasts{name}}", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query broadcasts status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQueryBroadcastByID(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={broadcast(id:"1"){name}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query broadcast status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerQuerySearch(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, `/graphql?query={search(query:"rock"){tracks}}`, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("query search status = %d, want 200", rr.Code)
	}
}

// --- HTTP path: resolveMutationField + executeMutation ---

func TestHTTPHandlerMutationLogout(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { logout }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation logout status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationLoginInvalidCreds(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { login(identifier: \"nobody\", password: \"wrong\") { token } }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// login fails with invalid credentials → returned as a GraphQL error in data, still 200
	if rr.Code != http.StatusOK {
		t.Errorf("mutation login(invalid) status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationCreatePlaylistUnauthenticated(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { createPlaylist(name: \"Test\") { id } }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation createPlaylist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationUpdatePlaylistUnauthenticated(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { updatePlaylist(id: \"1\") { id } }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation updatePlaylist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationDeletePlaylistUnauthenticated(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { deletePlaylist(id: \"1\") }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation deletePlaylist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationAddToPlaylistUnauthenticated(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { addToPlaylist(playlistId: \"1\", trackIds: [\"t1\"]) { id } }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation addToPlaylist status = %d, want 200", rr.Code)
	}
}

func TestHTTPHandlerMutationUpdateProfileUnauthenticated(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { updateProfile(input: {email: \"x@x.com\"}) { id } }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation updateProfile status = %d, want 200", rr.Code)
	}
}

// TestHTTPHandlerMutationUnknownField exercises the default branch in resolveMutationField.
func TestHTTPHandlerMutationUnknownField(t *testing.T) {
	t.Parallel()
	h := Handler(store.NewMemoryStore())
	body := strings.NewReader(`{"query":"mutation { doSomethingUnknown }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("mutation unknown field status = %d, want 200", rr.Code)
	}
}

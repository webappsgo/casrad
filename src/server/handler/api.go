// Package handler - API handlers
// See AI.md PART 14 for API structure - all routes versioned at /api/v1/*
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/store"
)

// APIResponse is the standard API response wrapper
// See AI.md PART 9 for response format specification
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total"`
	Page   int         `json:"page"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// Standard error codes per AI.md PART 9
const (
	ErrBadRequest       = "BAD_REQUEST"
	ErrValidationFailed = "VALIDATION_FAILED"
	ErrUnauthorized     = "UNAUTHORIZED"
	ErrTokenExpired     = "TOKEN_EXPIRED"
	ErrTokenInvalid     = "TOKEN_INVALID"
	Err2FARequired      = "2FA_REQUIRED"
	Err2FAInvalid       = "2FA_INVALID"
	ErrForbidden        = "FORBIDDEN"
	ErrAccountLocked    = "ACCOUNT_LOCKED"
	ErrNotFound         = "NOT_FOUND"
	ErrMethodNotAllowed = "METHOD_NOT_ALLOWED"
	ErrConflict         = "CONFLICT"
	ErrRateLimited      = "RATE_LIMITED"
	ErrServerError      = "SERVER_ERROR"
	ErrMaintenance      = "MAINTENANCE"
)

// errorCodeToHTTP maps error code to HTTP status
// See AI.md PART 9 for error code specification
func errorCodeToHTTP(code string) int {
	switch code {
	case ErrBadRequest, ErrValidationFailed:
		return 400
	case ErrUnauthorized, ErrTokenExpired, ErrTokenInvalid, Err2FARequired, Err2FAInvalid:
		return 401
	case ErrForbidden, ErrAccountLocked:
		return 403
	case ErrNotFound:
		return 404
	case ErrMethodNotAllowed:
		return 405
	case ErrConflict:
		return 409
	case ErrRateLimited:
		return 429
	case ErrMaintenance:
		return 503
	default:
		return 500
	}
}

// SendOK sends a success response with data
// See AI.md PART 9 for response format specification
func SendOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(APIResponse{OK: true, Data: data})
}

// SendError sends an error response with proper HTTP status
// See AI.md PART 9 for error code specification
func SendError(w http.ResponseWriter, code string, message string) {
	status := errorCodeToHTTP(code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(APIResponse{OK: false, Error: code, Message: message})
}

// SendMessage sends a success response with just a message
func SendMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(APIResponse{OK: true, Message: message})
}

// SendCreated sends a 201 Created response with data
func SendCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(APIResponse{OK: true, Data: data})
}

// ValidationErrorResponse wraps validation errors
type ValidationErrorResponse struct {
	OK     bool              `json:"ok"`
	Error  string            `json:"error"`
	Errors map[string]string `json:"errors"`
}

// SendValidationErrors sends a 400 response with field-level validation errors
func SendValidationErrors(w http.ResponseWriter, errors map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(ValidationErrorResponse{
		OK:     false,
		Error:  ErrValidationFailed,
		Errors: errors,
	})
}

// APIHandler handles all API endpoints
type APIHandler struct {
	store store.Store
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(s store.Store) *APIHandler {
	return &APIHandler{
		store: s,
	}
}

// writeJSON writes a JSON response with proper formatting
// See AI.md PART 14 - JSON must be indented with single trailing newline
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}

// getPagination extracts pagination parameters from request
func getPagination(r *http.Request) (page, limit, offset int) {
	page = 1
	limit = 50

	if p := r.URL.Query().Get("page"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
			limit = lInt
		}
	}

	offset = (page - 1) * limit
	return
}

// Tracks handles GET /api/v1/tracks - List tracks
func (h *APIHandler) Tracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	page, limit, offset := getPagination(r)

	// Get filter parameters
	artist := r.URL.Query().Get("artist")
	album := r.URL.Query().Get("album")
	genre := r.URL.Query().Get("genre")

	// Mock tracks - in production would query from store
	tracks := []map[string]interface{}{}

	// Filter would be applied here
	_ = artist
	_ = album
	_ = genre
	_ = userID
	_ = offset

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   tracks,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Track handles GET /api/v1/tracks/{id} - Get track details
func (h *APIHandler) Track(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid track ID")
		return
	}

	// Mock track - in production would query from store
	track := map[string]interface{}{
		"id":       id,
		"title":    "Sample Track",
		"artist":   "Sample Artist",
		"album":    "Sample Album",
		"duration": 180000,
		"bitrate":  320,
		"format":   "mp3",
	}

	SendOK(w, track)
}

// TrackStream handles GET /api/v1/tracks/{id}/stream - Stream track audio
func (h *APIHandler) TrackStream(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid track ID", http.StatusBadRequest)
		return
	}

	// Get format preference
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "mp3"
	}

	// Get bitrate preference
	bitrate := 192
	if b := r.URL.Query().Get("bitrate"); b != "" {
		if bInt, err := strconv.Atoi(b); err == nil && bInt > 0 {
			bitrate = bInt
		}
	}

	// In production would:
	// 1. Get track from store
	// 2. Check user authorization
	// 3. Transcode if needed
	// 4. Stream audio with proper headers

	_ = id
	_ = bitrate

	// Set audio headers
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-cache")

	// For now, return empty audio
	w.WriteHeader(http.StatusOK)
}

// Albums handles GET /api/v1/albums - List albums
func (h *APIHandler) Albums(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := getPagination(r)

	// Get filter parameters
	artist := r.URL.Query().Get("artist")
	year := r.URL.Query().Get("year")
	_ = artist
	_ = year

	// Mock albums
	albums := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   albums,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Album handles GET /api/v1/albums/{id} - Get album details
func (h *APIHandler) Album(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid album ID")
		return
	}

	album := map[string]interface{}{
		"id":           id,
		"title":        "Sample Album",
		"artist":       "Sample Artist",
		"year":         2024,
		"track_count":  12,
		"duration":     3600000,
		"cover_art_id": id,
	}

	SendOK(w, album)
}

// Artists handles GET /api/v1/artists - List artists
func (h *APIHandler) Artists(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := getPagination(r)

	artists := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   artists,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Artist handles GET /api/v1/artists/{id} - Get artist details
func (h *APIHandler) Artist(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid artist ID")
		return
	}

	artist := map[string]interface{}{
		"id":          id,
		"name":        "Sample Artist",
		"album_count": 5,
		"track_count": 50,
	}

	SendOK(w, artist)
}

// Playlists handles GET /api/v1/playlists - User's playlists
func (h *APIHandler) Playlists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	page, limit, offset := getPagination(r)

	playlists := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   playlists,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Playlist handles GET /api/v1/playlists/{id} - Get playlist details
func (h *APIHandler) Playlist(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid playlist ID")
		return
	}

	playlist := map[string]interface{}{
		"id":          id,
		"name":        "My Playlist",
		"track_count": 25,
		"duration":    4500000,
		"is_public":   false,
	}

	SendOK(w, playlist)
}

// PlaylistCreate handles POST /api/v1/playlists - Create playlist
func (h *APIHandler) PlaylistCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		SendError(w, ErrValidationFailed, "Name is required")
		return
	}

	// Create playlist in store
	// Mock ID
	playlistID := int64(1)

	SendCreated(w, map[string]interface{}{
		"id":   playlistID,
		"name": req.Name,
	})
}

// PlaylistUpdate handles PATCH /api/v1/playlists/{id} - Update playlist
func (h *APIHandler) PlaylistUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid playlist ID")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	_ = id

	SendMessage(w, "Playlist updated")
}

// PlaylistDelete handles DELETE /api/v1/playlists/{id} - Delete playlist
func (h *APIHandler) PlaylistDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid playlist ID")
		return
	}

	_ = id

	SendMessage(w, "Playlist deleted")
}

// PlaylistAddTracks handles POST /api/v1/playlists/{id}/tracks - Add tracks to playlist
func (h *APIHandler) PlaylistAddTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid playlist ID")
		return
	}

	var req struct {
		TrackIDs []int64 `json:"track_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	_ = id

	SendOK(w, map[string]interface{}{
		"message": "Tracks added",
		"added":   len(req.TrackIDs),
	})
}

// Broadcasts handles GET /api/v1/broadcasts - List mount points
func (h *APIHandler) Broadcasts(w http.ResponseWriter, r *http.Request) {
	broadcasts := []map[string]interface{}{
		{
			"mount":     "/live",
			"name":      "Live Stream",
			"listeners": 0,
			"is_active": false,
			"format":    "mp3",
			"bitrate":   128,
		},
	}

	SendOK(w, broadcasts)
}

// Broadcast handles GET /api/v1/broadcasts/{mount} - Get broadcast details
func (h *APIHandler) Broadcast(w http.ResponseWriter, r *http.Request) {
	mount := r.PathValue("mount")
	if mount == "" {
		SendError(w, ErrBadRequest, "Mount point required")
		return
	}

	broadcast := map[string]interface{}{
		"mount":          "/" + mount,
		"name":           "Live Stream",
		"description":    "Live broadcast",
		"listeners":      0,
		"peak_listeners": 0,
		"is_active":      false,
		"format":         "mp3",
		"bitrate":        128,
		"sample_rate":    44100,
	}

	SendOK(w, broadcast)
}

// Podcasts handles GET /api/v1/podcasts - User's subscriptions
func (h *APIHandler) Podcasts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	page, limit, offset := getPagination(r)

	podcasts := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   podcasts,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// PodcastSubscribe handles POST /api/v1/podcasts - Subscribe to podcast
func (h *APIHandler) PodcastSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		FeedURL string `json:"feed_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	if req.FeedURL == "" {
		SendError(w, ErrValidationFailed, "Feed URL is required")
		return
	}

	// Validate and subscribe to podcast
	// Mock ID
	podcastID := int64(1)

	SendCreated(w, map[string]interface{}{
		"id":       podcastID,
		"feed_url": req.FeedURL,
	})
}

// Audiobooks handles GET /api/v1/audiobooks - User's audiobooks
func (h *APIHandler) Audiobooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	page, limit, offset := getPagination(r)

	audiobooks := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   audiobooks,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Audiobook handles GET /api/v1/audiobooks/{id} - Get audiobook details
func (h *APIHandler) Audiobook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SendError(w, ErrBadRequest, "Invalid audiobook ID")
		return
	}

	audiobook := map[string]interface{}{
		"id":              id,
		"title":           "Sample Audiobook",
		"author":          "Sample Author",
		"narrator":        "Sample Narrator",
		"duration":        36000,
		"chapter_count":   20,
		"current_chapter": 0,
		"progress":        0,
	}

	SendOK(w, audiobook)
}

// Search handles GET /api/v1/search - Search library
func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		SendError(w, ErrValidationFailed, "Query parameter 'q' is required")
		return
	}

	// Get search types
	types := r.URL.Query().Get("type")
	if types == "" {
		types = "track,album,artist"
	}

	searchTypes := strings.Split(types, ",")

	results := map[string]interface{}{}

	for _, t := range searchTypes {
		switch strings.TrimSpace(t) {
		case "track":
			results["tracks"] = []map[string]interface{}{}
		case "album":
			results["albums"] = []map[string]interface{}{}
		case "artist":
			results["artists"] = []map[string]interface{}{}
		case "playlist":
			results["playlists"] = []map[string]interface{}{}
		case "podcast":
			results["podcasts"] = []map[string]interface{}{}
		}
	}

	SendOK(w, map[string]interface{}{
		"query":   query,
		"results": results,
	})
}

// Queue handles GET /api/v1/queue - Get user's playback queue
func (h *APIHandler) Queue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	queue := map[string]interface{}{
		"current_index": 0,
		"shuffle":       false,
		"repeat":        "off",
		"tracks":        []interface{}{},
	}

	SendOK(w, queue)
}

// QueueAdd handles POST /api/v1/queue - Add to queue (append by default)
func (h *APIHandler) QueueAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		TrackIDs []int64 `json:"track_ids"`
		// "next" or "end" (default)
		Position string `json:"position"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	if len(req.TrackIDs) == 0 {
		SendError(w, ErrValidationFailed, "Track IDs required")
		return
	}

	SendOK(w, map[string]interface{}{
		"message": "Tracks added to queue",
		"added":   len(req.TrackIDs),
	})
}

// QueueClear handles DELETE /api/v1/queue - Clear queue
func (h *APIHandler) QueueClear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	SendMessage(w, "Queue cleared")
}

// Player handles GET /api/v1/player - Get player state
func (h *APIHandler) Player(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	player := map[string]interface{}{
		"is_playing":    false,
		"current_track": nil,
		"position":      0,
		"volume":        70,
		"shuffle":       false,
		"repeat":        "off",
		"quality":       "auto",
	}

	SendOK(w, player)
}

// PlayerControl handles POST /api/v1/player/{action} - Control playback
func (h *APIHandler) PlayerControl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	action := r.PathValue("action")

	switch action {
	case "play":
		SendMessage(w, "Playing")
	case "pause":
		SendMessage(w, "Paused")
	case "stop":
		SendMessage(w, "Stopped")
	case "next":
		SendMessage(w, "Skipped to next")
	case "previous":
		SendMessage(w, "Skipped to previous")
	case "shuffle":
		SendMessage(w, "Shuffle toggled")
	case "repeat":
		SendMessage(w, "Repeat toggled")
	default:
		SendError(w, ErrBadRequest, "Unknown action")
	}
}

// CoverArt handles GET /api/v1/cover/{type}/{id} - Get cover art
func (h *APIHandler) CoverArt(w http.ResponseWriter, r *http.Request) {
	// album, artist, playlist
	artType := r.PathValue("type")
	idStr := r.PathValue("id")

	_, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	// Get size parameter
	size := r.URL.Query().Get("size")
	if size == "" {
		size = "300"
	}

	_ = artType

	// Cover art retrieval requires a loaded media library.
	// Return 404 until the library scanner populates cover art paths.
	http.NotFound(w, r)
}

// History handles GET /api/v1/history - Get listening history
func (h *APIHandler) History(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	page, limit, offset := getPagination(r)

	history := []map[string]interface{}{}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data:   history,
		Total:  0,
		Page:   page,
		Limit:  limit,
		Offset: offset,
	})
}

// Stats handles GET /api/v1/stats - Get user statistics
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	stats := map[string]interface{}{
		"total_tracks":     0,
		"total_albums":     0,
		"total_artists":    0,
		"total_playlists":  0,
		"total_play_count": 0,
		"total_play_time":  0,
		"storage_used":     0,
		"storage_quota":    53687091200,
	}

	SendOK(w, stats)
}

// Scrobble handles POST /api/v1/scrobble - Record play
func (h *APIHandler) Scrobble(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		TrackID   int64 `json:"track_id"`
		Timestamp int64 `json:"timestamp"`
		Duration  int   `json:"duration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	SendMessage(w, "Scrobble recorded")
}

// Rate handles POST /api/v1/rate - Rate track/album/artist
func (h *APIHandler) Rate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		// track, album, artist
		Type string `json:"type"`
		ID   int64  `json:"id"`
		// 0-5
		Rating int `json:"rating"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	if req.Rating < 0 || req.Rating > 5 {
		SendError(w, ErrValidationFailed, "Rating must be 0-5")
		return
	}

	SendMessage(w, "Rating saved")
}

// Favorite handles POST /api/v1/favorite - Toggle favorite
func (h *APIHandler) Favorite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	if userID == 0 {
		SendError(w, ErrUnauthorized, "Authentication required")
		return
	}

	var req struct {
		// track, album, artist
		Type string `json:"type"`
		ID   int64  `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest, "Invalid request body")
		return
	}

	SendOK(w, map[string]interface{}{
		"message":     "Favorite toggled",
		"is_favorite": true,
	})
}

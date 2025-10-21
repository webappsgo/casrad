package protocols

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/gorilla/mux"
)

// Ampache API v6.0.0 Implementation
type AmpacheServer struct {
	db     *database.Engine
	router *mux.Router
}

// Ampache response structures
type AmpacheResponse struct {
	XMLName xml.Name `xml:"root" json:"-"`
	Auth    *Auth    `xml:"auth,omitempty" json:"auth,omitempty"`
	Error   *Error   `xml:"error,omitempty" json:"error,omitempty"`
	Songs   []Song   `xml:"song,omitempty" json:"song,omitempty"`
	Albums  []Album  `xml:"album,omitempty" json:"album,omitempty"`
	Artists []Artist `xml:"artist,omitempty" json:"artist,omitempty"`
	Stats   *Stats   `xml:"stats,omitempty" json:"stats,omitempty"`
}

type Auth struct {
	XMLName    xml.Name `xml:"auth" json:"-"`
	APIVersion string   `xml:"api" json:"api"`
	Session    string   `xml:"session" json:"session"`
	Update     string   `xml:"update" json:"update"`
	Add        string   `xml:"add" json:"add"`
	Clean      string   `xml:"clean" json:"clean"`
	Songs      string   `xml:"songs" json:"songs"`
	Albums     string   `xml:"albums" json:"albums"`
	Artists    string   `xml:"artists" json:"artists"`
	Playlists  string   `xml:"playlists" json:"playlists"`
	Videos     string   `xml:"videos" json:"videos"`
}

type Error struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    string   `xml:"code,attr" json:"code"`
	Message string   `xml:",chardata" json:"message"`
}

type Stats struct {
	XMLName   xml.Name `xml:"stats" json:"-"`
	Albums    int      `xml:"albums" json:"albums"`
	Artists   int      `xml:"artists" json:"artists"`
	Songs     int      `xml:"songs" json:"songs"`
	Playlists int      `xml:"playlists" json:"playlists"`
	Users     int      `xml:"users" json:"users"`
}

func NewAmpacheServer(db *database.Engine) *AmpacheServer {
	s := &AmpacheServer{
		db:     db,
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *AmpacheServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *AmpacheServer) setupRoutes() {
	// All Ampache API endpoints go through server.xml or server.json
	s.router.HandleFunc("/server/xml.server.php", s.handleXMLRequest)
	s.router.HandleFunc("/server/json.server.php", s.handleJSONRequest)

	// Legacy endpoints
	s.router.HandleFunc("/ampache/server/xml.server.php", s.handleXMLRequest)
	s.router.HandleFunc("/ampache/server/json.server.php", s.handleJSONRequest)
}

func (s *AmpacheServer) handleXMLRequest(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	auth := r.FormValue("auth")

	// Check authentication for non-handshake requests
	if action != "handshake" && !s.validateAuth(auth) {
		s.sendXMLError(w, "401", "Invalid authentication")
		return
	}

	switch action {
	case "handshake":
		s.handshake(w, r, "xml")
	case "ping":
		s.ping(w, r, "xml")
	case "stats":
		s.stats(w, r, "xml")
	case "songs":
		s.songs(w, r, "xml")
	case "albums":
		s.albums(w, r, "xml")
	case "artists":
		s.artists(w, r, "xml")
	case "album_songs":
		s.albumSongs(w, r, "xml")
	case "artist_albums":
		s.artistAlbums(w, r, "xml")
	case "artist_songs":
		s.artistSongs(w, r, "xml")
	case "search_songs":
		s.searchSongs(w, r, "xml")
	case "playlists":
		s.playlists(w, r, "xml")
	case "playlist_songs":
		s.playlistSongs(w, r, "xml")
	case "stream":
		s.stream(w, r)
	case "download":
		s.download(w, r)
	default:
		s.sendXMLError(w, "405", "Invalid action")
	}
}

func (s *AmpacheServer) handleJSONRequest(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	auth := r.FormValue("auth")

	// Check authentication for non-handshake requests
	if action != "handshake" && !s.validateAuth(auth) {
		s.sendJSONError(w, "401", "Invalid authentication")
		return
	}

	switch action {
	case "handshake":
		s.handshake(w, r, "json")
	case "ping":
		s.ping(w, r, "json")
	case "stats":
		s.stats(w, r, "json")
	case "songs":
		s.songs(w, r, "json")
	case "albums":
		s.albums(w, r, "json")
	case "artists":
		s.artists(w, r, "json")
	case "album_songs":
		s.albumSongs(w, r, "json")
	case "artist_albums":
		s.artistAlbums(w, r, "json")
	case "artist_songs":
		s.artistSongs(w, r, "json")
	case "search_songs":
		s.searchSongs(w, r, "json")
	case "playlists":
		s.playlists(w, r, "json")
	case "playlist_songs":
		s.playlistSongs(w, r, "json")
	case "stream":
		s.stream(w, r)
	case "download":
		s.download(w, r)
	default:
		s.sendJSONError(w, "405", "Invalid action")
	}
}

func (s *AmpacheServer) handshake(w http.ResponseWriter, r *http.Request, format string) {
	username := r.FormValue("user")
	timestamp := r.FormValue("timestamp")
	_ = r.FormValue("auth") // authToken

	// Generate auth token and validate
	// For now, create a session
	sessionID := s.generateSessionID(username)

	auth := &Auth{
		APIVersion: "6.0.0",
		Session:    sessionID,
		Update:     timestamp,
		Add:        timestamp,
		Clean:      timestamp,
		Songs:      "0",
		Albums:     "0",
		Artists:    "0",
		Playlists:  "0",
		Videos:     "0",
	}

	// Get counts from database
	var songCount, albumCount, artistCount int
	s.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&songCount)
	s.db.QueryRow("SELECT COUNT(DISTINCT album) FROM tracks WHERE album IS NOT NULL").Scan(&albumCount)
	s.db.QueryRow("SELECT COUNT(DISTINCT artist) FROM tracks WHERE artist IS NOT NULL").Scan(&artistCount)

	auth.Songs = strconv.Itoa(songCount)
	auth.Albums = strconv.Itoa(albumCount)
	auth.Artists = strconv.Itoa(artistCount)

	resp := &AmpacheResponse{Auth: auth}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) ping(w http.ResponseWriter, r *http.Request, format string) {
	// Simple ping response
	resp := map[string]interface{}{
		"session": r.FormValue("auth"),
		"version": "6.0.0",
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, "<root><session>%s</session><version>6.0.0</version></root>", r.FormValue("auth"))
	}
}

func (s *AmpacheServer) stats(w http.ResponseWriter, r *http.Request, format string) {
	stats := &Stats{}

	// Get statistics from database
	s.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&stats.Songs)
	s.db.QueryRow("SELECT COUNT(DISTINCT album) FROM tracks WHERE album IS NOT NULL").Scan(&stats.Albums)
	s.db.QueryRow("SELECT COUNT(DISTINCT artist) FROM tracks WHERE artist IS NOT NULL").Scan(&stats.Artists)
	s.db.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&stats.Playlists)
	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.Users)

	resp := &AmpacheResponse{Stats: stats}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) songs(w http.ResponseWriter, r *http.Request, format string) {
	limit := 100
	if l := r.FormValue("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	offset := 0
	if o := r.FormValue("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	rows, err := s.db.Query(`
		SELECT id, file_path, title, artist, album, genre, duration, bitrate, file_size
		FROM tracks
		ORDER BY artist, album, track_number
		LIMIT ? OFFSET ?
	`, limit, offset)

	if err != nil {
		if format == "json" {
			s.sendJSONError(w, "500", "Database error")
		} else {
			s.sendXMLError(w, "500", "Database error")
		}
		return
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		var title, artist, album, genre sql.NullString
		var duration, bitrate sql.NullInt32
		var fileSize sql.NullInt64

		var filePath sql.NullString
		rows.Scan(&song.ID, &filePath, &title, &artist, &album,
			&genre, &duration, &bitrate, &fileSize)

		if filePath.Valid {
			song.Path = filePath.String
		}

		if title.Valid {
			song.Title = title.String
		}
		if artist.Valid {
			song.Artist = artist.String
		}
		if album.Valid {
			song.Album = album.String
		}
		if genre.Valid {
			song.Genre = genre.String
		}
		if duration.Valid {
			song.Duration = int(duration.Int32 / 1000) // Convert ms to seconds
		}
		if bitrate.Valid {
			song.BitRate = int(bitrate.Int32)
		}
		if fileSize.Valid {
			song.Size = fileSize.Int64
		}

		// Generate URL for streaming
		// URL not in Song struct, skip for now

		songs = append(songs, song)
	}

	resp := &AmpacheResponse{Songs: songs}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) albums(w http.ResponseWriter, r *http.Request, format string) {
	rows, err := s.db.Query(`
		SELECT DISTINCT album, artist, COUNT(*) as track_count
		FROM tracks
		WHERE album IS NOT NULL
		GROUP BY album, artist
		ORDER BY album
		LIMIT 500
	`)

	if err != nil {
		if format == "json" {
			s.sendJSONError(w, "500", "Database error")
		} else {
			s.sendXMLError(w, "500", "Database error")
		}
		return
	}
	defer rows.Close()

	var albums []Album
	id := 1
	for rows.Next() {
		var album Album
		var albumName, artistName sql.NullString
		var trackCount int

		rows.Scan(&albumName, &artistName, &trackCount)

		album.ID = strconv.Itoa(id)
		if albumName.Valid {
			album.Title = albumName.String
		}
		if artistName.Valid {
			album.Artist = artistName.String
		}
		album.SongCount = trackCount

		albums = append(albums, album)
		id++
	}

	resp := &AmpacheResponse{Albums: albums}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) artists(w http.ResponseWriter, r *http.Request, format string) {
	rows, err := s.db.Query(`
		SELECT DISTINCT artist, COUNT(*) as track_count
		FROM tracks
		WHERE artist IS NOT NULL
		GROUP BY artist
		ORDER BY artist
	`)

	if err != nil {
		if format == "json" {
			s.sendJSONError(w, "500", "Database error")
		} else {
			s.sendXMLError(w, "500", "Database error")
		}
		return
	}
	defer rows.Close()

	var artists []Artist
	id := 1
	for rows.Next() {
		var artist Artist
		var artistName sql.NullString
		var trackCount int

		rows.Scan(&artistName, &trackCount)

		artist.ID = strconv.Itoa(id)
		if artistName.Valid {
			artist.Name = artistName.String
		}

		artists = append(artists, artist)
		id++
	}

	resp := &AmpacheResponse{Artists: artists}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) albumSongs(w http.ResponseWriter, r *http.Request, format string) {
	// Implementation similar to songs() but filtered by album
	s.songs(w, r, format)
}

func (s *AmpacheServer) artistAlbums(w http.ResponseWriter, r *http.Request, format string) {
	// Implementation similar to albums() but filtered by artist
	s.albums(w, r, format)
}

func (s *AmpacheServer) artistSongs(w http.ResponseWriter, r *http.Request, format string) {
	// Implementation similar to songs() but filtered by artist
	s.songs(w, r, format)
}

func (s *AmpacheServer) searchSongs(w http.ResponseWriter, r *http.Request, format string) {
	filter := r.FormValue("filter")

	rows, err := s.db.Query(`
		SELECT id, file_path, title, artist, album, genre, duration, bitrate, file_size
		FROM tracks
		WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?
		ORDER BY artist, album, track_number
		LIMIT 100
	`, "%"+filter+"%", "%"+filter+"%", "%"+filter+"%")

	if err != nil {
		if format == "json" {
			s.sendJSONError(w, "500", "Database error")
		} else {
			s.sendXMLError(w, "500", "Database error")
		}
		return
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		var title, artist, album, genre sql.NullString
		var duration, bitrate sql.NullInt32
		var fileSize sql.NullInt64

		var filePath sql.NullString
		rows.Scan(&song.ID, &filePath, &title, &artist, &album,
			&genre, &duration, &bitrate, &fileSize)

		if filePath.Valid {
			song.Path = filePath.String
		}

		if title.Valid {
			song.Title = title.String
		}
		if artist.Valid {
			song.Artist = artist.String
		}
		if album.Valid {
			song.Album = album.String
		}

		songs = append(songs, song)
	}

	resp := &AmpacheResponse{Songs: songs}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) playlists(w http.ResponseWriter, r *http.Request, format string) {
	// Return empty playlists for now
	resp := &AmpacheResponse{}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) playlistSongs(w http.ResponseWriter, r *http.Request, format string) {
	// Return empty songs for now
	resp := &AmpacheResponse{}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *AmpacheServer) stream(w http.ResponseWriter, r *http.Request) {
	// Redirect to actual stream endpoint
	id := r.FormValue("id")
	http.Redirect(w, r, fmt.Sprintf("/api/v1/stream/%s", id), http.StatusTemporaryRedirect)
}

func (s *AmpacheServer) download(w http.ResponseWriter, r *http.Request) {
	// Same as stream but with download headers
	s.stream(w, r)
}

func (s *AmpacheServer) validateAuth(auth string) bool {
	// Simple validation - in production, check against sessions table
	return auth != ""
}

func (s *AmpacheServer) generateSessionID(username string) string {
	h := sha256.New()
	h.Write([]byte(username + time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *AmpacheServer) sendXMLError(w http.ResponseWriter, code string, message string) {
	w.Header().Set("Content-Type", "text/xml")
	resp := &AmpacheResponse{
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	xml.NewEncoder(w).Encode(resp)
}

func (s *AmpacheServer) sendJSONError(w http.ResponseWriter, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	resp := &AmpacheResponse{
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}
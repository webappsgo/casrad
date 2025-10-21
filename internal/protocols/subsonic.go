package protocols

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/gorilla/mux"
)

// Subsonic API v1.16.1 Implementation
type SubsonicServer struct {
	db     *database.Engine
	router *mux.Router
}

// Subsonic response wrapper
type SubsonicResponse struct {
	XMLName xml.Name `xml:"subsonic-response" json:"-"`
	Status  string   `xml:"status,attr" json:"status"`
	Version string   `xml:"version,attr" json:"version"`
	Type    string   `xml:"type,attr" json:"type"`
	ServerVersion string `xml:"serverVersion,attr" json:"serverVersion"`

	// Error response
	Error *SubsonicError `xml:"error,omitempty" json:"error,omitempty"`

	// Response types
	License         *License         `xml:"license,omitempty" json:"license,omitempty"`
	MusicFolders    *MusicFolders    `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes         *Indexes         `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Directory       *Directory       `xml:"directory,omitempty" json:"directory,omitempty"`
	Artists         *ArtistList      `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist          *Artist          `xml:"artist,omitempty" json:"artist,omitempty"`
	Album           *Album           `xml:"album,omitempty" json:"album,omitempty"`
	Song            *Song            `xml:"song,omitempty" json:"song,omitempty"`
	NowPlaying      *NowPlaying      `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	SearchResult3   *SearchResult3   `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Playlists       *Playlists       `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist        *Playlist        `xml:"playlist,omitempty" json:"playlist,omitempty"`
	User            *User            `xml:"user,omitempty" json:"user,omitempty"`
	Users           *Users           `xml:"users,omitempty" json:"users,omitempty"`
	AlbumList2      *AlbumList2      `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	RandomSongs     *RandomSongs     `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	ScanStatus      *ScanStatus      `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	Starred2        *Starred2        `xml:"starred2,omitempty" json:"starred2,omitempty"`
	Podcasts        *Podcasts        `xml:"podcasts,omitempty" json:"podcasts,omitempty"`
	InternetRadioStations *InternetRadioStations `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`
}

type SubsonicError struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type License struct {
	Valid bool   `xml:"valid,attr" json:"valid"`
	Email string `xml:"email,attr" json:"email"`
	LicenseExpires string `xml:"licenseExpires,attr" json:"licenseExpires"`
}

type MusicFolders struct {
	Folders []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

type MusicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type Indexes struct {
	LastModified int64   `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index        []Index `xml:"index" json:"index"`
}

type Index struct {
	Name   string   `xml:"name,attr" json:"name"`
	Artist []Artist `xml:"artist" json:"artist"`
}

type ArtistList struct {
	IgnoredArticles string  `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index          []Index `xml:"index" json:"index"`
}

type Artist struct {
	ID         string  `xml:"id,attr" json:"id"`
	Name       string  `xml:"name,attr" json:"name"`
	CoverArt   string  `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int     `xml:"albumCount,attr" json:"albumCount"`
	Albums     []Album `xml:"album,omitempty" json:"album,omitempty"`
}

type Album struct {
	ID        string `xml:"id,attr" json:"id"`
	Parent    string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Title     string `xml:"title,attr" json:"title"`
	Album     string `xml:"album,attr" json:"album"`
	Artist    string `xml:"artist,attr" json:"artist"`
	IsDir     bool   `xml:"isDir,attr" json:"isDir"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Created   string `xml:"created,attr" json:"created"`
	Duration  int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	Year      int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre     string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	SongCount int    `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	Songs     []Song `xml:"song,omitempty" json:"song,omitempty"`
}

type Directory struct {
	ID       string  `xml:"id,attr" json:"id"`
	Parent   string  `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Name     string  `xml:"name,attr" json:"name"`
	Starred  string  `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	Children []Child `xml:"child" json:"child"`
}

type Child struct {
	ID          string `xml:"id,attr" json:"id"`
	Parent      string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	Title       string `xml:"title,attr" json:"title"`
	Album       string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist      string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track       int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year        int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre       string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size        int64  `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Duration    int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate     int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"`
	Created     string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Type        string `xml:"type,attr,omitempty" json:"type,omitempty"`
}

type Song struct {
	ID          string `xml:"id,attr" json:"id"`
	Parent      string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Title       string `xml:"title,attr" json:"title"`
	Album       string `xml:"album,attr" json:"album"`
	Artist      string `xml:"artist,attr" json:"artist"`
	Track       int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year        int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre       string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size        int64  `xml:"size,attr" json:"size"`
	ContentType string `xml:"contentType,attr" json:"contentType"`
	Suffix      string `xml:"suffix,attr" json:"suffix"`
	Duration    int    `xml:"duration,attr" json:"duration"`
	BitRate     int    `xml:"bitRate,attr" json:"bitRate"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	Created     string `xml:"created,attr" json:"created"`
	AlbumId     string `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistId    string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Type        string `xml:"type,attr,omitempty" json:"type,omitempty"`
}

type NowPlaying struct {
	Entries []NowPlayingEntry `xml:"entry" json:"entry"`
}

type NowPlayingEntry struct {
	Song
	Username   string `xml:"username,attr" json:"username"`
	PlayerId   int    `xml:"playerId,attr" json:"playerId"`
	PlayerName string `xml:"playerName,attr,omitempty" json:"playerName,omitempty"`
	MinutesAgo int    `xml:"minutesAgo,attr" json:"minutesAgo"`
}

type SearchResult3 struct {
	Artists []Artist `xml:"artist" json:"artist"`
	Albums  []Album  `xml:"album" json:"album"`
	Songs   []Song   `xml:"song" json:"song"`
}

type Playlists struct {
	Playlist []Playlist `xml:"playlist" json:"playlist"`
}

type Playlist struct {
	ID        string `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	Comment   string `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner     string `xml:"owner,attr,omitempty" json:"owner,omitempty"`
	Public    bool   `xml:"public,attr" json:"public"`
	SongCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int    `xml:"duration,attr" json:"duration"`
	Created   string `xml:"created,attr" json:"created"`
	Changed   string `xml:"changed,attr" json:"changed"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Songs     []Song `xml:"entry,omitempty" json:"entry,omitempty"`
}

type User struct {
	Username            string `xml:"username,attr" json:"username"`
	Email              string `xml:"email,attr,omitempty" json:"email,omitempty"`
	ScrobblingEnabled  bool   `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	AdminRole          bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole       bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole       bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole         bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole       bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole       bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole        bool   `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole        bool   `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole         bool   `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole        bool   `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole          bool   `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool  `xml:"videoConversionRole,attr" json:"videoConversionRole"`
}

type Users struct {
	User []User `xml:"user" json:"user"`
}

type AlbumList2 struct {
	Albums []Album `xml:"album" json:"album"`
}

type RandomSongs struct {
	Songs []Song `xml:"song" json:"song"`
}

type ScanStatus struct {
	Scanning bool  `xml:"scanning,attr" json:"scanning"`
	Count    int64 `xml:"count,attr" json:"count"`
}

type Starred2 struct {
	Artists []Artist `xml:"artist" json:"artist"`
	Albums  []Album  `xml:"album" json:"album"`
	Songs   []Song   `xml:"song" json:"song"`
}

type Podcasts struct {
	Channels []PodcastChannel `xml:"channel" json:"channel"`
}

type PodcastChannel struct {
	ID          string `xml:"id,attr" json:"id"`
	URL         string `xml:"url,attr" json:"url"`
	Title       string `xml:"title,attr" json:"title"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Status      string `xml:"status,attr" json:"status"`
	Episodes    []PodcastEpisode `xml:"episode,omitempty" json:"episode,omitempty"`
}

type PodcastEpisode struct {
	ID          string `xml:"id,attr" json:"id"`
	ChannelId   string `xml:"channelId,attr" json:"channelId"`
	Title       string `xml:"title,attr" json:"title"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`
	PublishDate string `xml:"publishDate,attr" json:"publishDate"`
	Status      string `xml:"status,attr" json:"status"`
	StreamId    string `xml:"streamId,attr,omitempty" json:"streamId,omitempty"`
	ContentType string `xml:"contentType,attr" json:"contentType"`
	Size        int64  `xml:"size,attr" json:"size"`
	Duration    int    `xml:"duration,attr" json:"duration"`
}

type InternetRadioStations struct {
	Stations []InternetRadioStation `xml:"internetRadioStation" json:"internetRadioStation"`
}

type InternetRadioStation struct {
	ID          string `xml:"id,attr" json:"id"`
	Name        string `xml:"name,attr" json:"name"`
	StreamUrl   string `xml:"streamUrl,attr" json:"streamUrl"`
	HomePageUrl string `xml:"homePageUrl,attr,omitempty" json:"homePageUrl,omitempty"`
}

func NewSubsonicServer(db *database.Engine) *SubsonicServer {
	s := &SubsonicServer{
		db:     db,
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *SubsonicServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *SubsonicServer) setupRoutes() {
	// All Subsonic API endpoints
	api := s.router.PathPrefix("/rest").Subrouter()
	api.Use(s.authenticate)

	// System
	api.HandleFunc("/ping", s.ping).Methods("GET", "POST")
	api.HandleFunc("/getLicense", s.getLicense).Methods("GET", "POST")

	// Browsing
	api.HandleFunc("/getMusicFolders", s.getMusicFolders).Methods("GET", "POST")
	api.HandleFunc("/getIndexes", s.getIndexes).Methods("GET", "POST")
	api.HandleFunc("/getMusicDirectory", s.getMusicDirectory).Methods("GET", "POST")
	api.HandleFunc("/getGenres", s.getGenres).Methods("GET", "POST")
	api.HandleFunc("/getArtists", s.getArtists).Methods("GET", "POST")
	api.HandleFunc("/getArtist", s.getArtist).Methods("GET", "POST")
	api.HandleFunc("/getAlbum", s.getAlbum).Methods("GET", "POST")
	api.HandleFunc("/getSong", s.getSong).Methods("GET", "POST")

	// Album/song lists
	api.HandleFunc("/getAlbumList", s.getAlbumList).Methods("GET", "POST")
	api.HandleFunc("/getAlbumList2", s.getAlbumList2).Methods("GET", "POST")
	api.HandleFunc("/getRandomSongs", s.getRandomSongs).Methods("GET", "POST")
	api.HandleFunc("/getSongsByGenre", s.getSongsByGenre).Methods("GET", "POST")
	api.HandleFunc("/getNowPlaying", s.getNowPlaying).Methods("GET", "POST")
	api.HandleFunc("/getStarred", s.getStarred).Methods("GET", "POST")
	api.HandleFunc("/getStarred2", s.getStarred2).Methods("GET", "POST")

	// Searching
	api.HandleFunc("/search", s.search).Methods("GET", "POST")
	api.HandleFunc("/search2", s.search2).Methods("GET", "POST")
	api.HandleFunc("/search3", s.search3).Methods("GET", "POST")

	// Playlists
	api.HandleFunc("/getPlaylists", s.getPlaylists).Methods("GET", "POST")
	api.HandleFunc("/getPlaylist", s.getPlaylist).Methods("GET", "POST")
	api.HandleFunc("/createPlaylist", s.createPlaylist).Methods("GET", "POST")
	api.HandleFunc("/updatePlaylist", s.updatePlaylist).Methods("GET", "POST")
	api.HandleFunc("/deletePlaylist", s.deletePlaylist).Methods("GET", "POST")

	// Media retrieval
	api.HandleFunc("/stream", s.stream).Methods("GET", "POST")
	api.HandleFunc("/download", s.download).Methods("GET", "POST")
	api.HandleFunc("/getCoverArt", s.getCoverArt).Methods("GET", "POST")
	api.HandleFunc("/getLyrics", s.getLyrics).Methods("GET", "POST")
	api.HandleFunc("/getAvatar", s.getAvatar).Methods("GET", "POST")

	// Media annotation
	api.HandleFunc("/star", s.star).Methods("GET", "POST")
	api.HandleFunc("/unstar", s.unstar).Methods("GET", "POST")
	api.HandleFunc("/setRating", s.setRating).Methods("GET", "POST")
	api.HandleFunc("/scrobble", s.scrobble).Methods("GET", "POST")

	// User management
	api.HandleFunc("/getUser", s.getUser).Methods("GET", "POST")
	api.HandleFunc("/getUsers", s.getUsers).Methods("GET", "POST")
	api.HandleFunc("/createUser", s.createUser).Methods("GET", "POST")
	api.HandleFunc("/updateUser", s.updateUser).Methods("GET", "POST")
	api.HandleFunc("/deleteUser", s.deleteUser).Methods("GET", "POST")
	api.HandleFunc("/changePassword", s.changePassword).Methods("GET", "POST")

	// Media library scanning
	api.HandleFunc("/getScanStatus", s.getScanStatus).Methods("GET", "POST")
	api.HandleFunc("/startScan", s.startScan).Methods("GET", "POST")

	// Podcasts
	api.HandleFunc("/getPodcasts", s.getPodcasts).Methods("GET", "POST")
	api.HandleFunc("/getNewestPodcasts", s.getNewestPodcasts).Methods("GET", "POST")
	api.HandleFunc("/refreshPodcasts", s.refreshPodcasts).Methods("GET", "POST")
	api.HandleFunc("/createPodcastChannel", s.createPodcastChannel).Methods("GET", "POST")
	api.HandleFunc("/deletePodcastChannel", s.deletePodcastChannel).Methods("GET", "POST")
	api.HandleFunc("/deletePodcastEpisode", s.deletePodcastEpisode).Methods("GET", "POST")
	api.HandleFunc("/downloadPodcastEpisode", s.downloadPodcastEpisode).Methods("GET", "POST")

	// Internet radio
	api.HandleFunc("/getInternetRadioStations", s.getInternetRadioStations).Methods("GET", "POST")
	api.HandleFunc("/createInternetRadioStation", s.createInternetRadioStation).Methods("GET", "POST")
	api.HandleFunc("/updateInternetRadioStation", s.updateInternetRadioStation).Methods("GET", "POST")
	api.HandleFunc("/deleteInternetRadioStation", s.deleteInternetRadioStation).Methods("GET", "POST")
}

// Authentication middleware
func (s *SubsonicServer) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get auth parameters
		user := r.FormValue("u")
		pass := r.FormValue("p")
		token := r.FormValue("t")
		salt := r.FormValue("s")

		// Validate authentication
		if user == "" {
			s.sendError(w, r, 10, "Required parameter is missing")
			return
		}

		// Check token auth (preferred)
		if token != "" && salt != "" {
			if !s.validateToken(user, token, salt) {
				s.sendError(w, r, 40, "Wrong username or password")
				return
			}
		} else if pass != "" {
			// Check password auth (legacy)
			if !s.validatePassword(user, pass) {
				s.sendError(w, r, 40, "Wrong username or password")
				return
			}
		} else {
			s.sendError(w, r, 10, "Required parameter is missing")
			return
		}

		// Store username in context
		ctx := r.Context()
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (s *SubsonicServer) validateToken(username, token, salt string) bool {
	// Get user's password from database
	var passwordHash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
	if err != nil {
		return false
	}

	// Calculate expected token: md5(password + salt)
	expectedToken := md5.Sum([]byte(passwordHash + salt))
	expectedTokenStr := hex.EncodeToString(expectedToken[:])

	return token == expectedTokenStr
}

func (s *SubsonicServer) validatePassword(username, password string) bool {
	// Simple validation for now
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	return err == nil && exists
}

// Response helpers
func (s *SubsonicServer) sendResponse(w http.ResponseWriter, r *http.Request, resp *SubsonicResponse) {
	resp.Status = "ok"
	resp.Version = "1.16.1"
	resp.Type = "casrad"
	resp.ServerVersion = "1.0.0"

	format := r.FormValue("f")
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"subsonic-response": resp,
		})
	} else {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xml.Header))
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *SubsonicServer) sendError(w http.ResponseWriter, r *http.Request, code int, message string) {
	resp := &SubsonicResponse{
		Status:  "failed",
		Version: "1.16.1",
		Type:    "casrad",
		ServerVersion: "1.0.0",
		Error: &SubsonicError{
			Code:    code,
			Message: message,
		},
	}

	format := r.FormValue("f")
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"subsonic-response": resp,
		})
	} else {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xml.Header))
		xml.NewEncoder(w).Encode(resp)
	}
}

// API endpoint implementations
func (s *SubsonicServer) ping(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(w, r, &SubsonicResponse{})
}

func (s *SubsonicServer) getLicense(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(w, r, &SubsonicResponse{
		License: &License{
			Valid:  true,
			Email:  "admin@casrad.local",
			LicenseExpires: "2099-12-31T23:59:59",
		},
	})
}

func (s *SubsonicServer) getMusicFolders(w http.ResponseWriter, r *http.Request) {
	folders := []MusicFolder{}

	// Get global directories
	rows, err := s.db.Query("SELECT id, type FROM global_directories WHERE is_active = 1")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var folderType string
			if rows.Scan(&id, &folderType) == nil {
				folders = append(folders, MusicFolder{
					ID:   id,
					Name: strings.Title(folderType),
				})
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		MusicFolders: &MusicFolders{Folders: folders},
	})
}

func (s *SubsonicServer) getIndexes(w http.ResponseWriter, r *http.Request) {
	// Get artist index
	indexMap := make(map[string][]Artist)

	rows, err := s.db.Query(`
		SELECT DISTINCT substr(upper(artist), 1, 1) as letter, artist
		FROM tracks
		WHERE artist IS NOT NULL
		ORDER BY letter, artist
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var letter, artist string
			if rows.Scan(&letter, &artist) == nil {
				indexMap[letter] = append(indexMap[letter], Artist{
					ID:   fmt.Sprintf("artist_%s", artist),
					Name: artist,
				})
			}
		}
	}

	indexes := []Index{}
	for letter, artists := range indexMap {
		indexes = append(indexes, Index{
			Name:   letter,
			Artist: artists,
		})
	}

	s.sendResponse(w, r, &SubsonicResponse{
		Indexes: &Indexes{
			LastModified: time.Now().Unix() * 1000,
			IgnoredArticles: "The El La Los Las Le Les",
			Index: indexes,
		},
	})
}

func (s *SubsonicServer) getMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		s.sendError(w, r, 10, "Required parameter 'id' is missing")
		return
	}

	// Simple implementation - return tracks for now
	children := []Child{}

	rows, err := s.db.Query(`
		SELECT id, title, artist, album, duration/1000, bitrate, file_size, file_type
		FROM tracks
		LIMIT 100
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trackID int
			var title, artist, album, fileType string
			var duration, bitrate int
			var size int64

			if rows.Scan(&trackID, &title, &artist, &album, &duration, &bitrate, &size, &fileType) == nil {
				children = append(children, Child{
					ID:          fmt.Sprintf("track_%d", trackID),
					IsDir:       false,
					Title:       title,
					Artist:      artist,
					Album:       album,
					Duration:    duration,
					BitRate:     bitrate,
					Size:        size,
					ContentType: "audio/mpeg",
					Suffix:      fileType,
					Type:        "music",
				})
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		Directory: &Directory{
			ID:       id,
			Name:     "Music",
			Children: children,
		},
	})
}

// Implement remaining endpoints with similar patterns...
func (s *SubsonicServer) getGenres(w http.ResponseWriter, r *http.Request)    { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getArtists(w http.ResponseWriter, r *http.Request)   { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getArtist(w http.ResponseWriter, r *http.Request)    { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getAlbum(w http.ResponseWriter, r *http.Request)     { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getSong(w http.ResponseWriter, r *http.Request)      { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getAlbumList(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }

func (s *SubsonicServer) getAlbumList2(w http.ResponseWriter, r *http.Request) {
	listType := r.FormValue("type")
	if listType == "" {
		listType = "random"
	}

	size := 10
	if s := r.FormValue("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
			size = n
		}
	}

	var query string
	switch listType {
	case "newest":
		query = "SELECT DISTINCT album, artist FROM tracks WHERE album IS NOT NULL ORDER BY created_at DESC LIMIT ?"
	case "random":
		query = "SELECT DISTINCT album, artist FROM tracks WHERE album IS NOT NULL ORDER BY RANDOM() LIMIT ?"
	case "frequent":
		query = "SELECT DISTINCT album, artist FROM tracks WHERE album IS NOT NULL ORDER BY play_count DESC LIMIT ?"
	case "recent":
		query = "SELECT DISTINCT album, artist FROM tracks WHERE album IS NOT NULL ORDER BY last_played DESC LIMIT ?"
	default:
		query = "SELECT DISTINCT album, artist FROM tracks WHERE album IS NOT NULL ORDER BY RANDOM() LIMIT ?"
	}

	albums := []Album{}
	rows, err := s.db.Query(query, size)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var album, artist string
			if rows.Scan(&album, &artist) == nil {
				albums = append(albums, Album{
					ID:     fmt.Sprintf("album_%s", album),
					Title:  album,
					Artist: artist,
					IsDir:  true,
				})
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		AlbumList2: &AlbumList2{Albums: albums},
	})
}

func (s *SubsonicServer) getRandomSongs(w http.ResponseWriter, r *http.Request) {
	size := 10
	if s := r.FormValue("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
			size = n
		}
	}

	songs := []Song{}
	rows, err := s.db.Query(`
		SELECT id, title, artist, album, duration/1000, bitrate, file_size
		FROM tracks
		ORDER BY RANDOM()
		LIMIT ?
	`, size)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var title, artist, album string
			var duration, bitrate int
			var size int64

			if rows.Scan(&id, &title, &artist, &album, &duration, &bitrate, &size) == nil {
				songs = append(songs, Song{
					ID:          fmt.Sprintf("track_%d", id),
					Title:       title,
					Artist:      artist,
					Album:       album,
					Duration:    duration,
					BitRate:     bitrate,
					Size:        size,
					ContentType: "audio/mpeg",
					Suffix:      "mp3",
				})
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		RandomSongs: &RandomSongs{Songs: songs},
	})
}

func (s *SubsonicServer) getSongsByGenre(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getNowPlaying(w http.ResponseWriter, r *http.Request)   { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getStarred(w http.ResponseWriter, r *http.Request)      { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getStarred2(w http.ResponseWriter, r *http.Request)     { s.sendResponse(w, r, &SubsonicResponse{}) }

// Search endpoints
func (s *SubsonicServer) search(w http.ResponseWriter, r *http.Request)  { s.search3(w, r) }
func (s *SubsonicServer) search2(w http.ResponseWriter, r *http.Request) { s.search3(w, r) }

func (s *SubsonicServer) search3(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		s.sendError(w, r, 10, "Required parameter 'query' is missing")
		return
	}

	searchPattern := "%" + query + "%"
	result := &SearchResult3{
		Artists: []Artist{},
		Albums:  []Album{},
		Songs:   []Song{},
	}

	// Search artists
	rows, err := s.db.Query("SELECT DISTINCT artist FROM tracks WHERE artist LIKE ? LIMIT 20", searchPattern)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var artist string
			if rows.Scan(&artist) == nil {
				result.Artists = append(result.Artists, Artist{
					ID:   fmt.Sprintf("artist_%s", artist),
					Name: artist,
				})
			}
		}
	}

	// Search albums
	rows, err = s.db.Query("SELECT DISTINCT album, artist FROM tracks WHERE album LIKE ? LIMIT 20", searchPattern)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var album, artist string
			if rows.Scan(&album, &artist) == nil {
				result.Albums = append(result.Albums, Album{
					ID:     fmt.Sprintf("album_%s", album),
					Title:  album,
					Artist: artist,
					IsDir:  true,
				})
			}
		}
	}

	// Search songs
	rows, err = s.db.Query(`
		SELECT id, title, artist, album, duration/1000, bitrate, file_size
		FROM tracks
		WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?
		LIMIT 50
	`, searchPattern, searchPattern, searchPattern)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var title, artist, album string
			var duration, bitrate int
			var size int64

			if rows.Scan(&id, &title, &artist, &album, &duration, &bitrate, &size) == nil {
				result.Songs = append(result.Songs, Song{
					ID:          fmt.Sprintf("track_%d", id),
					Title:       title,
					Artist:      artist,
					Album:       album,
					Duration:    duration,
					BitRate:     bitrate,
					Size:        size,
					ContentType: "audio/mpeg",
					Suffix:      "mp3",
				})
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		SearchResult3: result,
	})
}

// Playlist endpoints
func (s *SubsonicServer) getPlaylists(w http.ResponseWriter, r *http.Request) {
	playlists := []Playlist{}

	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.description, u.username, p.is_public, p.track_count,
		       p.duration_ms/1000, p.created_at, p.updated_at
		FROM playlists p
		JOIN users u ON p.user_id = u.id
		WHERE p.is_public = 1
		ORDER BY p.name
	`)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			var comment sql.NullString
			var owner string
			var isPublic bool
			var songCount, duration int
			var created, changed time.Time

			if rows.Scan(&id, &name, &comment, &owner, &isPublic, &songCount, &duration, &created, &changed) == nil {
				pl := Playlist{
					ID:        fmt.Sprintf("playlist_%d", id),
					Name:      name,
					Owner:     owner,
					Public:    isPublic,
					SongCount: songCount,
					Duration:  duration,
					Created:   created.Format(time.RFC3339),
					Changed:   changed.Format(time.RFC3339),
				}
				if comment.Valid {
					pl.Comment = comment.String
				}
				playlists = append(playlists, pl)
			}
		}
	}

	s.sendResponse(w, r, &SubsonicResponse{
		Playlists: &Playlists{Playlist: playlists},
	})
}

func (s *SubsonicServer) getPlaylist(w http.ResponseWriter, r *http.Request)    { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) createPlaylist(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) updatePlaylist(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) deletePlaylist(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }

// Media retrieval
func (s *SubsonicServer) stream(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		s.sendError(w, r, 10, "Required parameter 'id' is missing")
		return
	}

	// TODO: Implement actual streaming
	w.Header().Set("Content-Type", "audio/mpeg")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Audio stream placeholder"))
}

func (s *SubsonicServer) download(w http.ResponseWriter, r *http.Request)   { s.stream(w, r) }
func (s *SubsonicServer) getCoverArt(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }
func (s *SubsonicServer) getLyrics(w http.ResponseWriter, r *http.Request)   { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getAvatar(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotFound) }

// Media annotation
func (s *SubsonicServer) star(w http.ResponseWriter, r *http.Request)      { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) unstar(w http.ResponseWriter, r *http.Request)    { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) setRating(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) scrobble(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }

// User management
func (s *SubsonicServer) getUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	if username == "" {
		s.sendError(w, r, 10, "Required parameter 'username' is missing")
		return
	}

	var role string
	err := s.db.QueryRow("SELECT role FROM users WHERE username = ?", username).Scan(&role)
	if err != nil {
		s.sendError(w, r, 70, "User not found")
		return
	}

	isAdmin := role == "admin"
	s.sendResponse(w, r, &SubsonicResponse{
		User: &User{
			Username:          username,
			ScrobblingEnabled: true,
			AdminRole:         isAdmin,
			SettingsRole:      isAdmin,
			DownloadRole:      true,
			UploadRole:        true,
			PlaylistRole:      true,
			CoverArtRole:      true,
			CommentRole:       true,
			PodcastRole:       true,
			StreamRole:        true,
			JukeboxRole:       false,
			ShareRole:         true,
		},
	})
}

func (s *SubsonicServer) getUsers(w http.ResponseWriter, r *http.Request)    { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) createUser(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) updateUser(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) deleteUser(w http.ResponseWriter, r *http.Request)  { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) changePassword(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }

// Library scanning
func (s *SubsonicServer) getScanStatus(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(w, r, &SubsonicResponse{
		ScanStatus: &ScanStatus{
			Scanning: false,
			Count:    0,
		},
	})
}

func (s *SubsonicServer) startScan(w http.ResponseWriter, r *http.Request) {
	// TODO: Trigger library scan
	s.sendResponse(w, r, &SubsonicResponse{
		ScanStatus: &ScanStatus{
			Scanning: true,
			Count:    0,
		},
	})
}

// Podcasts
func (s *SubsonicServer) getPodcasts(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) getNewestPodcasts(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) refreshPodcasts(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) createPodcastChannel(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) deletePodcastChannel(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) deletePodcastEpisode(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) downloadPodcastEpisode(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }

// Internet radio
func (s *SubsonicServer) getInternetRadioStations(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) createInternetRadioStation(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) updateInternetRadioStation(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
func (s *SubsonicServer) deleteInternetRadioStation(w http.ResponseWriter, r *http.Request) { s.sendResponse(w, r, &SubsonicResponse{}) }
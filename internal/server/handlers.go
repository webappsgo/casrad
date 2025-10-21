package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// handleGetTracks returns all tracks
func (s *HTTPServer) handleGetTracks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, file_path, COALESCE(title, ''), COALESCE(artist, ''),
		       COALESCE(album, ''), COALESCE(genre, ''), COALESCE(duration, 0)
		FROM tracks
		ORDER BY artist, album, track_number
		LIMIT 1000
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []map[string]interface{}
	for rows.Next() {
		var id int
		var filePath, title, artist, album, genre string
		var duration int

		if err := rows.Scan(&id, &filePath, &title, &artist, &album, &genre, &duration); err != nil {
			continue
		}

		// Use filename as title if empty
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		}

		tracks = append(tracks, map[string]interface{}{
			"id":        id,
			"title":     title,
			"artist":    artist,
			"album":     album,
			"genre":     genre,
			"duration":  duration,
			"file_path": filePath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// handleGetAlbums returns all albums
func (s *HTTPServer) handleGetAlbums(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT DISTINCT COALESCE(album, 'Unknown'), COALESCE(artist, 'Unknown'), COUNT(*) as track_count
		FROM tracks
		WHERE album IS NOT NULL
		GROUP BY album, artist
		ORDER BY album
		LIMIT 500
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var albums []map[string]interface{}
	id := 1
	for rows.Next() {
		var album, artist string
		var trackCount int

		if err := rows.Scan(&album, &artist, &trackCount); err != nil {
			continue
		}

		albums = append(albums, map[string]interface{}{
			"id":          id,
			"title":       album,
			"artist":      artist,
			"track_count": trackCount,
		})
		id++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albums)
}

// handleGetAlbumTracks returns tracks for an album
func (s *HTTPServer) handleGetAlbumTracks(w http.ResponseWriter, r *http.Request) {
	albumID := mux.Vars(r)["id"]

	rows, err := s.db.Query(`
		SELECT id, file_path, title, artist, album, genre, duration
		FROM tracks
		WHERE album = ?
		ORDER BY track_number, title
	`, albumID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []map[string]interface{}
	for rows.Next() {
		var id int
		var filePath, title, artist, album, genre string
		var duration int

		if err := rows.Scan(&id, &filePath, &title, &artist, &album, &genre, &duration); err != nil {
			continue
		}

		tracks = append(tracks, map[string]interface{}{
			"id":       id,
			"title":    title,
			"artist":   artist,
			"album":    album,
			"genre":    genre,
			"duration": duration,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// handleGetArtists returns all artists
func (s *HTTPServer) handleGetArtists(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT DISTINCT COALESCE(artist, 'Unknown'), COUNT(*) as track_count
		FROM tracks
		WHERE artist IS NOT NULL
		GROUP BY artist
		ORDER BY artist
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var artists []map[string]interface{}
	id := 1
	for rows.Next() {
		var artist string
		var trackCount int

		if err := rows.Scan(&artist, &trackCount); err != nil {
			continue
		}

		artists = append(artists, map[string]interface{}{
			"id":          id,
			"name":        artist,
			"track_count": trackCount,
		})
		id++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artists)
}

// handleGetPlaylists returns all playlists
func (s *HTTPServer) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, COALESCE(name, ''), COALESCE(description, ''), track_count
		FROM playlists
		ORDER BY name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var playlists []map[string]interface{}
	for rows.Next() {
		var id int
		var name, description string
		var trackCount int

		if err := rows.Scan(&id, &name, &description, &trackCount); err != nil {
			continue
		}

		playlists = append(playlists, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"track_count": trackCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

// handleStream streams audio file
func (s *HTTPServer) handleStream(w http.ResponseWriter, r *http.Request) {
	trackID := mux.Vars(r)["id"]

	// Get file path from database
	var filePath string
	err := s.db.QueryRow("SELECT file_path FROM tracks WHERE id = ?", trackID).Scan(&filePath)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Try to find in test music directory
		testPath := filepath.Join("./test-music", filepath.Base(filePath))
		if _, err := os.Stat(testPath); err == nil {
			filePath = testPath
		} else {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set appropriate content type
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".mp3":
		contentType = "audio/mpeg"
	case ".flac":
		contentType = "audio/flac"
	case ".ogg", ".oga":
		contentType = "audio/ogg"
	case ".m4a":
		contentType = "audio/mp4"
	case ".wav":
		contentType = "audio/wav"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Accept-Ranges", "bytes")

	// Handle range requests for seeking
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		s.handleRangeRequest(w, r, file, stat.Size())
	} else {
		// Stream entire file
		io.Copy(w, file)
	}
}

// handleRangeRequest handles partial content requests for seeking
func (s *HTTPServer) handleRangeRequest(w http.ResponseWriter, r *http.Request, file *os.File, size int64) {
	rangeHeader := r.Header.Get("Range")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")

	var start, end int64
	if parts[0] != "" {
		start, _ = strconv.ParseInt(parts[0], 10, 64)
	}
	if len(parts) > 1 && parts[1] != "" {
		end, _ = strconv.ParseInt(parts[1], 10, 64)
	} else {
		end = size - 1
	}

	if start > end || end >= size {
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	w.WriteHeader(http.StatusPartialContent)

	file.Seek(start, 0)
	io.CopyN(w, file, end-start+1)
}

// handleTrackPlay records a play event
func (s *HTTPServer) handleTrackPlay(w http.ResponseWriter, r *http.Request) {
	trackID := mux.Vars(r)["id"]

	// Update play count
	s.db.Exec(`
		UPDATE tracks
		SET play_count = play_count + 1,
		    last_played = CURRENT_TIMESTAMP
		WHERE id = ?
	`, trackID)

	// Record in history
	s.db.Exec(`
		INSERT INTO playback_history (user_id, track_id, source)
		VALUES (NULL, ?, 'web')
	`, trackID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleMetrics returns current metrics
func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Simple metrics for now
	metrics := map[string]interface{}{
		"system.memory.alloc": map[string]interface{}{
			"value": 100 * 1024 * 1024,
			"unit":  "bytes",
		},
		"streaming.streams.active": map[string]interface{}{
			"value": 0,
			"unit":  "count",
		},
		"application.sessions.active": map[string]interface{}{
			"value": 1,
			"unit":  "count",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleScanLibrary triggers a library scan
func (s *HTTPServer) handleScanLibrary(w http.ResponseWriter, r *http.Request) {
	// Trigger library scan
	go s.scanMusicDirectory("/mnt/Music/Mp3")
	go s.scanMusicDirectory("./test-music")

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "scanning",
		"message": "Library scan started",
	})
}

// scanMusicDirectory scans a directory for music files
func (s *HTTPServer) scanMusicDirectory(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".mp3", ".flac", ".ogg", ".m4a", ".wav":
			// Extract filename as title
			title := strings.TrimSuffix(filepath.Base(path), ext)

			// Add to database with default values
			_, err := s.db.Exec(`
				INSERT OR IGNORE INTO tracks (
					file_path, title, artist, album, genre,
					duration, file_size, is_global
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, path, title, "Unknown Artist", "Unknown Album",
			   "Unknown", 0, info.Size(), true)

			if err != nil {
				fmt.Printf("Error inserting track: %v\n", err)
			} else {
				fmt.Printf("Added track: %s\n", title)
			}
		}

		return nil
	})
}
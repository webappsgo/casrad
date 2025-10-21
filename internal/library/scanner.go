package library

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/dhowden/tag"
)

type Scanner struct {
	db           *database.Engine
	ffmpegPath   string
	ffprobePath  string
	scanMutex    sync.Mutex
	isScanning   bool
	progressChan chan ScanProgress
}

type ScanProgress struct {
	TotalFiles     int
	ProcessedFiles int
	CurrentFile    string
	Errors         []string
}

func NewScanner(db *database.Engine, ffmpegPath, ffprobePath string) *Scanner {
	return &Scanner{
		db:           db,
		ffmpegPath:   ffmpegPath,
		ffprobePath:  ffprobePath,
		progressChan: make(chan ScanProgress, 1),
	}
}

func (s *Scanner) IsScanning() bool {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()
	return s.isScanning
}

func (s *Scanner) ScanDirectory(path string, isGlobal bool, userID *int) error {
	s.scanMutex.Lock()
	if s.isScanning {
		s.scanMutex.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	s.isScanning = true
	s.scanMutex.Unlock()

	defer func() {
		s.scanMutex.Lock()
		s.isScanning = false
		s.scanMutex.Unlock()
	}()

	log.Printf("Starting library scan of %s (global: %v)", path, isGlobal)

	// Find all audio files
	audioFiles, err := s.findAudioFiles(path)
	if err != nil {
		return fmt.Errorf("failed to find audio files: %w", err)
	}

	progress := ScanProgress{
		TotalFiles: len(audioFiles),
		Errors:     []string{},
	}

	// Process each file
	for i, filePath := range audioFiles {
		progress.ProcessedFiles = i + 1
		progress.CurrentFile = filePath

		// Send progress update
		select {
		case s.progressChan <- progress:
		default:
		}

		// Check if file already exists in database
		var existingID int
		err := s.db.QueryRow("SELECT id FROM tracks WHERE file_path = ?", filePath).Scan(&existingID)
		if err == nil {
			// File already exists, check if it needs updating
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			var lastUpdated time.Time
			s.db.QueryRow("SELECT updated_at FROM tracks WHERE id = ?", existingID).Scan(&lastUpdated)

			if fileInfo.ModTime().After(lastUpdated) {
				// File has been modified, update metadata
				if err := s.updateTrack(existingID, filePath); err != nil {
					progress.Errors = append(progress.Errors, fmt.Sprintf("%s: %v", filePath, err))
				}
			}
		} else {
			// New file, add to database
			if err := s.addTrack(filePath, isGlobal, userID); err != nil {
				progress.Errors = append(progress.Errors, fmt.Sprintf("%s: %v", filePath, err))
			}
		}
	}

	// Update scan timestamp
	_, err = s.db.Exec(`
		UPDATE global_directories
		SET last_scan = CURRENT_TIMESTAMP,
		    file_count = ?,
		    total_size_bytes = (
		        SELECT COALESCE(SUM(file_size), 0)
		        FROM tracks
		        WHERE file_path LIKE ? || '%'
		    )
		WHERE path = ?
	`, len(audioFiles), path, path)

	log.Printf("Library scan completed: %d files processed, %d errors",
		progress.ProcessedFiles, len(progress.Errors))

	return nil
}

func (s *Scanner) findAudioFiles(root string) ([]string, error) {
	var files []string
	supportedExtensions := []string{
		".mp3", ".flac", ".ogg", ".m4a", ".aac", ".wav",
		".opus", ".wma", ".ape", ".mpc", ".wv", ".dsf",
		".dff", ".mp4", ".webm", ".mkv", // Video files with audio
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file has supported extension
		ext := strings.ToLower(filepath.Ext(path))
		for _, supported := range supportedExtensions {
			if ext == supported {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

func (s *Scanner) addTrack(filePath string, isGlobal bool, userID *int) error {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Calculate file hash for deduplication
	hash, err := s.calculateFileHash(filePath)
	if err != nil {
		return err
	}

	// Extract metadata
	metadata, err := s.extractMetadata(filePath)
	if err != nil {
		// If metadata extraction fails, use filename
		metadata = &TrackMetadata{
			Title:  strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
			Artist: "Unknown Artist",
			Album:  "Unknown Album",
		}
	}

	// Insert into database
	result, err := s.db.Exec(`
		INSERT INTO tracks (
			file_path, file_hash, user_id, is_global,
			title, artist, album, album_artist,
			genre, year, track_number, disc_number,
			duration, bitrate, sample_rate, channels,
			codec, file_type, file_size,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`, filePath, hash, userID, isGlobal,
		metadata.Title, metadata.Artist, metadata.Album, metadata.AlbumArtist,
		metadata.Genre, metadata.Year, metadata.TrackNumber, metadata.DiscNumber,
		metadata.Duration, metadata.Bitrate, metadata.SampleRate, metadata.Channels,
		metadata.Codec, strings.TrimPrefix(filepath.Ext(filePath), "."), fileInfo.Size())

	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}

	trackID, _ := result.LastInsertId()

	// Create or update album
	s.createOrUpdateAlbum(metadata, trackID)

	// Create or update artist
	s.createOrUpdateArtist(metadata, trackID)

	return nil
}

func (s *Scanner) updateTrack(trackID int, filePath string) error {
	// Re-extract metadata
	metadata, err := s.extractMetadata(filePath)
	if err != nil {
		return err
	}

	// Update track in database
	_, err = s.db.Exec(`
		UPDATE tracks SET
			title = ?, artist = ?, album = ?, album_artist = ?,
			genre = ?, year = ?, track_number = ?, disc_number = ?,
			duration = ?, bitrate = ?, sample_rate = ?, channels = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, metadata.Title, metadata.Artist, metadata.Album, metadata.AlbumArtist,
		metadata.Genre, metadata.Year, metadata.TrackNumber, metadata.DiscNumber,
		metadata.Duration, metadata.Bitrate, metadata.SampleRate, metadata.Channels,
		trackID)

	return err
}

func (s *Scanner) calculateFileHash(filePath string) (string, error) {
	// Read first 1MB of file for hash
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buffer := make([]byte, 1024*1024) // 1MB buffer
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	hasher.Write(buffer[:n])
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *Scanner) extractMetadata(filePath string) (*TrackMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Use dhowden/tag library for metadata extraction
	m, err := tag.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	metadata := &TrackMetadata{
		Title:       m.Title(),
		Artist:      m.Artist(),
		Album:       m.Album(),
		AlbumArtist: m.AlbumArtist(),
		Genre:       m.Genre(),
		Year:        m.Year(),
	}

	// Handle track and disc numbers
	trackNum, trackTotal := m.Track()
	metadata.TrackNumber = trackNum
	metadata.TrackTotal = trackTotal

	discNum, discTotal := m.Disc()
	metadata.DiscNumber = discNum
	metadata.DiscTotal = discTotal

	// Get technical metadata using file info
	// In production, you would use ffprobe for accurate technical data
	fileInfo, _ := file.Stat()
	metadata.FileSize = fileInfo.Size()

	// Codec detection from file extension (simplified)
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		metadata.Codec = "mp3"
		metadata.Bitrate = 192 // Default assumption
	case ".flac":
		metadata.Codec = "flac"
		metadata.Bitrate = 0 // Lossless
	case ".ogg":
		metadata.Codec = "vorbis"
		metadata.Bitrate = 160
	case ".m4a":
		metadata.Codec = "aac"
		metadata.Bitrate = 256
	case ".opus":
		metadata.Codec = "opus"
		metadata.Bitrate = 128
	default:
		metadata.Codec = strings.TrimPrefix(ext, ".")
	}

	// Default sample rate and channels
	metadata.SampleRate = 44100
	metadata.Channels = 2

	return metadata, nil
}

func (s *Scanner) createOrUpdateAlbum(metadata *TrackMetadata, trackID int64) {
	if metadata.Album == "" {
		return
	}

	// Check if album exists
	var albumID int
	err := s.db.QueryRow(`
		SELECT id FROM albums
		WHERE title = ? AND album_artist = ?
	`, metadata.Album, metadata.AlbumArtist).Scan(&albumID)

	if err == sql.ErrNoRows {
		// Create new album
		s.db.Exec(`
			INSERT INTO albums (
				title, artist, album_artist, year, genre,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, metadata.Album, metadata.Artist, metadata.AlbumArtist,
		   metadata.Year, metadata.Genre)
	} else if err == nil {
		// Update album track count
		s.db.Exec(`
			UPDATE albums
			SET total_tracks = (
				SELECT COUNT(*) FROM tracks
				WHERE album = ? AND album_artist = ?
			),
			updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, metadata.Album, metadata.AlbumArtist, albumID)
	}
}

func (s *Scanner) createOrUpdateArtist(metadata *TrackMetadata, trackID int64) {
	if metadata.Artist == "" {
		return
	}

	// Check if artist exists
	var artistID int
	err := s.db.QueryRow(`
		SELECT id FROM artists WHERE name = ?
	`, metadata.Artist).Scan(&artistID)

	if err == sql.ErrNoRows {
		// Create new artist
		s.db.Exec(`
			INSERT INTO artists (
				name, sort_name, genre,
				created_at, updated_at
			) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, metadata.Artist, metadata.Artist, metadata.Genre)
	}
}

type TrackMetadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        int
	TrackNumber int
	TrackTotal  int
	DiscNumber  int
	DiscTotal   int
	Duration    int
	Bitrate     int
	SampleRate  int
	Channels    int
	Codec       string
	FileSize    int64
}

// ScanAllLibraries scans all configured libraries
func (s *Scanner) ScanAllLibraries() error {
	// Scan global directories
	rows, err := s.db.Query(`
		SELECT path, type FROM global_directories
		WHERE is_active = 1
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path, dirType string
		if err := rows.Scan(&path, &dirType); err != nil {
			continue
		}

		log.Printf("Scanning global %s directory: %s", dirType, path)
		if err := s.ScanDirectory(path, true, nil); err != nil {
			log.Printf("Error scanning %s: %v", path, err)
		}
	}

	// Scan user directories
	userRows, err := s.db.Query(`
		SELECT u.id, us.music_paths
		FROM users u
		JOIN user_storage us ON u.id = us.user_id
		WHERE u.is_active = 1
	`)
	if err != nil {
		return err
	}
	defer userRows.Close()

	for userRows.Next() {
		var userID int
		var musicPaths string
		if err := userRows.Scan(&userID, &musicPaths); err != nil {
			continue
		}

		// Parse JSON array of paths (simplified)
		// In production, use proper JSON parsing
		paths := strings.Trim(musicPaths, "[]\"")
		for _, path := range strings.Split(paths, ",") {
			path = strings.Trim(path, "\"")
			if path != "" {
				log.Printf("Scanning user %d music directory: %s", userID, path)
				if err := s.ScanDirectory(path, false, &userID); err != nil {
					log.Printf("Error scanning %s: %v", path, err)
				}
			}
		}
	}

	return nil
}

// GetProgress returns the current scan progress
func (s *Scanner) GetProgress() ScanProgress {
	select {
	case progress := <-s.progressChan:
		return progress
	default:
		return ScanProgress{}
	}
}
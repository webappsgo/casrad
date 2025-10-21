package media

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// PodcastManager handles podcast subscriptions and downloads
type PodcastManager struct {
	db          *database.Engine
	storagePath string
	httpClient  *http.Client
	ffmpeg      *FFMPEGManager
}

// RSS feed structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Author      string    `xml:"author"`
	Image       Image     `xml:"image"`
	Category    string    `xml:"category"`
	Explicit    string    `xml:"explicit"`
	Items       []Episode `xml:"item"`
}

type Image struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

type Episode struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	GUID        string    `xml:"guid"`
	PubDate     string    `xml:"pubDate"`
	Link        string    `xml:"link"`
	Enclosure   Enclosure `xml:"enclosure"`
	Duration    string    `xml:"duration"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int64  `xml:"length,attr"`
}

// NewPodcastManager creates a new podcast manager
func NewPodcastManager(db *database.Engine, storagePath string, ffmpeg *FFMPEGManager) *PodcastManager {
	return &PodcastManager{
		db:          db,
		storagePath: storagePath,
		ffmpeg:      ffmpeg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubscribePodcast adds a new podcast subscription
func (pm *PodcastManager) SubscribePodcast(userID int, feedURL string) error {
	// Fetch and parse feed
	feed, err := pm.fetchFeed(feedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Determine storage path
	var storagePath string
	if userID > 0 {
		// User-specific path
		var username string
		err = pm.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
		if err != nil {
			return err
		}
		storagePath = filepath.Join(pm.storagePath, "users", username, "podcasts", pm.sanitizePath(feed.Channel.Title))
	} else {
		// Global path
		globalPath, _ := pm.db.GetSetting("storage.global_podcast_path")
		if globalPath == "" {
			globalPath = "/mnt/Podcasts"
		}
		storagePath = filepath.Join(globalPath, pm.sanitizePath(feed.Channel.Title))
	}

	// Create storage directory
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Parse explicit flag
	explicit := strings.ToLower(feed.Channel.Explicit) == "yes" ||
	           strings.ToLower(feed.Channel.Explicit) == "true"

	// Insert or update podcast
	var podcastID int64
	result, err := pm.db.Exec(`
		INSERT INTO podcasts (
			user_id, feed_url, title, description, author,
			image_url, website, language, category, explicit,
			storage_path, auto_download, download_quality,
			max_episodes, retention_days, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(feed_url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			author = excluded.author,
			image_url = excluded.image_url,
			last_check = CURRENT_TIMESTAMP
		RETURNING id
	`, userID, feedURL, feed.Channel.Title, feed.Channel.Description,
		feed.Channel.Author, feed.Channel.Image.URL, feed.Channel.Link,
		feed.Channel.Language, feed.Channel.Category, explicit,
		storagePath, true, "original", 100, 30, true)

	if err != nil {
		return fmt.Errorf("failed to save podcast: %w", err)
	}

	podcastID, _ = result.LastInsertId()

	// Process episodes
	for _, item := range feed.Channel.Items {
		if err := pm.processEpisode(int(podcastID), item); err != nil {
			log.Printf("Failed to process episode %s: %v", item.Title, err)
		}
	}

	return nil
}

// fetchFeed downloads and parses an RSS feed
func (pm *PodcastManager) fetchFeed(feedURL string) (*RSS, error) {
	resp, err := pm.httpClient.Get(feedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	var feed RSS
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

// processEpisode processes a single podcast episode
func (pm *PodcastManager) processEpisode(podcastID int, episode Episode) error {
	// Parse publication date
	pubDate, err := pm.parseDate(episode.PubDate)
	if err != nil {
		pubDate = time.Now()
	}

	// Parse duration
	duration := pm.parseDuration(episode.Duration)

	// Insert or update episode
	_, err = pm.db.Exec(`
		INSERT INTO podcast_episodes (
			podcast_id, guid, title, description, audio_url,
			website_url, published_at, duration, file_size
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(podcast_id, guid) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			audio_url = excluded.audio_url
	`, podcastID, episode.GUID, episode.Title, episode.Description,
		episode.Enclosure.URL, episode.Link, pubDate, duration, episode.Enclosure.Length)

	return err
}

// UpdatePodcasts checks all active podcasts for new episodes
func (pm *PodcastManager) UpdatePodcasts() error {
	rows, err := pm.db.Query(`
		SELECT id, feed_url, auto_download, max_episodes, retention_days, storage_path
		FROM podcasts
		WHERE is_active = 1
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var podcastID int
		var feedURL string
		var autoDownload bool
		var maxEpisodes, retentionDays int
		var storagePath string

		if err := rows.Scan(&podcastID, &feedURL, &autoDownload, &maxEpisodes, &retentionDays, &storagePath); err != nil {
			continue
		}

		// Update feed
		if err := pm.updatePodcast(podcastID, feedURL, autoDownload, maxEpisodes, retentionDays, storagePath); err != nil {
			log.Printf("Failed to update podcast %d: %v", podcastID, err)
			pm.db.Exec(`
				UPDATE podcasts
				SET last_error = ?, last_check = CURRENT_TIMESTAMP
				WHERE id = ?
			`, err.Error(), podcastID)
		} else {
			pm.db.Exec(`
				UPDATE podcasts
				SET last_error = NULL, last_check = CURRENT_TIMESTAMP
				WHERE id = ?
			`, podcastID)
		}
	}

	return nil
}

// updatePodcast updates a single podcast
func (pm *PodcastManager) updatePodcast(podcastID int, feedURL string, autoDownload bool, maxEpisodes, retentionDays int, storagePath string) error {
	// Fetch feed
	feed, err := pm.fetchFeed(feedURL)
	if err != nil {
		return err
	}

	// Update podcast metadata
	pm.db.Exec(`
		UPDATE podcasts SET
			title = ?, description = ?, author = ?,
			image_url = ?, website = ?, language = ?,
			category = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, feed.Channel.Title, feed.Channel.Description, feed.Channel.Author,
		feed.Channel.Image.URL, feed.Channel.Link, feed.Channel.Language,
		feed.Channel.Category, podcastID)

	// Process episodes
	episodeCount := 0
	for _, item := range feed.Channel.Items {
		if episodeCount >= maxEpisodes {
			break
		}

		// Process episode
		if err := pm.processEpisode(podcastID, item); err != nil {
			log.Printf("Failed to process episode: %v", err)
			continue
		}

		// Auto-download if enabled
		if autoDownload {
			go pm.downloadEpisode(podcastID, item.GUID, item.Enclosure.URL, storagePath)
		}

		episodeCount++
	}

	// Clean old episodes
	if retentionDays > 0 {
		pm.cleanOldEpisodes(podcastID, retentionDays)
	}

	return nil
}

// downloadEpisode downloads a podcast episode
func (pm *PodcastManager) downloadEpisode(podcastID int, guid, audioURL, storagePath string) error {
	// Check if already downloaded
	var isDownloaded bool
	pm.db.QueryRow(`
		SELECT is_downloaded FROM podcast_episodes
		WHERE podcast_id = ? AND guid = ?
	`, podcastID, guid).Scan(&isDownloaded)

	if isDownloaded {
		return nil
	}

	// Download file
	resp, err := pm.httpClient.Get(audioURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Generate filename
	filename := pm.sanitizePath(guid)
	if !strings.HasSuffix(filename, ".mp3") && !strings.HasSuffix(filename, ".m4a") {
		// Determine extension from content type
		contentType := resp.Header.Get("Content-Type")
		switch contentType {
		case "audio/mpeg":
			filename += ".mp3"
		case "audio/mp4", "audio/x-m4a":
			filename += ".m4a"
		case "audio/ogg":
			filename += ".ogg"
		default:
			filename += ".mp3"
		}
	}

	filePath := filepath.Join(storagePath, filename)

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy content
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filePath)
		return err
	}

	// Update database
	pm.db.Exec(`
		UPDATE podcast_episodes SET
			file_path = ?,
			file_size = ?,
			is_downloaded = 1,
			downloaded_at = CURRENT_TIMESTAMP,
			download_error = NULL
		WHERE podcast_id = ? AND guid = ?
	`, filePath, written, podcastID, guid)

	return nil
}

// cleanOldEpisodes removes episodes older than retention days
func (pm *PodcastManager) cleanOldEpisodes(podcastID int, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// Get episodes to delete
	rows, err := pm.db.Query(`
		SELECT id, file_path FROM podcast_episodes
		WHERE podcast_id = ? AND published_at < ? AND is_downloaded = 1
	`, podcastID, cutoffDate)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var episodeID int
		var filePath string

		if err := rows.Scan(&episodeID, &filePath); err != nil {
			continue
		}

		// Delete file
		if filePath != "" {
			os.Remove(filePath)
		}

		// Delete from database
		pm.db.Exec("DELETE FROM podcast_episodes WHERE id = ?", episodeID)
	}

	return nil
}

// GetPodcasts returns all podcasts for a user
func (pm *PodcastManager) GetPodcasts(userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, title, description, author, image_url,
		       (SELECT COUNT(*) FROM podcast_episodes WHERE podcast_id = p.id) as episode_count,
		       (SELECT COUNT(*) FROM podcast_episodes WHERE podcast_id = p.id AND is_played = 0) as unplayed_count
		FROM podcasts p
		WHERE user_id = ? OR user_id IS NULL
		ORDER BY title
	`

	rows, err := pm.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var podcasts []map[string]interface{}
	for rows.Next() {
		var id, episodeCount, unplayedCount int
		var title, description, author, imageURL string

		if err := rows.Scan(&id, &title, &description, &author, &imageURL, &episodeCount, &unplayedCount); err != nil {
			continue
		}

		podcasts = append(podcasts, map[string]interface{}{
			"id":             id,
			"title":          title,
			"description":    description,
			"author":         author,
			"image_url":      imageURL,
			"episode_count":  episodeCount,
			"unplayed_count": unplayedCount,
		})
	}

	return podcasts, nil
}

// GetEpisodes returns episodes for a podcast
func (pm *PodcastManager) GetEpisodes(podcastID int, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, title, description, audio_url, file_path,
		       published_at, duration, file_size, play_position,
		       is_played, is_downloaded
		FROM podcast_episodes
		WHERE podcast_id = ?
		ORDER BY published_at DESC
		LIMIT ?
	`

	rows, err := pm.db.Query(query, podcastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []map[string]interface{}
	for rows.Next() {
		var id, duration, playPosition int
		var fileSize int64
		var title, description, audioURL, filePath string
		var publishedAt time.Time
		var isPlayed, isDownloaded bool

		if err := rows.Scan(&id, &title, &description, &audioURL, &filePath,
			&publishedAt, &duration, &fileSize, &playPosition,
			&isPlayed, &isDownloaded); err != nil {
			continue
		}

		episodes = append(episodes, map[string]interface{}{
			"id":            id,
			"title":         title,
			"description":   description,
			"audio_url":     audioURL,
			"file_path":     filePath,
			"published_at":  publishedAt,
			"duration":      duration,
			"file_size":     fileSize,
			"play_position": playPosition,
			"is_played":     isPlayed,
			"is_downloaded": isDownloaded,
		})
	}

	return episodes, nil
}

// MarkEpisodePlayed marks an episode as played
func (pm *PodcastManager) MarkEpisodePlayed(episodeID int) error {
	_, err := pm.db.Exec(`
		UPDATE podcast_episodes
		SET is_played = 1, played_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, episodeID)
	return err
}

// UpdatePlayPosition updates the playback position of an episode
func (pm *PodcastManager) UpdatePlayPosition(episodeID int, position int) error {
	_, err := pm.db.Exec(`
		UPDATE podcast_episodes
		SET play_position = ?
		WHERE id = ?
	`, position, episodeID)
	return err
}

// sanitizePath sanitizes a string for use as a file/directory name
func (pm *PodcastManager) sanitizePath(s string) string {
	// Remove or replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		"\n", " ",
		"\r", " ",
	)
	s = replacer.Replace(s)
	s = strings.TrimSpace(s)

	// Limit length
	if len(s) > 100 {
		s = s[:100]
	}

	return s
}

// parseDate parses various date formats from RSS feeds
func (pm *PodcastManager) parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseDuration parses duration from various formats
func (pm *PodcastManager) parseDuration(durationStr string) int {
	// Try parsing as seconds
	var seconds int
	if _, err := fmt.Sscanf(durationStr, "%d", &seconds); err == nil {
		return seconds
	}

	// Try parsing as HH:MM:SS
	parts := strings.Split(durationStr, ":")
	if len(parts) == 3 {
		var h, m, s int
		fmt.Sscanf(parts[0], "%d", &h)
		fmt.Sscanf(parts[1], "%d", &m)
		fmt.Sscanf(parts[2], "%d", &s)
		return h*3600 + m*60 + s
	} else if len(parts) == 2 {
		var m, s int
		fmt.Sscanf(parts[0], "%d", &m)
		fmt.Sscanf(parts[1], "%d", &s)
		return m*60 + s
	}

	return 0
}
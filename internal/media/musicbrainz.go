package media

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/casapps/casrad/internal/database"
)

const (
	// MusicBrainz API
	MusicBrainzAPI = "https://musicbrainz.org/ws/2"
	AcoustIDAPI    = "https://api.acoustid.org/v2"
	CoverArtAPI    = "https://coverartarchive.org"

	// Rate limiting
	MBRateLimit = 1 * time.Second // 1 request per second

	// User agent
	UserAgent = "CASRAD/1.0 (https://github.com/casapps/casrad)"
)

// MusicBrainzClient handles MusicBrainz integration
type MusicBrainzClient struct {
	db         *database.Engine
	httpClient *http.Client
	ffmpeg     *FFMPEGManager
	lastCall   time.Time
	acoustIDKey string
}

// Recording represents a MusicBrainz recording
type Recording struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Length      int         `json:"length"`
	ArtistCredit []Artist   `json:"artist-credit"`
	Releases    []Release   `json:"releases"`
	ISRC        []string    `json:"isrc-list"`
	Tags        []Tag       `json:"tags"`
}

// Artist represents a MusicBrainz artist
type Artist struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort-name"`
	Type     string `json:"type"`
	Country  string `json:"country"`
	Disambiguation string `json:"disambiguation"`
}

// Release represents a MusicBrainz release (album)
type Release struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Status       string       `json:"status"`
	Country      string       `json:"country"`
	Date         string       `json:"date"`
	Barcode      string       `json:"barcode"`
	ArtistCredit []Artist     `json:"artist-credit"`
	LabelInfo    []LabelInfo  `json:"label-info"`
	Media        []Medium     `json:"media"`
}

// LabelInfo represents label information
type LabelInfo struct {
	Label         Label  `json:"label"`
	CatalogNumber string `json:"catalog-number"`
}

// Label represents a record label
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Medium represents a physical medium (CD, vinyl, etc)
type Medium struct {
	Format     string  `json:"format"`
	DiscCount  int     `json:"disc-count"`
	TrackCount int     `json:"track-count"`
	Tracks     []MBTrack `json:"tracks"`
}

// MBTrack represents a track on a release from MusicBrainz
type MBTrack struct {
	ID       string    `json:"id"`
	Number   string    `json:"number"`
	Title    string    `json:"title"`
	Length   int       `json:"length"`
	Recording Recording `json:"recording"`
}

// Tag represents a folksonomy tag
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// AcoustIDResult represents an AcoustID lookup result
type AcoustIDResult struct {
	Results []struct {
		ID         string      `json:"id"`
		Score      float64     `json:"score"`
		Recordings []Recording `json:"recordings"`
	} `json:"results"`
	Status string `json:"status"`
}

// NewMusicBrainzClient creates a new MusicBrainz client
func NewMusicBrainzClient(db *database.Engine, ffmpeg *FFMPEGManager) *MusicBrainzClient {
	return &MusicBrainzClient{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ffmpeg: ffmpeg,
		acoustIDKey: "cSpUJKpD", // Default AcoustID key for open source projects
	}
}

// LookupByFingerprint looks up a track using acoustic fingerprint
func (mb *MusicBrainzClient) LookupByFingerprint(filePath string) (*Recording, error) {
	// Generate fingerprint using chromaprint
	fingerprint, duration, err := mb.generateFingerprint(filePath)
	if err != nil {
		return nil, fmt.Errorf("fingerprint generation failed: %w", err)
	}

	// Lookup on AcoustID
	params := url.Values{
		"client":      {mb.acoustIDKey},
		"fingerprint": {fingerprint},
		"duration":    {fmt.Sprintf("%d", duration)},
		"meta":        {"recordings releasegroups releases tracks compress"},
		"format":      {"json"},
	}

	resp, err := mb.httpClient.Get(AcoustIDAPI + "/lookup?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AcoustIDResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "ok" || len(result.Results) == 0 {
		return nil, fmt.Errorf("no matches found")
	}

	// Get the best match
	bestMatch := result.Results[0]
	if len(bestMatch.Recordings) == 0 {
		return nil, fmt.Errorf("no recordings in match")
	}

	recording := &bestMatch.Recordings[0]

	// Enhance with MusicBrainz data
	if recording.ID != "" {
		enhanced, err := mb.GetRecording(recording.ID)
		if err == nil {
			recording = enhanced
		}
	}

	return recording, nil
}

// generateFingerprint generates an acoustic fingerprint using chromaprint
func (mb *MusicBrainzClient) generateFingerprint(filePath string) (string, int, error) {
	// Check if chromaprint is available
	chromaprintPath, err := exec.LookPath("fpcalc")
	if err != nil {
		// Try to use embedded chromaprint if available
		chromaprintPath = "/usr/local/bin/fpcalc"
		if _, err := os.Stat(chromaprintPath); err != nil {
			return "", 0, fmt.Errorf("chromaprint not found")
		}
	}

	// Run fpcalc
	cmd := exec.Command(chromaprintPath, "-json", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", 0, err
	}

	// Parse JSON output
	var result struct {
		Duration    float64 `json:"duration"`
		Fingerprint string  `json:"fingerprint"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", 0, err
	}

	return result.Fingerprint, int(result.Duration), nil
}

// GetRecording gets recording details from MusicBrainz
func (mb *MusicBrainzClient) GetRecording(mbid string) (*Recording, error) {
	mb.enforceRateLimit()

	url := fmt.Sprintf("%s/recording/%s?inc=artist-credits+releases+isrcs+tags&fmt=json",
		MusicBrainzAPI, mbid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := mb.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz returned status %d", resp.StatusCode)
	}

	var recording Recording
	if err := json.NewDecoder(resp.Body).Decode(&recording); err != nil {
		return nil, err
	}

	return &recording, nil
}

// GetRelease gets release details from MusicBrainz
func (mb *MusicBrainzClient) GetRelease(mbid string) (*Release, error) {
	mb.enforceRateLimit()

	url := fmt.Sprintf("%s/release/%s?inc=artist-credits+labels+recordings&fmt=json",
		MusicBrainzAPI, mbid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := mb.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// GetArtist gets artist details from MusicBrainz
func (mb *MusicBrainzClient) GetArtist(mbid string) (*Artist, error) {
	mb.enforceRateLimit()

	url := fmt.Sprintf("%s/artist/%s?inc=aliases+tags+ratings&fmt=json",
		MusicBrainzAPI, mbid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := mb.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz returned status %d", resp.StatusCode)
	}

	var artist Artist
	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		return nil, err
	}

	return &artist, nil
}

// GetCoverArt gets cover art for a release
func (mb *MusicBrainzClient) GetCoverArt(releaseMBID string) (string, error) {
	url := fmt.Sprintf("%s/release/%s", CoverArtAPI, releaseMBID)

	resp, err := mb.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no cover art available")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover art API returned status %d", resp.StatusCode)
	}

	var result struct {
		Images []struct {
			Thumbnails map[string]string `json:"thumbnails"`
			Image      string            `json:"image"`
			Types      []string          `json:"types"`
			Front      bool              `json:"front"`
			Back       bool              `json:"back"`
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Find front cover
	for _, img := range result.Images {
		if img.Front {
			// Prefer large thumbnail
			if large, ok := img.Thumbnails["large"]; ok {
				return large, nil
			}
			return img.Image, nil
		}
	}

	// Return first image if no front cover
	if len(result.Images) > 0 {
		return result.Images[0].Image, nil
	}

	return "", fmt.Errorf("no images found")
}

// TagFile tags a file with MusicBrainz metadata
func (mb *MusicBrainzClient) TagFile(trackID int) error {
	// Get track from database
	var filePath, fileHash string
	err := mb.db.QueryRow(`
		SELECT file_path, file_hash FROM tracks WHERE id = ?
	`, trackID).Scan(&filePath, &fileHash)
	if err != nil {
		return err
	}

	// Generate hash if not present
	if fileHash == "" {
		hash, err := mb.generateFileHash(filePath)
		if err == nil {
			fileHash = hash
			mb.db.Exec(`UPDATE tracks SET file_hash = ? WHERE id = ?`, fileHash, trackID)
		}
	}

	// Try fingerprint lookup
	recording, err := mb.LookupByFingerprint(filePath)
	if err != nil {
		return fmt.Errorf("fingerprint lookup failed: %w", err)
	}

	// Update track metadata
	if recording != nil {
		mb.updateTrackMetadata(trackID, recording)
	}

	return nil
}

// updateTrackMetadata updates track metadata in database
func (mb *MusicBrainzClient) updateTrackMetadata(trackID int, recording *Recording) error {
	// Extract primary artist
	artist := ""
	artistMBID := ""
	if len(recording.ArtistCredit) > 0 {
		artist = recording.ArtistCredit[0].Name
		artistMBID = recording.ArtistCredit[0].ID
	}

	// Extract album info from first release
	album := ""
	albumMBID := ""
	date := ""
	label := ""
	barcode := ""
	if len(recording.Releases) > 0 {
		release := recording.Releases[0]
		album = release.Title
		albumMBID = release.ID
		date = release.Date
		barcode = release.Barcode

		if len(release.LabelInfo) > 0 && release.LabelInfo[0].Label.Name != "" {
			label = release.LabelInfo[0].Label.Name
		}
	}

	// Extract ISRC
	isrc := ""
	if len(recording.ISRC) > 0 {
		isrc = recording.ISRC[0]
	}

	// Extract tags
	var tags []string
	for _, tag := range recording.Tags {
		tags = append(tags, tag.Name)
	}
	tagsJSON, _ := json.Marshal(tags)

	// Update database
	_, err := mb.db.Exec(`
		UPDATE tracks SET
			title = COALESCE(NULLIF(?, ''), title),
			artist = COALESCE(NULLIF(?, ''), artist),
			album = COALESCE(NULLIF(?, ''), album),
			mbid = ?,
			album_mbid = ?,
			artist_mbid = ?,
			date = COALESCE(NULLIF(?, ''), date),
			label = COALESCE(NULLIF(?, ''), label),
			barcode = COALESCE(NULLIF(?, ''), barcode),
			isrc = COALESCE(NULLIF(?, ''), isrc),
			tags = ?,
			analyzed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, recording.Title, artist, album, recording.ID, albumMBID, artistMBID,
		date, label, barcode, isrc, string(tagsJSON), trackID)

	if err != nil {
		return err
	}

	// Try to get cover art
	if albumMBID != "" {
		coverURL, err := mb.GetCoverArt(albumMBID)
		if err == nil && coverURL != "" {
			mb.downloadCoverArt(trackID, albumMBID, coverURL)
		}
	}

	return nil
}

// downloadCoverArt downloads and saves cover art
func (mb *MusicBrainzClient) downloadCoverArt(trackID int, albumMBID, coverURL string) error {
	resp, err := mb.httpClient.Get(coverURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create cover art directory
	coverDir := "/var/lib/casrad/covers"
	os.MkdirAll(coverDir, 0755)

	// Save cover art
	coverPath := fmt.Sprintf("%s/%s.jpg", coverDir, albumMBID)
	file, err := os.Create(coverPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	// Update album cover in database
	mb.db.Exec(`
		UPDATE albums SET cover_art_path = ?
		WHERE mbid = ?
	`, coverPath, albumMBID)

	return nil
}

// SearchRecording searches for recordings
func (mb *MusicBrainzClient) SearchRecording(query string, limit int) ([]Recording, error) {
	mb.enforceRateLimit()

	params := url.Values{
		"query":  {query},
		"limit":  {fmt.Sprintf("%d", limit)},
		"fmt":    {"json"},
	}

	url := fmt.Sprintf("%s/recording?%s", MusicBrainzAPI, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := mb.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Recordings []Recording `json:"recordings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Recordings, nil
}

// SearchArtist searches for artists
func (mb *MusicBrainzClient) SearchArtist(query string, limit int) ([]Artist, error) {
	mb.enforceRateLimit()

	params := url.Values{
		"query":  {query},
		"limit":  {fmt.Sprintf("%d", limit)},
		"fmt":    {"json"},
	}

	url := fmt.Sprintf("%s/artist?%s", MusicBrainzAPI, params.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := mb.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Artists []Artist `json:"artists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Artists, nil
}

// enforceRateLimit ensures we respect MusicBrainz rate limits
func (mb *MusicBrainzClient) enforceRateLimit() {
	elapsed := time.Since(mb.lastCall)
	if elapsed < MBRateLimit {
		time.Sleep(MBRateLimit - elapsed)
	}
	mb.lastCall = time.Now()
}

// generateFileHash generates SHA256 hash of a file
func (mb *MusicBrainzClient) generateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ProcessLibrary processes all untagged tracks in the library
func (mb *MusicBrainzClient) ProcessLibrary() error {
	rows, err := mb.db.Query(`
		SELECT id, file_path FROM tracks
		WHERE mbid IS NULL OR mbid = ''
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var trackID int
		var filePath string

		if err := rows.Scan(&trackID, &filePath); err != nil {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(filePath); err != nil {
			continue
		}

		// Try to tag the file
		if err := mb.TagFile(trackID); err != nil {
			// Log error but continue
			mb.db.Exec(`
				UPDATE tracks SET analyzed_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, trackID)
		}

		// Rate limit between tracks
		time.Sleep(2 * time.Second)
	}

	return nil
}

// SubmitFingerprint submits a fingerprint to AcoustID
func (mb *MusicBrainzClient) SubmitFingerprint(trackID int, mbid string) error {
	var filePath string
	err := mb.db.QueryRow(`
		SELECT file_path FROM tracks WHERE id = ?
	`, trackID).Scan(&filePath)
	if err != nil {
		return err
	}

	// Generate fingerprint
	fingerprint, duration, err := mb.generateFingerprint(filePath)
	if err != nil {
		return err
	}

	// Submit to AcoustID
	params := url.Values{
		"client":      {mb.acoustIDKey},
		"fingerprint": {fingerprint},
		"duration":    {fmt.Sprintf("%d", duration)},
		"mbid":        {mbid},
		"format":      {"json"},
	}

	resp, err := mb.httpClient.PostForm(AcoustIDAPI+"/submit", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Status != "ok" {
		return fmt.Errorf("submission failed")
	}

	// Store fingerprint in database
	mb.db.Exec(`
		UPDATE tracks SET acoustid_fingerprint = ?
		WHERE id = ?
	`, fingerprint, trackID)

	return nil
}
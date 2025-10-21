package protocols

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// AudioPlayer manages the current playback state and queue
type AudioPlayer struct {
	db       *database.Engine
	mu       sync.RWMutex

	// Queue management
	queue         []*Track
	queueVersion  int
	currentIndex  int

	// Playback state
	isPlaying     bool
	isPaused      bool
	volume        int
	elapsed       int // milliseconds

	// Playback modes
	repeat        bool
	random        bool
	single        bool
	consume       bool

	// Stats
	startTime     time.Time
	totalPlaytime int
}

type Track struct {
	ID           int
	FilePath     string
	Title        string
	Artist       string
	Album        string
	AlbumArtist  string
	Genre        string
	Duration     int // milliseconds
	Bitrate      int
	SampleRate   int
	Channels     int
	UpdatedAt    time.Time
}

type PlayerStatus struct {
	Volume          int
	Repeat          bool
	Random          bool
	Single          bool
	Consume         bool
	PlaylistVersion int
	PlaylistLength  int
	State           string // play, stop, pause
	Song            int
	SongID          int
	Elapsed         int
	Duration        int
	Bitrate         int
	SampleRate      int
	Bits            int
	Channels        int
}

type PlayerStats struct {
	Artists    int
	Albums     int
	Songs      int
	Uptime     int
	DBPlaytime int
	DBUpdate   int
	Playtime   int
}

type DirectoryItem struct {
	Path  string
	IsDir bool
}

func NewAudioPlayer(db *database.Engine) *AudioPlayer {
	return &AudioPlayer{
		db:           db,
		queue:        make([]*Track, 0),
		queueVersion: 0,
		currentIndex: -1,
		volume:       70,
		startTime:    time.Now(),
	}
}

// Playback control
func (p *AudioPlayer) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentIndex < 0 && len(p.queue) > 0 {
		p.currentIndex = 0
	}

	p.isPlaying = true
	p.isPaused = false
}

func (p *AudioPlayer) PlayPosition(position string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pos, err := strconv.Atoi(position)
	if err != nil || pos < 0 || pos >= len(p.queue) {
		return
	}

	p.currentIndex = pos
	p.isPlaying = true
	p.isPaused = false
	p.elapsed = 0
}

func (p *AudioPlayer) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isPaused = true
	p.isPlaying = false
}

func (p *AudioPlayer) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isPaused {
		p.isPlaying = true
		p.isPaused = false
	}
}

func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isPlaying = false
	p.isPaused = false
	p.elapsed = 0
}

func (p *AudioPlayer) Next() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return
	}

	if p.random {
		// Random mode - pick random track
		p.currentIndex = time.Now().Nanosecond() % len(p.queue)
	} else {
		p.currentIndex++
		if p.currentIndex >= len(p.queue) {
			if p.repeat {
				p.currentIndex = 0
			} else {
				p.currentIndex = len(p.queue) - 1
				p.isPlaying = false
			}
		}
	}

	if p.consume && p.currentIndex > 0 {
		// Remove previous track from queue
		p.queue = append(p.queue[:p.currentIndex-1], p.queue[p.currentIndex:]...)
		p.currentIndex--
		p.queueVersion++
	}

	p.elapsed = 0
}

func (p *AudioPlayer) Previous() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return
	}

	p.currentIndex--
	if p.currentIndex < 0 {
		if p.repeat {
			p.currentIndex = len(p.queue) - 1
		} else {
			p.currentIndex = 0
		}
	}

	p.elapsed = 0
}

// Queue management - ALWAYS adds to queue (never destroys)
func (p *AudioPlayer) AddToQueue(uri string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Query database for track
	track := p.loadTrack(uri)
	if track != nil {
		p.queue = append(p.queue, track)
		p.queueVersion++
	}
}

func (p *AudioPlayer) AddToQueueWithID(uri string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	track := p.loadTrack(uri)
	if track != nil {
		p.queue = append(p.queue, track)
		p.queueVersion++
		return track.ID
	}
	return -1
}

func (p *AudioPlayer) ClearQueue() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.queue = make([]*Track, 0)
	p.currentIndex = -1
	p.queueVersion++
	p.isPlaying = false
	p.isPaused = false
}

func (p *AudioPlayer) DeleteFromQueue(position string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pos, err := strconv.Atoi(position)
	if err != nil || pos < 0 || pos >= len(p.queue) {
		return
	}

	p.queue = append(p.queue[:pos], p.queue[pos+1:]...)

	if p.currentIndex == pos {
		p.isPlaying = false
	} else if p.currentIndex > pos {
		p.currentIndex--
	}

	p.queueVersion++
}

func (p *AudioPlayer) MoveInQueue(from, to string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fromPos, err1 := strconv.Atoi(from)
	toPos, err2 := strconv.Atoi(to)

	if err1 != nil || err2 != nil || fromPos < 0 || fromPos >= len(p.queue) || toPos < 0 || toPos >= len(p.queue) {
		return
	}

	track := p.queue[fromPos]

	// Remove from old position
	p.queue = append(p.queue[:fromPos], p.queue[fromPos+1:]...)

	// Insert at new position
	p.queue = append(p.queue[:toPos], append([]*Track{track}, p.queue[toPos:]...)...)

	// Adjust current index
	if p.currentIndex == fromPos {
		p.currentIndex = toPos
	} else if fromPos < p.currentIndex && toPos >= p.currentIndex {
		p.currentIndex--
	} else if fromPos > p.currentIndex && toPos <= p.currentIndex {
		p.currentIndex++
	}

	p.queueVersion++
}

// Queue info
func (p *AudioPlayer) GetQueue() []*Track {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.queue
}

func (p *AudioPlayer) GetQueuePosition() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentIndex
}

// Database queries
func (p *AudioPlayer) FindTracks(tag, value string) []*Track {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tracks []*Track

	query := fmt.Sprintf(`
		SELECT id, file_path, title, artist, album, album_artist, genre,
		       duration, bitrate, sample_rate, channels, updated_at
		FROM tracks
		WHERE %s = ?
		LIMIT 1000
	`, tag)

	rows, err := p.db.Query(query, value)
	if err != nil {
		return tracks
	}
	defer rows.Close()

	for rows.Next() {
		track := &Track{}
		rows.Scan(&track.ID, &track.FilePath, &track.Title, &track.Artist,
			&track.Album, &track.AlbumArtist, &track.Genre, &track.Duration,
			&track.Bitrate, &track.SampleRate, &track.Channels, &track.UpdatedAt)
		tracks = append(tracks, track)
	}

	return tracks
}

func (p *AudioPlayer) ListValues(listType string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var values []string
	var query string

	switch listType {
	case "artist":
		query = "SELECT DISTINCT artist FROM tracks WHERE artist IS NOT NULL ORDER BY artist"
	case "album":
		query = "SELECT DISTINCT album FROM tracks WHERE album IS NOT NULL ORDER BY album"
	case "genre":
		query = "SELECT DISTINCT genre FROM tracks WHERE genre IS NOT NULL ORDER BY genre"
	default:
		return values
	}

	rows, err := p.db.Query(query)
	if err != nil {
		return values
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		rows.Scan(&value)
		values = append(values, value)
	}

	return values
}

func (p *AudioPlayer) GetAllTracks() []*Track {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tracks []*Track

	rows, err := p.db.Query(`
		SELECT id, file_path, title, artist, album, album_artist, genre,
		       duration, bitrate, sample_rate, channels, updated_at
		FROM tracks
		ORDER BY artist, album, track_number
		LIMIT 10000
	`)
	if err != nil {
		return tracks
	}
	defer rows.Close()

	for rows.Next() {
		track := &Track{}
		rows.Scan(&track.ID, &track.FilePath, &track.Title, &track.Artist,
			&track.Album, &track.AlbumArtist, &track.Genre, &track.Duration,
			&track.Bitrate, &track.SampleRate, &track.Channels, &track.UpdatedAt)
		tracks = append(tracks, track)
	}

	return tracks
}

func (p *AudioPlayer) ListDirectory(path string) []DirectoryItem {
	// Simplified implementation - returns all tracks
	items := []DirectoryItem{}

	tracks := p.GetAllTracks()
	for _, track := range tracks {
		items = append(items, DirectoryItem{
			Path:  track.FilePath,
			IsDir: false,
		})
	}

	return items
}

// Status and stats
func (p *AudioPlayer) GetStatus() *PlayerStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state := "stop"
	if p.isPlaying {
		state = "play"
	} else if p.isPaused {
		state = "pause"
	}

	status := &PlayerStatus{
		Volume:          p.volume,
		Repeat:          p.repeat,
		Random:          p.random,
		Single:          p.single,
		Consume:         p.consume,
		PlaylistVersion: p.queueVersion,
		PlaylistLength:  len(p.queue),
		State:           state,
		Song:            p.currentIndex,
		SongID:          -1,
		Elapsed:         p.elapsed,
		SampleRate:      44100,
		Bits:            16,
		Channels:        2,
	}

	if p.currentIndex >= 0 && p.currentIndex < len(p.queue) {
		track := p.queue[p.currentIndex]
		status.SongID = track.ID
		status.Duration = track.Duration / 1000
		status.Bitrate = track.Bitrate
		if track.SampleRate > 0 {
			status.SampleRate = track.SampleRate
		}
		if track.Channels > 0 {
			status.Channels = track.Channels
		}
	}

	return status
}

func (p *AudioPlayer) GetStats() *PlayerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := &PlayerStats{
		Uptime:   int(time.Since(p.startTime).Seconds()),
		Playtime: p.totalPlaytime,
	}

	// Query database for counts
	p.db.QueryRow("SELECT COUNT(DISTINCT artist) FROM tracks").Scan(&stats.Artists)
	p.db.QueryRow("SELECT COUNT(DISTINCT album) FROM tracks").Scan(&stats.Albums)
	p.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&stats.Songs)
	p.db.QueryRow("SELECT COALESCE(SUM(duration), 0) FROM tracks").Scan(&stats.DBPlaytime)

	stats.DBPlaytime /= 1000 // Convert to seconds
	stats.DBUpdate = int(time.Now().Unix())

	return stats
}

func (p *AudioPlayer) GetCurrentTrack() *Track {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.currentIndex >= 0 && p.currentIndex < len(p.queue) {
		return p.queue[p.currentIndex]
	}
	return nil
}

// Volume control
func (p *AudioPlayer) SetVolume(volumeStr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	volume, err := strconv.Atoi(volumeStr)
	if err != nil || volume < 0 || volume > 100 {
		return
	}

	p.volume = volume
}

// Internal helper
func (p *AudioPlayer) loadTrack(uri string) *Track {
	track := &Track{}

	err := p.db.QueryRow(`
		SELECT id, file_path, title, artist, album, album_artist, genre,
		       duration, bitrate, sample_rate, channels, updated_at
		FROM tracks
		WHERE file_path = ? OR id = ?
		LIMIT 1
	`, uri, uri).Scan(&track.ID, &track.FilePath, &track.Title, &track.Artist,
		&track.Album, &track.AlbumArtist, &track.Genre, &track.Duration,
		&track.Bitrate, &track.SampleRate, &track.Channels, &track.UpdatedAt)

	if err != nil {
		// If not found in DB, create a basic track entry
		track.FilePath = uri
		track.Title = uri
		track.ID = int(time.Now().UnixNano())
		return track
	}

	return track
}
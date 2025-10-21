package media

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// AutoDJ implements an intelligent automatic DJ system
type AutoDJ struct {
	db         *database.Engine
	transcoder *Transcoder

	// Current state
	currentTrack  *Track
	nextTrack     *Track
	queue         []*Track
	history       []*Track

	// Configuration
	config        *AutoDJConfig
	rules         []AutoDJRule

	// Control
	isRunning     bool
	isPaused      bool
	mu            sync.RWMutex
	stopChan      chan struct{}

	// Audio processing
	crossfader    *Crossfader
	beatMatcher   *BeatMatcher
	harmonicMixer *HarmonicMixer
}

// AutoDJConfig holds AutoDJ configuration
type AutoDJConfig struct {
	// Playback settings
	CrossfadeDuration int    `json:"crossfade_duration"` // seconds (default: 5)
	SilenceThreshold  float64 `json:"silence_threshold"`   // dB (default: -40)

	// Selection algorithm
	Algorithm         string  `json:"algorithm"`          // random, weighted, smart (default: smart)
	RepeatArtistHours float64 `json:"repeat_artist_hours"` // Hours before repeating artist (default: 0.5)
	RepeatTrackHours  float64 `json:"repeat_track_hours"`  // Hours before repeating track (default: 2)

	// Mood and energy
	MaintainEnergy    bool    `json:"maintain_energy"`     // Try to maintain energy levels
	EnergyVariation   float64 `json:"energy_variation"`    // Allowed energy variation (0-1)

	// Genre control
	MixGenres         bool    `json:"mix_genres"`          // Allow genre mixing
	GenreTransition   bool    `json:"genre_transition"`    // Smooth genre transitions

	// Time-based rules
	TimeBasedRules    bool    `json:"time_based_rules"`    // Enable time-based selection

	// Technical
	EnableBeatMatch   bool    `json:"enable_beat_match"`   // BPM matching
	EnableHarmonic    bool    `json:"enable_harmonic"`     // Harmonic mixing
	BPMTolerance      float64 `json:"bpm_tolerance"`       // BPM tolerance percentage

	// Sources
	Sources           []string `json:"sources"`             // music sources to use
	ExcludeTags       []string `json:"exclude_tags"`        // Tags to exclude
	RequireTags       []string `json:"require_tags"`        // Required tags
}

// AutoDJRule represents a rule for track selection
type AutoDJRule struct {
	Type       string      `json:"type"`        // time, sequence, energy, genre
	Priority   int         `json:"priority"`
	Condition  interface{} `json:"condition"`
	Action     string      `json:"action"`
	Parameters interface{} `json:"parameters"`
}

// Track represents a track for AutoDJ
type Track struct {
	ID           int
	FilePath     string
	Title        string
	Artist       string
	Album        string
	Genre        string
	Duration     int // milliseconds
	BPM          float64
	Key          string
	Energy       float64
	Mood         string
	Color        string // For visual theming
	Tags         []string
	Rating       int
	PlayCount    int
	LastPlayed   time.Time

	// Audio analysis
	IntroEnd     int // ms - where intro ends
	OutroStart   int // ms - where outro starts
	FirstBeat    int // ms - first beat position
	LastBeat     int // ms - last beat position
	PeakLoudness float64
}

// Crossfader handles crossfading between tracks
type Crossfader struct {
	duration   int // seconds
	curveType  string // linear, exponential, logarithmic, s-curve
}

// BeatMatcher handles BPM matching
type BeatMatcher struct {
	tolerance  float64 // percentage
	pitchShift bool    // Allow pitch shifting for matching
}

// HarmonicMixer handles harmonic mixing using Camelot Wheel
type HarmonicMixer struct {
	// Camelot Wheel mapping
	camelotWheel map[string][]string
}

// NewAutoDJ creates a new AutoDJ instance
func NewAutoDJ(db *database.Engine, transcoder *Transcoder) *AutoDJ {
	return &AutoDJ{
		db:         db,
		transcoder: transcoder,
		queue:      make([]*Track, 0),
		history:    make([]*Track, 0),
		config: &AutoDJConfig{
			CrossfadeDuration: 5,
			SilenceThreshold:  -40,
			Algorithm:         "smart",
			RepeatArtistHours: 0.5,
			RepeatTrackHours:  2,
			MaintainEnergy:    true,
			EnergyVariation:   0.3,
			MixGenres:         true,
			GenreTransition:   true,
			TimeBasedRules:    true,
			EnableBeatMatch:   false,
			EnableHarmonic:    false,
			BPMTolerance:      5.0,
		},
		crossfader: &Crossfader{
			duration:  5,
			curveType: "s-curve",
		},
		beatMatcher: &BeatMatcher{
			tolerance:  5.0,
			pitchShift: false,
		},
		harmonicMixer: NewHarmonicMixer(),
		stopChan: make(chan struct{}),
	}
}

// NewHarmonicMixer creates a new harmonic mixer with Camelot Wheel
func NewHarmonicMixer() *HarmonicMixer {
	return &HarmonicMixer{
		camelotWheel: map[string][]string{
			// Major keys (B)
			"1B":  []string{"1B", "12B", "2B", "1A"},   // B Major
			"2B":  []string{"2B", "1B", "3B", "2A"},    // F# Major
			"3B":  []string{"3B", "2B", "4B", "3A"},    // Db Major
			"4B":  []string{"4B", "3B", "5B", "4A"},    // Ab Major
			"5B":  []string{"5B", "4B", "6B", "5A"},    // Eb Major
			"6B":  []string{"6B", "5B", "7B", "6A"},    // Bb Major
			"7B":  []string{"7B", "6B", "8B", "7A"},    // F Major
			"8B":  []string{"8B", "7B", "9B", "8A"},    // C Major
			"9B":  []string{"9B", "8B", "10B", "9A"},   // G Major
			"10B": []string{"10B", "9B", "11B", "10A"}, // D Major
			"11B": []string{"11B", "10B", "12B", "11A"},// A Major
			"12B": []string{"12B", "11B", "1B", "12A"}, // E Major

			// Minor keys (A)
			"1A":  []string{"1A", "12A", "2A", "1B"},   // Ab minor
			"2A":  []string{"2A", "1A", "3A", "2B"},    // Eb minor
			"3A":  []string{"3A", "2A", "4A", "3B"},    // Bb minor
			"4A":  []string{"4A", "3A", "5A", "4B"},    // F minor
			"5A":  []string{"5A", "4A", "6A", "5B"},    // C minor
			"6A":  []string{"6A", "5A", "7A", "6B"},    // G minor
			"7A":  []string{"7A", "6A", "8A", "7B"},    // D minor
			"8A":  []string{"8A", "7A", "9A", "8B"},    // A minor
			"9A":  []string{"9A", "8A", "10A", "9B"},   // E minor
			"10A": []string{"10A", "9A", "11A", "10B"}, // B minor
			"11A": []string{"11A", "10A", "12A", "11B"},// F# minor
			"12A": []string{"12A", "11A", "1A", "12B"}, // Db minor
		},
	}
}

// Start starts the AutoDJ
func (dj *AutoDJ) Start(mountPoint string) error {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	if dj.isRunning {
		return fmt.Errorf("AutoDJ already running")
	}

	dj.isRunning = true
	dj.isPaused = false

	// Load configuration from database
	dj.loadConfig()
	dj.loadRules()

	// Start the DJ loop
	go dj.run(mountPoint)

	log.Printf("AutoDJ started for mount point: %s", mountPoint)
	return nil
}

// Stop stops the AutoDJ
func (dj *AutoDJ) Stop() {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	if !dj.isRunning {
		return
	}

	dj.isRunning = false
	close(dj.stopChan)

	log.Println("AutoDJ stopped")
}

// Pause pauses the AutoDJ
func (dj *AutoDJ) Pause() {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	dj.isPaused = true
}

// Resume resumes the AutoDJ
func (dj *AutoDJ) Resume() {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	dj.isPaused = false
}

// run is the main AutoDJ loop
func (dj *AutoDJ) run(mountPoint string) {
	for {
		select {
		case <-dj.stopChan:
			return
		default:
			if dj.isPaused {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Ensure queue has tracks
			if len(dj.queue) < 5 {
				dj.fillQueue()
			}

			// Get next track
			track := dj.getNextTrack()
			if track == nil {
				log.Println("AutoDJ: No tracks available")
				time.Sleep(5 * time.Second)
				continue
			}

			// Play track
			dj.playTrack(track, mountPoint)

			// Add to history
			dj.addToHistory(track)
		}
	}
}

// fillQueue fills the queue with tracks
func (dj *AutoDJ) fillQueue() {
	// Get candidate tracks
	candidates := dj.getCandidateTracks()

	// Score and rank candidates
	scoredTracks := dj.scoreTracks(candidates)

	// Add top tracks to queue
	for i := 0; i < 10 && i < len(scoredTracks); i++ {
		dj.queue = append(dj.queue, scoredTracks[i])
	}
}

// getCandidateTracks gets candidate tracks from database
func (dj *AutoDJ) getCandidateTracks() []*Track {
	var tracks []*Track

	// Build query based on configuration
	query := `
		SELECT id, file_path, title, artist, album, genre,
		       duration, bpm, key, energy, mood, tags,
		       rating, play_count, last_played
		FROM tracks
		WHERE 1=1
	`

	args := []interface{}{}

	// Add time-based filtering
	if dj.config.RepeatTrackHours > 0 {
		cutoff := time.Now().Add(-time.Duration(dj.config.RepeatTrackHours) * time.Hour)
		query += " AND (last_played IS NULL OR last_played < ?)"
		args = append(args, cutoff)
	}

	// Add tag filtering
	if len(dj.config.RequireTags) > 0 {
		tagJSON, _ := json.Marshal(dj.config.RequireTags)
		query += " AND tags @> ?"
		args = append(args, string(tagJSON))
	}

	query += " ORDER BY RANDOM() LIMIT 100"

	rows, err := dj.db.Query(query, args...)
	if err != nil {
		log.Printf("AutoDJ: Failed to get candidate tracks: %v", err)
		return tracks
	}
	defer rows.Close()

	for rows.Next() {
		track := &Track{}
		var tagsJSON string
		var lastPlayed *time.Time

		err := rows.Scan(
			&track.ID, &track.FilePath, &track.Title, &track.Artist,
			&track.Album, &track.Genre, &track.Duration, &track.BPM,
			&track.Key, &track.Energy, &track.Mood, &tagsJSON,
			&track.Rating, &track.PlayCount, &lastPlayed,
		)

		if err != nil {
			continue
		}

		if lastPlayed != nil {
			track.LastPlayed = *lastPlayed
		}

		json.Unmarshal([]byte(tagsJSON), &track.Tags)
		tracks = append(tracks, track)
	}

	return tracks
}

// scoreTracks scores and ranks tracks
func (dj *AutoDJ) scoreTracks(tracks []*Track) []*Track {
	type scoredTrack struct {
		track *Track
		score float64
	}

	scored := make([]scoredTrack, 0, len(tracks))

	for _, track := range tracks {
		score := dj.calculateTrackScore(track)
		scored = append(scored, scoredTrack{track, score})
	}

	// Sort by score descending
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Extract sorted tracks
	result := make([]*Track, len(scored))
	for i, st := range scored {
		result[i] = st.track
	}

	return result
}

// calculateTrackScore calculates a score for a track
func (dj *AutoDJ) calculateTrackScore(track *Track) float64 {
	score := 100.0

	// Rating bonus (0-5 stars = 0-50 points)
	score += float64(track.Rating) * 10

	// Popularity penalty (avoid overplaying popular tracks)
	if track.PlayCount > 100 {
		score -= math.Log10(float64(track.PlayCount)) * 5
	}

	// Freshness bonus (tracks not played recently)
	if !track.LastPlayed.IsZero() {
		hoursSince := time.Since(track.LastPlayed).Hours()
		score += math.Min(hoursSince/24, 20) // Max 20 points for freshness
	} else {
		score += 25 // Never played bonus
	}

	// Energy matching
	if dj.config.MaintainEnergy && dj.currentTrack != nil {
		energyDiff := math.Abs(track.Energy - dj.currentTrack.Energy)
		if energyDiff <= dj.config.EnergyVariation {
			score += (1 - energyDiff/dj.config.EnergyVariation) * 20
		} else {
			score -= energyDiff * 10
		}
	}

	// BPM matching
	if dj.config.EnableBeatMatch && dj.currentTrack != nil && track.BPM > 0 {
		bpmDiff := math.Abs(track.BPM - dj.currentTrack.BPM)
		bpmPercent := bpmDiff / dj.currentTrack.BPM * 100

		if bpmPercent <= dj.config.BPMTolerance {
			score += (1 - bpmPercent/dj.config.BPMTolerance) * 15
		} else if bpmPercent <= dj.config.BPMTolerance*2 {
			// Half/double tempo is OK
			score += 5
		} else {
			score -= bpmPercent
		}
	}

	// Harmonic mixing
	if dj.config.EnableHarmonic && dj.currentTrack != nil && track.Key != "" {
		if dj.harmonicMixer.isCompatible(dj.currentTrack.Key, track.Key) {
			score += 25
		} else {
			score -= 10
		}
	}

	// Genre compatibility
	if dj.currentTrack != nil {
		if track.Genre == dj.currentTrack.Genre {
			score += 10
		} else if dj.config.MixGenres {
			// Allow but slightly penalize genre changes
			score -= 5
		} else {
			// Heavily penalize genre changes if not mixing
			score -= 30
		}
	}

	// Time-based rules
	if dj.config.TimeBasedRules {
		score += dj.applyTimeRules(track)
	}

	// Apply custom rules
	for _, rule := range dj.rules {
		score += dj.applyRule(track, rule)
	}

	// Artist repetition check
	if dj.isArtistInRecentHistory(track.Artist) {
		score -= 50
	}

	return math.Max(0, score)
}

// isCompatible checks if two keys are harmonically compatible
func (hm *HarmonicMixer) isCompatible(key1, key2 string) bool {
	compatible, exists := hm.camelotWheel[key1]
	if !exists {
		return false
	}

	for _, k := range compatible {
		if k == key2 {
			return true
		}
	}

	return false
}

// applyTimeRules applies time-based scoring rules
func (dj *AutoDJ) applyTimeRules(track *Track) float64 {
	score := 0.0
	hour := time.Now().Hour()

	// Morning (6-10): Upbeat, energetic
	if hour >= 6 && hour < 10 {
		if track.Energy > 0.7 {
			score += 15
		}
		if track.Mood == "happy" || track.Mood == "energetic" {
			score += 10
		}
	}

	// Midday (10-14): Varied
	// No special rules

	// Afternoon (14-18): Maintain energy
	if hour >= 14 && hour < 18 {
		if track.Energy > 0.5 && track.Energy < 0.8 {
			score += 10
		}
	}

	// Evening (18-22): Relaxing down
	if hour >= 18 && hour < 22 {
		if track.Energy < 0.6 {
			score += 10
		}
		if track.Mood == "relaxed" || track.Mood == "chill" {
			score += 10
		}
	}

	// Night (22-6): Chill, ambient
	if hour >= 22 || hour < 6 {
		if track.Energy < 0.4 {
			score += 15
		}
		if track.Mood == "ambient" || track.Mood == "chill" {
			score += 15
		}
		if track.BPM < 100 {
			score += 10
		}
	}

	return score
}

// applyRule applies a custom rule
func (dj *AutoDJ) applyRule(track *Track, rule AutoDJRule) float64 {
	// Simplified rule application
	// In production, this would be more sophisticated
	return 0
}

// isArtistInRecentHistory checks if artist was recently played
func (dj *AutoDJ) isArtistInRecentHistory(artist string) bool {
	cutoff := time.Now().Add(-time.Duration(dj.config.RepeatArtistHours) * time.Hour)

	for _, track := range dj.history {
		if track.Artist == artist && track.LastPlayed.After(cutoff) {
			return true
		}
	}

	return false
}

// getNextTrack gets the next track from queue
func (dj *AutoDJ) getNextTrack() *Track {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	if len(dj.queue) == 0 {
		return nil
	}

	track := dj.queue[0]
	dj.queue = dj.queue[1:]

	return track
}

// playTrack plays a track
func (dj *AutoDJ) playTrack(track *Track, mountPoint string) {
	dj.mu.Lock()
	dj.currentTrack = track
	dj.mu.Unlock()

	// Update now playing in database
	dj.db.Exec(`
		UPDATE broadcasts SET
			current_track = ?,
			metadata_url = ?
		WHERE mount_point = ?
	`, fmt.Sprintf("%s - %s", track.Artist, track.Title),
	   fmt.Sprintf("/api/v1/track/%d", track.ID),
	   mountPoint)

	// Update track statistics
	dj.db.Exec(`
		UPDATE tracks SET
			play_count = play_count + 1,
			last_played = CURRENT_TIMESTAMP
		WHERE id = ?
	`, track.ID)

	// Record in playback history
	dj.db.Exec(`
		INSERT INTO playback_history (user_id, track_id, source, track_duration)
		VALUES (NULL, ?, 'autodj', ?)
	`, track.ID, track.Duration)

	// Calculate actual play duration with crossfade
	playDuration := track.Duration
	if dj.config.CrossfadeDuration > 0 {
		playDuration -= dj.config.CrossfadeDuration * 1000
	}

	// Simulate playing (in production, this would stream audio)
	time.Sleep(time.Duration(playDuration) * time.Millisecond)

	// Prepare next track during crossfade
	if dj.config.CrossfadeDuration > 0 {
		go dj.prepareCrossfade(track, mountPoint)
	}
}

// prepareCrossfade prepares crossfade to next track
func (dj *AutoDJ) prepareCrossfade(currentTrack *Track, mountPoint string) {
	// Get next track
	nextTrack := dj.queue[0]
	if nextTrack == nil {
		return
	}

	dj.mu.Lock()
	dj.nextTrack = nextTrack
	dj.mu.Unlock()

	// In production, this would:
	// 1. Start decoding next track
	// 2. Analyze beat grid for alignment
	// 3. Prepare crossfade envelope
	// 4. Begin mixing at appropriate point

	log.Printf("AutoDJ: Preparing crossfade from '%s' to '%s'",
		currentTrack.Title, nextTrack.Title)
}

// addToHistory adds track to history
func (dj *AutoDJ) addToHistory(track *Track) {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	track.LastPlayed = time.Now()
	dj.history = append(dj.history, track)

	// Keep only last 100 tracks in memory
	if len(dj.history) > 100 {
		dj.history = dj.history[1:]
	}
}

// loadConfig loads configuration from database
func (dj *AutoDJ) loadConfig() {
	// Load settings from database
	if val, err := dj.db.GetSetting("autodj.crossfade_duration"); err == nil {
		fmt.Sscanf(val, "%d", &dj.config.CrossfadeDuration)
	}

	if val, err := dj.db.GetSetting("autodj.algorithm"); err == nil {
		dj.config.Algorithm = val
	}

	if val, err := dj.db.GetSetting("autodj.repeat_artist_hours"); err == nil {
		fmt.Sscanf(val, "%f", &dj.config.RepeatArtistHours)
	}

	if val, err := dj.db.GetSetting("autodj.repeat_track_hours"); err == nil {
		fmt.Sscanf(val, "%f", &dj.config.RepeatTrackHours)
	}

	if val, err := dj.db.GetSetting("autodj.enable_beat_match"); err == nil {
		dj.config.EnableBeatMatch = val == "true"
	}

	if val, err := dj.db.GetSetting("autodj.enable_harmonic"); err == nil {
		dj.config.EnableHarmonic = val == "true"
	}
}

// loadRules loads AutoDJ rules from database
func (dj *AutoDJ) loadRules() {
	// In production, load from database
	// For now, use default rules

	dj.rules = []AutoDJRule{
		{
			Type:     "energy",
			Priority: 10,
			Action:   "boost_energy_morning",
		},
		{
			Type:     "genre",
			Priority: 5,
			Action:   "smooth_genre_transition",
		},
	}
}

// GetStatus returns current AutoDJ status
func (dj *AutoDJ) GetStatus() map[string]interface{} {
	dj.mu.RLock()
	defer dj.mu.RUnlock()

	status := map[string]interface{}{
		"running": dj.isRunning,
		"paused":  dj.isPaused,
		"queue_size": len(dj.queue),
		"history_size": len(dj.history),
	}

	if dj.currentTrack != nil {
		status["current_track"] = map[string]interface{}{
			"id":     dj.currentTrack.ID,
			"title":  dj.currentTrack.Title,
			"artist": dj.currentTrack.Artist,
			"album":  dj.currentTrack.Album,
		}
	}

	if dj.nextTrack != nil {
		status["next_track"] = map[string]interface{}{
			"id":     dj.nextTrack.ID,
			"title":  dj.nextTrack.Title,
			"artist": dj.nextTrack.Artist,
		}
	}

	return status
}

// SetConfig updates AutoDJ configuration
func (dj *AutoDJ) SetConfig(config *AutoDJConfig) {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	dj.config = config

	// Update crossfader
	dj.crossfader.duration = config.CrossfadeDuration

	// Update beat matcher
	dj.beatMatcher.tolerance = config.BPMTolerance

	// Save to database
	dj.db.SetSetting("autodj.crossfade_duration", fmt.Sprintf("%d", config.CrossfadeDuration), nil)
	dj.db.SetSetting("autodj.algorithm", config.Algorithm, nil)
	dj.db.SetSetting("autodj.repeat_artist_hours", fmt.Sprintf("%f", config.RepeatArtistHours), nil)
	dj.db.SetSetting("autodj.repeat_track_hours", fmt.Sprintf("%f", config.RepeatTrackHours), nil)
	dj.db.SetSetting("autodj.enable_beat_match", fmt.Sprintf("%t", config.EnableBeatMatch), nil)
	dj.db.SetSetting("autodj.enable_harmonic", fmt.Sprintf("%t", config.EnableHarmonic), nil)
}

// AddRule adds a new AutoDJ rule
func (dj *AutoDJ) AddRule(rule AutoDJRule) {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	dj.rules = append(dj.rules, rule)

	// Sort rules by priority
	for i := 0; i < len(dj.rules)-1; i++ {
		for j := i + 1; j < len(dj.rules); j++ {
			if dj.rules[j].Priority > dj.rules[i].Priority {
				dj.rules[i], dj.rules[j] = dj.rules[j], dj.rules[i]
			}
		}
	}
}

// ClearQueue clears the current queue
func (dj *AutoDJ) ClearQueue() {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	dj.queue = make([]*Track, 0)
}

// SkipTrack skips the current track
func (dj *AutoDJ) SkipTrack() {
	dj.mu.Lock()
	defer dj.mu.Unlock()

	// In production, this would trigger immediate transition
	log.Println("AutoDJ: Skipping current track")
}
// Package model - Track data model
// See IDEA.md Data Models - Tracks
package model

import "time"

// Track represents an audio track in the library
type Track struct {
	ID           int64     `json:"id"`
	FilePath     string    `json:"-"`           // Internal path, not exposed
	FileHash     string    `json:"-"`           // SHA256 for deduplication
	UserID       int64     `json:"user_id"`     // NULL/0 for global tracks
	IsGlobal     bool      `json:"is_global"`   // From global directories

	// Basic metadata
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`

	// Extended metadata
	Genre     string `json:"genre"`
	Year      int    `json:"year"`
	Composer  string `json:"composer,omitempty"`
	Performer string `json:"performer,omitempty"`
	Conductor string `json:"conductor,omitempty"`

	// Track information
	TrackNumber int `json:"track_number"`
	TrackTotal  int `json:"track_total"`
	DiscNumber  int `json:"disc_number"`
	DiscTotal   int `json:"disc_total"`

	// Technical metadata
	DurationMS    int64  `json:"duration_ms"` // Milliseconds
	Bitrate       int    `json:"bitrate"`     // kbps
	SampleRate    int    `json:"sample_rate"` // Hz
	Channels      int    `json:"channels"`
	BitsPerSample int    `json:"bits_per_sample,omitempty"`
	Codec         string `json:"codec"`
	FileType      string `json:"file_type"`
	FileSize      int64  `json:"file_size"`

	// MusicBrainz integration
	MBID                 string `json:"mbid,omitempty"`
	AlbumMBID            string `json:"album_mbid,omitempty"`
	ArtistMBID           string `json:"artist_mbid,omitempty"`
	AcoustIDFingerprint  string `json:"-"` // Long string, not serialized

	// Additional metadata
	ISRC          string `json:"isrc,omitempty"`
	Label         string `json:"label,omitempty"`
	Copyright     string `json:"copyright,omitempty"`
	Lyrics        string `json:"lyrics,omitempty"`
	Comment       string `json:"comment,omitempty"`

	// User metadata
	Rating     int      `json:"rating"` // 0-5
	Tags       []string `json:"tags,omitempty"`

	// Statistics
	PlayCount  int64      `json:"play_count"`
	SkipCount  int64      `json:"skip_count"`
	LastPlayed *time.Time `json:"last_played,omitempty"`

	// Analysis data (optional)
	ReplayGainTrackGain float64 `json:"replaygain_track_gain,omitempty"`
	ReplayGainTrackPeak float64 `json:"replaygain_track_peak,omitempty"`
	ReplayGainAlbumGain float64 `json:"replaygain_album_gain,omitempty"`
	ReplayGainAlbumPeak float64 `json:"replaygain_album_peak,omitempty"`
	BPM                 float64 `json:"bpm,omitempty"`
	Key                 string  `json:"key,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	AnalyzedAt *time.Time `json:"analyzed_at,omitempty"`

	// Cover art (computed, not stored)
	CoverArtURL string `json:"cover_art_url,omitempty"`
}

// TrackSummary is a lightweight track for listings
type TrackSummary struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	DurationMS  int64  `json:"duration_ms"`
	Rating      int    `json:"rating"`
	PlayCount   int64  `json:"play_count"`
	CoverArtURL string `json:"cover_art_url,omitempty"`
}

// ToSummary converts a full track to a summary
func (t *Track) ToSummary() TrackSummary {
	return TrackSummary{
		ID:          t.ID,
		Title:       t.Title,
		Artist:      t.Artist,
		Album:       t.Album,
		DurationMS:  t.DurationMS,
		Rating:      t.Rating,
		PlayCount:   t.PlayCount,
		CoverArtURL: t.CoverArtURL,
	}
}

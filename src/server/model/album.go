// Package model - Album data model
// See IDEA.md Data Models - Albums
package model

import "time"

// Album represents an album grouping for tracks
type Album struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	AlbumArtist string    `json:"album_artist"`
	Year        int       `json:"year"`
	Genre       string    `json:"genre"`
	CoverArtURL string    `json:"cover_art_url,omitempty"`
	// Internal path
	CoverArtPath string   `json:"-"`
	MBID        string    `json:"mbid,omitempty"`
	TotalTracks int       `json:"total_tracks"`
	TotalDiscs  int       `json:"total_discs"`
	Label       string    `json:"label,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AlbumSummary is a lightweight album for listings
type AlbumSummary struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Year        int    `json:"year"`
	TotalTracks int    `json:"total_tracks"`
	CoverArtURL string `json:"cover_art_url,omitempty"`
}

// ToSummary converts a full album to a summary
func (a *Album) ToSummary() AlbumSummary {
	return AlbumSummary{
		ID:          a.ID,
		Title:       a.Title,
		Artist:      a.Artist,
		Year:        a.Year,
		TotalTracks: a.TotalTracks,
		CoverArtURL: a.CoverArtURL,
	}
}

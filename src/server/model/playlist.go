// Package model - Playlist data model
// See IDEA.md Data Models - Playlists
package model

import "time"

// Playlist represents a user-created track collection
type Playlist struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	CoverImage      string    `json:"cover_image,omitempty"`
	IsPublic        bool      `json:"is_public"`
	IsCollaborative bool      `json:"is_collaborative"`
	IsSmart         bool      `json:"is_smart"`
	SmartCriteria   string    `json:"-"` // JSON for smart playlist rules
	SortOrder       string    `json:"sort_order"`
	PlayCount       int64     `json:"play_count"`
	FollowerCount   int64     `json:"follower_count"`
	DurationMS      int64     `json:"duration_ms"`
	TrackCount      int       `json:"track_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastPlayed      time.Time `json:"last_played,omitempty"`
}

// PlaylistTrack represents a track in a playlist with position
type PlaylistTrack struct {
	PlaylistID int64     `json:"playlist_id"`
	TrackID    int64     `json:"track_id"`
	Position   int       `json:"position"`
	AddedAt    time.Time `json:"added_at"`
	AddedBy    int64     `json:"added_by,omitempty"`
}

// PlaylistSummary is a lightweight playlist for listings
type PlaylistSummary struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	TrackCount int    `json:"track_count"`
	DurationMS int64  `json:"duration_ms"`
	CoverImage string `json:"cover_image,omitempty"`
	IsPublic   bool   `json:"is_public"`
}

// ToSummary converts a full playlist to a summary
func (p *Playlist) ToSummary() PlaylistSummary {
	return PlaylistSummary{
		ID:         p.ID,
		Name:       p.Name,
		TrackCount: p.TrackCount,
		DurationMS: p.DurationMS,
		CoverImage: p.CoverImage,
		IsPublic:   p.IsPublic,
	}
}

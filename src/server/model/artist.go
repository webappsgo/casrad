// Package model - Artist data model
// See IDEA.md Data Models - Artists
package model

import "time"

// Artist represents an artist entity
type Artist struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	SortName     string    `json:"sort_name"`
	MBID         string    `json:"mbid,omitempty"`
	Biography    string    `json:"biography,omitempty"`
	ImageURL     string    `json:"image_url,omitempty"`
	Website      string    `json:"website,omitempty"`
	Country      string    `json:"country,omitempty"`
	FormedYear   int       `json:"formed_year,omitempty"`
	DisbandedYear int      `json:"disbanded_year,omitempty"`
	Genre        string    `json:"genre,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ArtistSummary is a lightweight artist for listings
type ArtistSummary struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url,omitempty"`
}

// ToSummary converts a full artist to a summary
func (a *Artist) ToSummary() ArtistSummary {
	return ArtistSummary{
		ID:       a.ID,
		Name:     a.Name,
		ImageURL: a.ImageURL,
	}
}

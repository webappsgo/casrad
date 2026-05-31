// Package model - Audiobook models per IDEA.md
package model

import "time"

// Audiobook represents an audiobook
type Audiobook struct {
	ID              int64
	UserID          int64
	Title           string
	Author          string
	Narrator        string
	Series          string
	SeriesNumber    float64
	FilePath        string
	CoverPath       string
	ISBN            string
	Publisher       string
	PublishedDate   time.Time
	Language        string
	Description     string
	// seconds
	TotalDuration int
	// seconds
	CurrentPosition int
	CurrentChapter  int
	PlayCount       int
	Completed       bool
	CompletedAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AudiobookChapter represents a chapter in an audiobook
type AudiobookChapter struct {
	ID            int64
	AudiobookID   int64
	ChapterNumber int
	Title         string
	// seconds
	StartTime int
	// seconds
	EndTime int
	FilePath      string
}

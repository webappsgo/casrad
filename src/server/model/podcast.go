// Package model - Podcast models per IDEA.md
package model

import "time"

// Podcast represents a podcast subscription
type Podcast struct {
	ID              int64
	UserID          int64
	FeedURL         string
	Title           string
	Description     string
	Author          string
	ImageURL        string
	Website         string
	Language        string
	Category        string
	Explicit        bool
	StoragePath     string
	AutoDownload    bool
	DownloadQuality string
	MaxEpisodes     int
	RetentionDays   int
	IsActive        bool
	LastCheck       time.Time
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PodcastEpisode represents a single podcast episode
type PodcastEpisode struct {
	ID            int64
	PodcastID     int64
	GUID          string
	Title         string
	Description   string
	AudioURL      string
	WebsiteURL    string
	PublishedAt   time.Time
	Duration      int // seconds
	FileSize      int64
	FilePath      string
	PlayPosition  int // seconds
	IsPlayed      bool
	PlayedAt      time.Time
	IsDownloaded  bool
	DownloadedAt  time.Time
	DownloadError string
	CreatedAt     time.Time
}

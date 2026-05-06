// Package model - Storage models per IDEA.md
package model

import "time"

// UserStorage represents per-user storage configuration
type UserStorage struct {
	UserID              int64
	MusicPaths          string // JSON array of paths
	PodcastPath         string
	AudiobookPath       string
	RadioPath           string
	PlaylistPath        string
	RecordingPath       string
	TranscodePath       string
	QuotaMusicBytes     int64
	QuotaPodcastBytes   int64
	QuotaAudiobookBytes int64
	QuotaRecordingBytes int64
	QuotaOtherBytes     int64
	UsedMusicBytes      int64
	UsedPodcastBytes    int64
	UsedAudiobookBytes  int64
	UsedRecordingBytes  int64
	UsedOtherBytes      int64
	UpdatedAt           time.Time
}

// GlobalDirectory represents a global media directory
type GlobalDirectory struct {
	ID                int64
	Type              string // music, podcast, audiobook, playlist
	Path              string
	IsActive          bool
	IsPublic          bool
	ScanIntervalHours int
	AllowGuestAccess  bool
	AllowUserAccess   bool
	LastScan          time.Time
	FileCount         int
	TotalSizeBytes    int64
	CreatedAt         time.Time
}

// PlaybackHistory represents a play history entry
type PlaybackHistory struct {
	ID            int64
	UserID        int64
	TrackID       int64
	StartedAt     time.Time
	EndedAt       time.Time
	PlayDuration  int // seconds
	TrackDuration int // seconds
	Source        string
	SourceIP      string
	UserAgent     string
	Skipped       bool
	SkipPosition  int
	PlaylistID    int64
	BroadcastID   int64
}

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	ID                int64
	Name              string
	Schedule          string
	TaskType          string
	IsEnabled         bool
	Command           string
	Parameters        string
	LastRun           time.Time
	NextRun           time.Time
	LastStatus        string
	LastError         string
	RunCount          int64
	AverageDurationMS int64
	TimeoutSeconds    int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

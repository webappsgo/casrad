// Package model - Broadcast models per IDEA.md
package model

import "time"

// Broadcast represents a live audio stream/radio station (mount point)
type Broadcast struct {
	ID               int64
	MountPoint       string
	Type             string // live, autodj, relay, user
	Name             string
	Description      string
	Genre            string
	UserID           int64
	StreamKey        string
	Bitrate          int
	Format           string // mp3, aac, opus, ogg, flac
	Channels         int
	SampleRate       int
	IsPublic         bool
	RequiresAuth     bool
	MaxListeners     int
	IsActive         bool
	IsEnabled        bool
	ListenersCurrent int
	ListenersPeak    int
	ListenersTotal   int64
	BytesSentTotal   int64
	CurrentTrack     string
	StartedAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

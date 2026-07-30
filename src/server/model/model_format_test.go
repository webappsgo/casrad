// Package model — Tests for ToSummary methods on Album, Artist, Track, Playlist.
// These are pure functions: zero-value, populated, and boundary inputs.
package model

import (
	"testing"
	"time"
)

// --- Album.ToSummary ---

func TestAlbumToSummaryZeroValue(t *testing.T) {
	t.Parallel()
	a := &Album{}
	s := a.ToSummary()
	if s.ID != 0 {
		t.Errorf("ID = %d, want 0", s.ID)
	}
	if s.Title != "" {
		t.Errorf("Title = %q, want empty", s.Title)
	}
	if s.Artist != "" {
		t.Errorf("Artist = %q, want empty", s.Artist)
	}
	if s.Year != 0 {
		t.Errorf("Year = %d, want 0", s.Year)
	}
	if s.TotalTracks != 0 {
		t.Errorf("TotalTracks = %d, want 0", s.TotalTracks)
	}
	if s.CoverArtURL != "" {
		t.Errorf("CoverArtURL = %q, want empty", s.CoverArtURL)
	}
}

func TestAlbumToSummaryPopulated(t *testing.T) {
	t.Parallel()
	a := &Album{
		ID:           42,
		Title:        "Dark Side of the Moon",
		Artist:       "Pink Floyd",
		Year:         1973,
		TotalTracks:  10,
		CoverArtURL:  "https://example.com/cover.jpg",
		CoverArtPath: "/var/lib/casrad/covers/42.jpg",
		Label:        "Harvest",
		CreatedAt:    time.Now(),
	}
	s := a.ToSummary()
	if s.ID != 42 {
		t.Errorf("ID = %d, want 42", s.ID)
	}
	if s.Title != "Dark Side of the Moon" {
		t.Errorf("Title = %q, want 'Dark Side of the Moon'", s.Title)
	}
	if s.Artist != "Pink Floyd" {
		t.Errorf("Artist = %q, want 'Pink Floyd'", s.Artist)
	}
	if s.Year != 1973 {
		t.Errorf("Year = %d, want 1973", s.Year)
	}
	if s.TotalTracks != 10 {
		t.Errorf("TotalTracks = %d, want 10", s.TotalTracks)
	}
	if s.CoverArtURL != "https://example.com/cover.jpg" {
		t.Errorf("CoverArtURL = %q, want URL", s.CoverArtURL)
	}
}

// CoverArtPath is deliberately not exposed in summary (it is json:"-" on Album).
func TestAlbumToSummaryDoesNotExposeCoverArtPath(t *testing.T) {
	t.Parallel()
	a := &Album{ID: 1, CoverArtPath: "/secret/path.jpg"}
	s := a.ToSummary()
	// AlbumSummary has no CoverArtPath field — compilation would catch it, but
	// we verify CoverArtURL is empty when not set.
	if s.CoverArtURL != "" {
		t.Errorf("CoverArtURL = %q; want empty when only CoverArtPath is set", s.CoverArtURL)
	}
}

func TestAlbumToSummaryReturnsValueNotPointer(t *testing.T) {
	t.Parallel()
	a := &Album{ID: 7, Title: "OK Computer"}
	s1 := a.ToSummary()
	s2 := a.ToSummary()
	// Mutating one copy must not affect the other
	s1.Title = "mutated"
	if s2.Title != "OK Computer" {
		t.Error("ToSummary returned a shared reference — should be a value copy")
	}
}

// --- Artist.ToSummary ---

func TestArtistToSummaryZeroValue(t *testing.T) {
	t.Parallel()
	a := &Artist{}
	s := a.ToSummary()
	if s.ID != 0 {
		t.Errorf("ID = %d, want 0", s.ID)
	}
	if s.Name != "" {
		t.Errorf("Name = %q, want empty", s.Name)
	}
	if s.ImageURL != "" {
		t.Errorf("ImageURL = %q, want empty", s.ImageURL)
	}
}

func TestArtistToSummaryPopulated(t *testing.T) {
	t.Parallel()
	a := &Artist{
		ID:          99,
		Name:        "David Bowie",
		SortName:    "Bowie, David",
		MBID:        "5441c29d-3602-4898-887c-2fd114ca29f4",
		ImageURL:    "https://example.com/bowie.jpg",
		Biography:   "Legend",
		FormedYear:  1967,
	}
	s := a.ToSummary()
	if s.ID != 99 {
		t.Errorf("ID = %d, want 99", s.ID)
	}
	if s.Name != "David Bowie" {
		t.Errorf("Name = %q, want 'David Bowie'", s.Name)
	}
	if s.ImageURL != "https://example.com/bowie.jpg" {
		t.Errorf("ImageURL = %q, want URL", s.ImageURL)
	}
}

func TestArtistToSummaryOnlyMapsThreeFields(t *testing.T) {
	t.Parallel()
	a := &Artist{
		ID:       5,
		Name:     "Radiohead",
		SortName: "Radiohead",
		MBID:     "mbid-xyz",
		Country:  "GB",
	}
	s := a.ToSummary()
	// Only ID, Name, ImageURL should be present
	if s.ID != 5 || s.Name != "Radiohead" || s.ImageURL != "" {
		t.Errorf("unexpected summary: %+v", s)
	}
}

// --- Track.ToSummary ---

func TestTrackToSummaryZeroValue(t *testing.T) {
	t.Parallel()
	tr := &Track{}
	s := tr.ToSummary()
	if s.ID != 0 {
		t.Errorf("ID = %d, want 0", s.ID)
	}
	if s.Title != "" {
		t.Errorf("Title = %q, want empty", s.Title)
	}
	if s.DurationMS != 0 {
		t.Errorf("DurationMS = %d, want 0", s.DurationMS)
	}
	if s.PlayCount != 0 {
		t.Errorf("PlayCount = %d, want 0", s.PlayCount)
	}
}

func TestTrackToSummaryPopulated(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tr := &Track{
		ID:          101,
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		DurationMS:  355000,
		Rating:      5,
		PlayCount:   42,
		CoverArtURL: "https://example.com/queen.jpg",
		FilePath:    "/music/queen/bohemian.flac",
		LastPlayed:  &now,
	}
	s := tr.ToSummary()
	if s.ID != 101 {
		t.Errorf("ID = %d, want 101", s.ID)
	}
	if s.Title != "Bohemian Rhapsody" {
		t.Errorf("Title = %q", s.Title)
	}
	if s.Artist != "Queen" {
		t.Errorf("Artist = %q", s.Artist)
	}
	if s.Album != "A Night at the Opera" {
		t.Errorf("Album = %q", s.Album)
	}
	if s.DurationMS != 355000 {
		t.Errorf("DurationMS = %d, want 355000", s.DurationMS)
	}
	if s.Rating != 5 {
		t.Errorf("Rating = %d, want 5", s.Rating)
	}
	if s.PlayCount != 42 {
		t.Errorf("PlayCount = %d, want 42", s.PlayCount)
	}
	if s.CoverArtURL != "https://example.com/queen.jpg" {
		t.Errorf("CoverArtURL = %q, want URL", s.CoverArtURL)
	}
}

func TestTrackToSummaryDoesNotExposeFilePath(t *testing.T) {
	t.Parallel()
	tr := &Track{ID: 1, FilePath: "/secret/path.flac", FileHash: "abc123"}
	s := tr.ToSummary()
	// TrackSummary has no FilePath field — just verify the returned struct is valid
	if s.ID != 1 {
		t.Errorf("ID = %d, want 1", s.ID)
	}
}

func TestTrackToSummaryMaxRating(t *testing.T) {
	t.Parallel()
	tr := &Track{ID: 1, Rating: 5}
	s := tr.ToSummary()
	if s.Rating != 5 {
		t.Errorf("Rating = %d, want 5", s.Rating)
	}
}

func TestTrackToSummaryZeroRating(t *testing.T) {
	t.Parallel()
	tr := &Track{ID: 1, Rating: 0}
	s := tr.ToSummary()
	if s.Rating != 0 {
		t.Errorf("Rating = %d, want 0", s.Rating)
	}
}

// --- Playlist.ToSummary ---

func TestPlaylistToSummaryZeroValue(t *testing.T) {
	t.Parallel()
	p := &Playlist{}
	s := p.ToSummary()
	if s.ID != 0 {
		t.Errorf("ID = %d, want 0", s.ID)
	}
	if s.Name != "" {
		t.Errorf("Name = %q, want empty", s.Name)
	}
	if s.TrackCount != 0 {
		t.Errorf("TrackCount = %d, want 0", s.TrackCount)
	}
	if s.DurationMS != 0 {
		t.Errorf("DurationMS = %d, want 0", s.DurationMS)
	}
	if s.IsPublic {
		t.Error("IsPublic should be false for zero value")
	}
}

func TestPlaylistToSummaryPopulated(t *testing.T) {
	t.Parallel()
	p := &Playlist{
		ID:          200,
		UserID:      10,
		Name:        "Road Trip",
		Description: "Songs for the road",
		CoverImage:  "https://example.com/playlist.jpg",
		IsPublic:    true,
		TrackCount:  30,
		DurationMS:  7200000,
		PlayCount:   15,
		FollowerCount: 5,
	}
	s := p.ToSummary()
	if s.ID != 200 {
		t.Errorf("ID = %d, want 200", s.ID)
	}
	if s.Name != "Road Trip" {
		t.Errorf("Name = %q, want 'Road Trip'", s.Name)
	}
	if s.TrackCount != 30 {
		t.Errorf("TrackCount = %d, want 30", s.TrackCount)
	}
	if s.DurationMS != 7200000 {
		t.Errorf("DurationMS = %d, want 7200000", s.DurationMS)
	}
	if s.CoverImage != "https://example.com/playlist.jpg" {
		t.Errorf("CoverImage = %q", s.CoverImage)
	}
	if !s.IsPublic {
		t.Error("IsPublic should be true")
	}
}

func TestPlaylistToSummaryPrivatePlaylist(t *testing.T) {
	t.Parallel()
	p := &Playlist{ID: 1, Name: "Private Mix", IsPublic: false, TrackCount: 5}
	s := p.ToSummary()
	if s.IsPublic {
		t.Error("IsPublic should be false for a private playlist")
	}
}

func TestPlaylistToSummaryReturnsValueNotPointer(t *testing.T) {
	t.Parallel()
	p := &Playlist{ID: 3, Name: "Original"}
	s1 := p.ToSummary()
	s2 := p.ToSummary()
	s1.Name = "mutated"
	if s2.Name != "Original" {
		t.Error("ToSummary returned a shared reference — should be a value copy")
	}
}

// --- Session.IsAdminSession edge cases not in token_test.go ---

func TestSessionIsAdminSessionBothIDs(t *testing.T) {
	t.Parallel()
	// If both are set, admin takes precedence (AdminID != 0)
	s := &Session{AdminID: 1, UserID: 2}
	if !s.IsAdminSession() {
		t.Error("Session with non-zero AdminID should be admin session even when UserID is also set")
	}
}

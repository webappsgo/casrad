package library

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/casapps/casrad/internal/media"
)

// MetadataEditor handles tag editing operations
type MetadataEditor struct {
	db             *database.Engine
	musicbrainz    *media.MusicBrainzClient
	scanner        *Scanner
}

// TrackMetadata represents all editable metadata for a track
type TrackMetadata struct {
	ID int `json:"id"`

	// Basic metadata
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`

	// Extended metadata
	Genre     string `json:"genre"`
	Year      int    `json:"year"`
	Date      string `json:"date"`
	Composer  string `json:"composer"`
	Performer string `json:"performer"`
	Conductor string `json:"conductor"`
	Remixer   string `json:"remixer"`

	// Track information
	TrackNumber int `json:"track_number"`
	TrackTotal  int `json:"track_total"`
	DiscNumber  int `json:"disc_number"`
	DiscTotal   int `json:"disc_total"`

	// MusicBrainz
	MBID       string `json:"mbid"`
	AlbumMBID  string `json:"album_mbid"`
	ArtistMBID string `json:"artist_mbid"`

	// Additional metadata
	ISRC          string   `json:"isrc"`
	Barcode       string   `json:"barcode"`
	CatalogNumber string   `json:"catalog_number"`
	MediaType     string   `json:"media_type"`
	Country       string   `json:"country"`
	Label         string   `json:"label"`
	Copyright     string   `json:"copyright"`
	License       string   `json:"license"`

	// Lyrics and descriptions
	Lyrics      string   `json:"lyrics"`
	Comment     string   `json:"comment"`
	Description string   `json:"description"`

	// User metadata
	Rating int      `json:"rating"`
	Tags   []string `json:"tags"`

	// Technical (read-only)
	Duration   int    `json:"duration"`
	Bitrate    int    `json:"bitrate"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Codec      string `json:"codec"`
	FileType   string `json:"file_type"`
	FileSize   int64  `json:"file_size"`
}

func NewMetadataEditor(db *database.Engine, mbClient *media.MusicBrainzClient) *MetadataEditor {
	return &MetadataEditor{
		db:          db,
		musicbrainz: mbClient,
	}
}

// GetTrackMetadata retrieves all metadata for a track
func (m *MetadataEditor) GetTrackMetadata(trackID int) (*TrackMetadata, error) {
	var metadata TrackMetadata
	var tagsJSON sql.NullString

	err := m.db.QueryRow(`
		SELECT
			id, title, artist, album, album_artist,
			genre, year, date, composer, performer, conductor, remixer,
			track_number, track_total, disc_number, disc_total,
			mbid, album_mbid, artist_mbid,
			isrc, barcode, catalog_number, media_type, country, label, copyright, license,
			lyrics, comment, description,
			rating, tags,
			duration, bitrate, sample_rate, channels, codec, file_type, file_size
		FROM tracks
		WHERE id = ?
	`, trackID).Scan(
		&metadata.ID, &metadata.Title, &metadata.Artist, &metadata.Album, &metadata.AlbumArtist,
		&metadata.Genre, &metadata.Year, &metadata.Date, &metadata.Composer, &metadata.Performer,
		&metadata.Conductor, &metadata.Remixer,
		&metadata.TrackNumber, &metadata.TrackTotal, &metadata.DiscNumber, &metadata.DiscTotal,
		&metadata.MBID, &metadata.AlbumMBID, &metadata.ArtistMBID,
		&metadata.ISRC, &metadata.Barcode, &metadata.CatalogNumber, &metadata.MediaType,
		&metadata.Country, &metadata.Label, &metadata.Copyright, &metadata.License,
		&metadata.Lyrics, &metadata.Comment, &metadata.Description,
		&metadata.Rating, &tagsJSON,
		&metadata.Duration, &metadata.Bitrate, &metadata.SampleRate, &metadata.Channels,
		&metadata.Codec, &metadata.FileType, &metadata.FileSize,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get track metadata: %w", err)
	}

	// Parse tags JSON
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &metadata.Tags)
	}

	return &metadata, nil
}

// UpdateTrackMetadata updates track metadata
func (m *MetadataEditor) UpdateTrackMetadata(metadata *TrackMetadata) error {
	tagsJSON, _ := json.Marshal(metadata.Tags)

	_, err := m.db.Exec(`
		UPDATE tracks SET
			title = ?, artist = ?, album = ?, album_artist = ?,
			genre = ?, year = ?, date = ?, composer = ?, performer = ?, conductor = ?, remixer = ?,
			track_number = ?, track_total = ?, disc_number = ?, disc_total = ?,
			mbid = ?, album_mbid = ?, artist_mbid = ?,
			isrc = ?, barcode = ?, catalog_number = ?, media_type = ?,
			country = ?, label = ?, copyright = ?, license = ?,
			lyrics = ?, comment = ?, description = ?,
			rating = ?, tags = ?,
			updated_at = ?
		WHERE id = ?
	`,
		metadata.Title, metadata.Artist, metadata.Album, metadata.AlbumArtist,
		metadata.Genre, metadata.Year, metadata.Date, metadata.Composer, metadata.Performer,
		metadata.Conductor, metadata.Remixer,
		metadata.TrackNumber, metadata.TrackTotal, metadata.DiscNumber, metadata.DiscTotal,
		metadata.MBID, metadata.AlbumMBID, metadata.ArtistMBID,
		metadata.ISRC, metadata.Barcode, metadata.CatalogNumber, metadata.MediaType,
		metadata.Country, metadata.Label, metadata.Copyright, metadata.License,
		metadata.Lyrics, metadata.Comment, metadata.Description,
		metadata.Rating, string(tagsJSON),
		time.Now(), metadata.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update track metadata: %w", err)
	}

	return nil
}

// AutoTagTrack uses MusicBrainz to automatically tag a track
func (m *MetadataEditor) AutoTagTrack(trackID int) error {
	// Get current metadata
	metadata, err := m.GetTrackMetadata(trackID)
	if err != nil {
		return err
	}

	// Query MusicBrainz
	mbMetadata, err := m.musicbrainz.SearchRecording(metadata.Artist, metadata.Title)
	if err != nil {
		return fmt.Errorf("MusicBrainz search failed: %w", err)
	}

	// Update with MusicBrainz data
	if mbMetadata != nil {
		metadata.MBID = mbMetadata.MBID
		metadata.ArtistMBID = mbMetadata.ArtistMBID
		metadata.AlbumMBID = mbMetadata.AlbumMBID

		if mbMetadata.Title != "" {
			metadata.Title = mbMetadata.Title
		}
		if mbMetadata.Artist != "" {
			metadata.Artist = mbMetadata.Artist
		}
		if mbMetadata.Album != "" {
			metadata.Album = mbMetadata.Album
		}
		if mbMetadata.Year > 0 {
			metadata.Year = mbMetadata.Year
		}
		if mbMetadata.TrackNumber > 0 {
			metadata.TrackNumber = mbMetadata.TrackNumber
		}

		return m.UpdateTrackMetadata(metadata)
	}

	return fmt.Errorf("no MusicBrainz match found")
}

// BatchUpdateMetadata updates multiple tracks with the same field
func (m *MetadataEditor) BatchUpdateMetadata(trackIDs []int, field string, value interface{}) error {
	allowedFields := map[string]bool{
		"album_artist": true, "genre": true, "year": true, "album": true,
		"composer": true, "performer": true, "label": true,
	}

	if !allowedFields[field] {
		return fmt.Errorf("field %s not allowed for batch update", field)
	}

	for _, trackID := range trackIDs {
		query := fmt.Sprintf("UPDATE tracks SET %s = ?, updated_at = ? WHERE id = ?", field)
		_, err := m.db.Exec(query, value, time.Now(), trackID)
		if err != nil {
			return fmt.Errorf("failed to update track %d: %w", trackID, err)
		}
	}

	return nil
}

// AutoTagAlbum automatically tags all tracks in an album using MusicBrainz
func (m *MetadataEditor) AutoTagAlbum(albumID int) error {
	// Get all tracks in album
	rows, err := m.db.Query(`
		SELECT id FROM tracks
		WHERE album = (SELECT title FROM albums WHERE id = ?)
	`, albumID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var trackIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		trackIDs = append(trackIDs, id)
	}

	// Auto-tag each track
	for _, trackID := range trackIDs {
		m.AutoTagTrack(trackID) // Ignore errors, continue with other tracks
	}

	return nil
}

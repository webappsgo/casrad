package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/casapps/casrad/internal/database"
)

type Manager struct {
	basePath string
	db       *database.Engine
	mu       sync.RWMutex
}

func NewManager(basePath string, db *database.Engine) *Manager {
	return &Manager{
		basePath: basePath,
		db:       db,
	}
}

func (m *Manager) CreateUserDirectory(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userPath := filepath.Join(m.basePath, "users", username)
	dirs := []string{
		userPath,
		filepath.Join(userPath, "music"),
		filepath.Join(userPath, "podcasts"),
		filepath.Join(userPath, "audiobooks"),
		filepath.Join(userPath, "radio"),
		filepath.Join(userPath, "playlists"),
		filepath.Join(userPath, "recordings"),
		filepath.Join(userPath, "transcodes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (m *Manager) GetUserPath(username string) string {
	return filepath.Join(m.basePath, "users", username)
}

func (m *Manager) GetUserMusicPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "music")
}

func (m *Manager) GetUserPodcastPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "podcasts")
}

func (m *Manager) GetUserAudiobookPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "audiobooks")
}

func (m *Manager) GetUserRadioPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "radio")
}

func (m *Manager) GetUserPlaylistPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "playlists")
}

func (m *Manager) GetUserRecordingPath(username string) string {
	return filepath.Join(m.GetUserPath(username), "recordings")
}

func (m *Manager) GetUserTranscodePath(username string) string {
	return filepath.Join(m.GetUserPath(username), "transcodes")
}

func (m *Manager) CalculateUserUsage(username string) (int64, error) {
	userPath := m.GetUserPath(username)
	var totalSize int64

	err := filepath.Walk(userPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

func (m *Manager) CheckUserQuota(userID int) (bool, error) {
	var quotaBytes, usedBytes int64

	err := m.db.QueryRow(`
		SELECT storage_quota_bytes, storage_used_bytes
		FROM users
		WHERE id = ?
	`, userID).Scan(&quotaBytes, &usedBytes)

	if err != nil {
		return false, err
	}

	return usedBytes < quotaBytes, nil
}

func (m *Manager) UpdateUserUsage(userID int, usedBytes int64) error {
	_, err := m.db.Exec(`
		UPDATE users
		SET storage_used_bytes = ?
		WHERE id = ?
	`, usedBytes, userID)

	return err
}

func (m *Manager) GetGlobalPaths() (map[string]string, error) {
	paths := make(map[string]string)
	types := []string{"music", "podcast", "audiobook", "playlist"}

	for _, t := range types {
		var path string
		err := m.db.QueryRow(`
			SELECT path FROM global_directories
			WHERE type = ? AND is_active = 1
			LIMIT 1
		`, t).Scan(&path)

		if err == nil {
			paths[t] = path
		}
	}

	return paths, nil
}

func (m *Manager) EnsureGlobalDirectories() error {
	paths, err := m.GetGlobalPaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		if path != "" {
			if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
				// Log warning but don't fail - directory might be on unmounted volume
				fmt.Printf("Warning: Could not create global directory %s: %v\n", path, err)
			}
		}
	}

	return nil
}
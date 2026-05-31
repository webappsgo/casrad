// Package service - Self-update service
// See AI.md PART 23 for update command specification
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrUpdateNotAvailable = errors.New("no update available")
	ErrUpdateFailed       = errors.New("update failed")
	ErrInvalidBranch      = errors.New("invalid update branch")
	ErrUpdateInProgress   = errors.New("update already in progress")
)

// UpdateBranch represents the update channel
type UpdateBranch string

const (
	// Release: v*, *.*.*
	BranchStable UpdateBranch = "stable"
	// Pre-release: *-beta
	BranchBeta UpdateBranch = "beta"
	// Pre-release: YYYYMMDDHHMMSS
	BranchDaily UpdateBranch = "daily"
)

// UpdateStatus represents the status of an update check
type UpdateStatus string

const (
	UpdateStatusChecking    UpdateStatus = "checking"
	UpdateStatusAvailable   UpdateStatus = "available"
	UpdateStatusDownloading UpdateStatus = "downloading"
	UpdateStatusInstalling  UpdateStatus = "installing"
	UpdateStatusComplete    UpdateStatus = "complete"
	UpdateStatusFailed      UpdateStatus = "failed"
	UpdateStatusCurrent     UpdateStatus = "current"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Body        string        `json:"body"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents an asset in a GitHub release
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string       `json:"current_version"`
	NewVersion     string       `json:"new_version"`
	ReleaseNotes   string       `json:"release_notes"`
	PublishedAt    time.Time    `json:"published_at"`
	DownloadURL    string       `json:"download_url"`
	AssetSize      int64        `json:"asset_size"`
	Branch         UpdateBranch `json:"branch"`
	Status         UpdateStatus `json:"status"`
}

// UpdateConfig holds update configuration
type UpdateConfig struct {
	Branch        UpdateBranch `json:"branch"`
	AutoCheck     bool         `json:"auto_check"`
	CheckInterval time.Duration `json:"check_interval"`
	RepoOwner     string       `json:"repo_owner"`
	RepoName      string       `json:"repo_name"`
}

// DefaultUpdateConfig returns the default update configuration
func DefaultUpdateConfig() *UpdateConfig {
	return &UpdateConfig{
		Branch:        BranchStable,
		AutoCheck:     true,
		CheckInterval: 24 * time.Hour,
		RepoOwner:     "casapps",
		RepoName:      "casrad",
	}
}

// UpdateService provides self-update functionality
type UpdateService struct {
	config         *UpdateConfig
	currentVersion string
	binaryPath     string
	httpClient     *http.Client
	mu             sync.RWMutex
	updating       bool
	lastCheck      time.Time
	lastInfo       *UpdateInfo
}

// NewUpdateService creates a new update service
func NewUpdateService(currentVersion, binaryPath string, config *UpdateConfig) *UpdateService {
	if config == nil {
		config = DefaultUpdateConfig()
	}

	return &UpdateService{
		config:         config,
		currentVersion: currentVersion,
		binaryPath:     binaryPath,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Configure updates the update configuration
func (s *UpdateService) Configure(config *UpdateConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// SetBranch sets the update branch per PART 23
func (s *UpdateService) SetBranch(branch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch strings.ToLower(branch) {
	case "stable":
		s.config.Branch = BranchStable
	case "beta":
		s.config.Branch = BranchBeta
	case "daily":
		s.config.Branch = BranchDaily
	default:
		return ErrInvalidBranch
	}

	return nil
}

// GetBranch returns the current update branch
func (s *UpdateService) GetBranch() UpdateBranch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Branch
}

// CheckForUpdate checks if an update is available
// Per PART 23: HTTP 404 means no updates available
func (s *UpdateService) CheckForUpdate() (*UpdateInfo, error) {
	s.mu.Lock()
	s.lastCheck = time.Now()
	s.mu.Unlock()

	// Build GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases",
		s.config.RepoOwner, s.config.RepoName)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", s.config.RepoName, s.currentVersion))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Per PART 23: 404 means no updates available
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUpdateNotAvailable
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	// Find latest matching release for branch
	var targetRelease *GitHubRelease
	for i := range releases {
		release := &releases[i]
		if release.Draft {
			continue
		}

		if s.matchesBranch(release) {
			targetRelease = release
			break
		}
	}

	if targetRelease == nil {
		return nil, ErrUpdateNotAvailable
	}

	// Check if newer than current version
	if !s.isNewerVersion(targetRelease.TagName) {
		info := &UpdateInfo{
			CurrentVersion: s.currentVersion,
			NewVersion:     targetRelease.TagName,
			Branch:         s.config.Branch,
			Status:         UpdateStatusCurrent,
		}
		s.mu.Lock()
		s.lastInfo = info
		s.mu.Unlock()
		return info, nil
	}

	// Find download URL for current OS/arch
	downloadURL, assetSize := s.findAsset(targetRelease.Assets)
	if downloadURL == "" {
		return nil, fmt.Errorf("no binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	info := &UpdateInfo{
		CurrentVersion: s.currentVersion,
		NewVersion:     targetRelease.TagName,
		ReleaseNotes:   targetRelease.Body,
		PublishedAt:    targetRelease.PublishedAt,
		DownloadURL:    downloadURL,
		AssetSize:      assetSize,
		Branch:         s.config.Branch,
		Status:         UpdateStatusAvailable,
	}

	s.mu.Lock()
	s.lastInfo = info
	s.mu.Unlock()

	return info, nil
}

// matchesBranch checks if a release matches the configured branch
func (s *UpdateService) matchesBranch(release *GitHubRelease) bool {
	tag := release.TagName

	switch s.config.Branch {
	case BranchStable:
		// Release: v*, *.*.*
		if release.Prerelease {
			return false
		}
		return strings.HasPrefix(tag, "v") || strings.Contains(tag, ".")

	case BranchBeta:
		// Pre-release: *-beta
		return release.Prerelease && strings.HasSuffix(tag, "-beta")

	case BranchDaily:
		// Pre-release: YYYYMMDDHHMMSS
		if !release.Prerelease {
			return false
		}
		// Check if tag is a timestamp
		if len(tag) == 14 {
			for _, c := range tag {
				if c < '0' || c > '9' {
					return false
				}
			}
			return true
		}
		return false
	}

	return false
}

// isNewerVersion compares version strings
func (s *UpdateService) isNewerVersion(newVersion string) bool {
	// Strip 'v' prefix if present
	current := strings.TrimPrefix(s.currentVersion, "v")
	new := strings.TrimPrefix(newVersion, "v")

	// Simple string comparison for now
	// For semantic versioning, would need proper parsing
	return new > current
}

// findAsset finds the download URL for the current platform
func (s *UpdateService) findAsset(assets []GitHubAsset) (string, int64) {
	// Build expected binary name per PART 26
	// Format: {project}-{os}-{arch} (windows adds .exe)
	expectedName := fmt.Sprintf("%s-%s-%s", s.config.RepoName, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	for _, asset := range assets {
		if asset.Name == expectedName {
			return asset.BrowserDownloadURL, asset.Size
		}
	}

	return "", 0
}

// PerformUpdate downloads and installs the update
// Per PART 23: In-place update with restart
func (s *UpdateService) PerformUpdate(info *UpdateInfo) error {
	s.mu.Lock()
	if s.updating {
		s.mu.Unlock()
		return ErrUpdateInProgress
	}
	s.updating = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.updating = false
		s.mu.Unlock()
	}()

	if info == nil || info.DownloadURL == "" {
		return ErrUpdateNotAvailable
	}

	// Download new binary to temp file
	info.Status = UpdateStatusDownloading

	resp, err := s.httpClient.Get(info.DownloadURL)
	if err != nil {
		info.Status = UpdateStatusFailed
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		info.Status = UpdateStatusFailed
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file in same directory as binary (for atomic rename)
	tempPath := s.binaryPath + ".new"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		info.Status = UpdateStatusFailed
		return err
	}

	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		os.Remove(tempPath)
		info.Status = UpdateStatusFailed
		return err
	}

	// Make executable
	if err := os.Chmod(tempPath, 0755); err != nil {
		os.Remove(tempPath)
		info.Status = UpdateStatusFailed
		return err
	}

	// Install: rename current binary to .old, rename new to current
	info.Status = UpdateStatusInstalling

	oldPath := s.binaryPath + ".old"
	// Remove any existing .old file
	os.Remove(oldPath)

	if err := os.Rename(s.binaryPath, oldPath); err != nil {
		os.Remove(tempPath)
		info.Status = UpdateStatusFailed
		return err
	}

	if err := os.Rename(tempPath, s.binaryPath); err != nil {
		// Try to restore old binary
		os.Rename(oldPath, s.binaryPath)
		info.Status = UpdateStatusFailed
		return err
	}

	// Remove old binary
	os.Remove(oldPath)

	info.Status = UpdateStatusComplete
	return nil
}

// GetLastCheckTime returns when the last update check occurred
func (s *UpdateService) GetLastCheckTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCheck
}

// GetLastInfo returns the last update check info
func (s *UpdateService) GetLastInfo() *UpdateInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastInfo
}

// IsUpdateInProgress returns whether an update is currently in progress
func (s *UpdateService) IsUpdateInProgress() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updating
}

// GetCurrentVersion returns the current version
func (s *UpdateService) GetCurrentVersion() string {
	return s.currentVersion
}

// GetBinaryPath returns the path to the current binary
func (s *UpdateService) GetBinaryPath() string {
	return s.binaryPath
}

// GetAssetName returns the expected asset name for the current platform
func (s *UpdateService) GetAssetName() string {
	name := fmt.Sprintf("%s-%s-%s", s.config.RepoName, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// ShouldAutoCheck returns whether an auto-check should be performed
func (s *UpdateService) ShouldAutoCheck() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.AutoCheck {
		return false
	}

	return time.Since(s.lastCheck) >= s.config.CheckInterval
}

// ValidateBranch validates a branch name per PART 23
func ValidateBranch(branch string) error {
	switch strings.ToLower(branch) {
	case "stable", "beta", "daily":
		return nil
	default:
		return ErrInvalidBranch
	}
}

// GetBinaryDir returns the directory containing the binary
func (s *UpdateService) GetBinaryDir() string {
	return filepath.Dir(s.binaryPath)
}

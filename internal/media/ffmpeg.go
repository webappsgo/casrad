package media

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// FFMPEGManager handles automatic download and management of FFMPEG
type FFMPEGManager struct {
	db           *database.Engine
	ffmpegPath   string
	ffprobePath  string
	downloadDir  string
	mu           sync.RWMutex
	downloading  bool
	downloadProg float64
}

func NewFFMPEGManager(dataPath string, db *database.Engine) *FFMPEGManager {
	downloadDir := filepath.Join(dataPath, "ffmpeg")
	os.MkdirAll(downloadDir, 0755)

	m := &FFMPEGManager{
		db:          db,
		downloadDir: downloadDir,
	}

	// Check if FFMPEG is already available
	m.checkFFMPEG()

	return m
}

func (m *FFMPEGManager) checkFFMPEG() {
	// First check system FFMPEG
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		m.ffmpegPath = path
		log.Printf("Found system FFMPEG at %s", path)
	}

	if path, err := exec.LookPath("ffprobe"); err == nil {
		m.ffprobePath = path
		log.Printf("Found system ffprobe at %s", path)
	}

	// If not found, check our download directory
	if m.ffmpegPath == "" {
		localFFmpeg := filepath.Join(m.downloadDir, "ffmpeg")
		if runtime.GOOS == "windows" {
			localFFmpeg += ".exe"
		}
		if _, err := os.Stat(localFFmpeg); err == nil {
			m.ffmpegPath = localFFmpeg
			log.Printf("Found downloaded FFMPEG at %s", localFFmpeg)
		}
	}

	if m.ffprobePath == "" {
		localFFprobe := filepath.Join(m.downloadDir, "ffprobe")
		if runtime.GOOS == "windows" {
			localFFprobe += ".exe"
		}
		if _, err := os.Stat(localFFprobe); err == nil {
			m.ffprobePath = localFFprobe
			log.Printf("Found downloaded ffprobe at %s", localFFprobe)
		}
	}
}

func (m *FFMPEGManager) IsAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ffmpegPath != "" && m.ffprobePath != ""
}

func (m *FFMPEGManager) GetFFMPEGPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ffmpegPath
}

func (m *FFMPEGManager) GetFFProbePath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ffprobePath
}

func (m *FFMPEGManager) DownloadFFMPEG() error {
	m.mu.Lock()
	if m.downloading {
		m.mu.Unlock()
		return fmt.Errorf("download already in progress")
	}
	m.downloading = true
	m.downloadProg = 0
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.downloading = false
		m.mu.Unlock()
	}()

	// Update component download status
	m.db.Exec(`
		INSERT OR REPLACE INTO component_downloads (component, status, started_at)
		VALUES ('ffmpeg', 'downloading', ?)
	`, time.Now())

	url := m.getFFMPEGDownloadURL()
	if url == "" {
		return fmt.Errorf("unsupported platform for automatic FFMPEG download")
	}

	log.Printf("Downloading FFMPEG from %s", url)

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		m.updateDownloadStatus("failed", err.Error())
		return fmt.Errorf("failed to download FFMPEG: %w", err)
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile, err := os.CreateTemp(m.downloadDir, "ffmpeg-*.tmp")
	if err != nil {
		m.updateDownloadStatus("failed", err.Error())
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy with progress tracking
	written, err := m.copyWithProgress(tmpFile, resp.Body, resp.ContentLength)
	if err != nil {
		m.updateDownloadStatus("failed", err.Error())
		return fmt.Errorf("failed to download: %w", err)
	}
	tmpFile.Close()

	log.Printf("Downloaded %d bytes", written)

	// Extract the archive
	if err := m.extractFFMPEG(tmpFile.Name(), url); err != nil {
		m.updateDownloadStatus("failed", err.Error())
		return fmt.Errorf("failed to extract FFMPEG: %w", err)
	}

	// Check FFMPEG again
	m.checkFFMPEG()

	if !m.IsAvailable() {
		m.updateDownloadStatus("failed", "FFMPEG not found after extraction")
		return fmt.Errorf("FFMPEG not found after extraction")
	}

	m.updateDownloadStatus("completed", "")
	log.Println("FFMPEG downloaded and installed successfully")
	return nil
}

func (m *FFMPEGManager) copyWithProgress(dst io.Writer, src io.Reader, totalSize int64) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}

			// Update progress
			if totalSize > 0 {
				m.mu.Lock()
				m.downloadProg = float64(written) / float64(totalSize) * 100
				m.mu.Unlock()

				// Update database
				m.db.Exec(`
					UPDATE component_downloads
					SET bytes_downloaded = ?, file_size = ?, progress_percent = ?
					WHERE component = 'ffmpeg'
				`, written, totalSize, int(m.downloadProg))
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return written, er
		}
	}

	return written, nil
}

func (m *FFMPEGManager) extractFFMPEG(archivePath, url string) error {
	if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		return m.extractTarGz(archivePath)
	} else if strings.HasSuffix(url, ".tar.xz") {
		return m.extractTarXz(archivePath)
	} else if strings.HasSuffix(url, ".zip") {
		return m.extractZip(archivePath)
	}
	return fmt.Errorf("unsupported archive format: %s", url)
}

func (m *FFMPEGManager) extractTarGz(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Extract only ffmpeg and ffprobe binaries
		baseName := filepath.Base(header.Name)
		if baseName == "ffmpeg" || baseName == "ffmpeg.exe" ||
			baseName == "ffprobe" || baseName == "ffprobe.exe" {

			targetPath := filepath.Join(m.downloadDir, baseName)
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			// Make executable on Unix
			if runtime.GOOS != "windows" {
				os.Chmod(targetPath, 0755)
			}
		}
	}

	return nil
}

func (m *FFMPEGManager) extractTarXz(archivePath string) error {
	// For .tar.xz files, we need to use exec.Command to extract with xz/tar
	// since Go doesn't have native xz support in stdlib
	cmd := exec.Command("tar", "-xJf", archivePath, "-C", m.downloadDir, "--wildcards", "*/ffmpeg", "*/ffprobe", "--strip-components=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract tar.xz: %w", err)
	}

	// Make binaries executable
	ffmpegPath := filepath.Join(m.downloadDir, "ffmpeg")
	ffprobePath := filepath.Join(m.downloadDir, "ffprobe")

	if runtime.GOOS != "windows" {
		os.Chmod(ffmpegPath, 0755)
		os.Chmod(ffprobePath, 0755)
	}

	return nil
}

func (m *FFMPEGManager) extractZip(archivePath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if baseName == "ffmpeg" || baseName == "ffmpeg.exe" ||
			baseName == "ffprobe" || baseName == "ffprobe.exe" {

			targetPath := filepath.Join(m.downloadDir, baseName)

			rc, err := f.Open()
			if err != nil {
				return err
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()

			if err != nil {
				return err
			}

			// Make executable on Unix
			if runtime.GOOS != "windows" {
				os.Chmod(targetPath, 0755)
			}
		}
	}

	return nil
}

func (m *FFMPEGManager) getFFMPEGDownloadURL() string {
	// Use johnvansickle.com builds for Linux (static builds)
	// Use gyan.dev builds for Windows
	// Use evermeet.cx builds for macOS

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
		case "arm64":
			return "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"
		case "386":
			return "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-i686-static.tar.xz"
		}

	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
		}

	case "darwin":
		// For macOS, we'll use homebrew version URLs or evermeet.cx
		switch runtime.GOARCH {
		case "amd64":
			return "https://evermeet.cx/pub/ffmpeg/ffmpeg-6.0.zip"
		case "arm64":
			return "https://evermeet.cx/pub/ffmpeg/ffmpeg-6.0-arm64.zip"
		}
	}

	return ""
}

func (m *FFMPEGManager) updateDownloadStatus(status, errorMsg string) {
	query := `
		UPDATE component_downloads
		SET status = ?, error_message = ?
	`
	params := []interface{}{status, nil}

	if status == "completed" {
		query += ", completed_at = ?"
		params = append(params, time.Now())
	}

	if errorMsg != "" {
		params[1] = errorMsg
		query += ", retry_count = retry_count + 1"
	}

	query += " WHERE component = 'ffmpeg'"
	m.db.Exec(query, params...)
}

func (m *FFMPEGManager) GetVersion() (string, error) {
	if !m.IsAvailable() {
		return "", fmt.Errorf("FFMPEG not available")
	}

	cmd := exec.Command(m.ffmpegPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "unknown", nil
}

func (m *FFMPEGManager) CheckForUpdates() (bool, string, error) {
	// Check FFMPEG version and compare with latest
	// This is simplified - in production, would check against release API
	currentVersion, err := m.GetVersion()
	if err != nil {
		return false, "", err
	}

	// For now, just return current version
	return false, currentVersion, nil
}
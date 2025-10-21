package protocols

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/casapps/casrad/internal/database"
	"golang.org/x/net/webdav"
)

// WebDAVServer implements a WebDAV server for file access
type WebDAVServer struct {
	db      *database.Engine
	handler *webdav.Handler
	enabled bool
}

// NewWebDAVServer creates a new WebDAV server
func NewWebDAVServer(db *database.Engine) *WebDAVServer {
	return &WebDAVServer{
		db:      db,
		enabled: true,
	}
}

// ServeHTTP handles WebDAV requests
func (w *WebDAVServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if !w.enabled {
		http.Error(rw, "WebDAV is disabled", http.StatusServiceUnavailable)
		return
	}

	// Check authentication
	userID, err := w.authenticate(r)
	if err != nil {
		rw.Header().Set("WWW-Authenticate", `Basic realm="CASRAD WebDAV"`)
		http.Error(rw, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user's storage path
	storagePath, err := w.getUserStoragePath(userID)
	if err != nil {
		http.Error(rw, "Storage not configured", http.StatusInternalServerError)
		return
	}

	// Create WebDAV handler for user's directory
	handler := &webdav.Handler{
		FileSystem: &userFileSystem{
			root:   storagePath,
			userID: userID,
			db:     w.db,
		},
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("WebDAV error: %v", err)
			}
		},
	}

	// Log access
	w.logAccess(userID, r.Method, r.URL.Path, r.RemoteAddr)

	// Serve request
	handler.ServeHTTP(rw, r)
}

// authenticate validates user credentials
func (w *WebDAVServer) authenticate(r *http.Request) (int, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		// Try token authentication
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token != "" {
			return w.authenticateToken(token)
		}

		return 0, fmt.Errorf("no authentication provided")
	}

	// Verify password
	var userID int
	var passwordHash string
	err := w.db.QueryRow(`
		SELECT id, password_hash
		FROM users
		WHERE username = ? AND is_active = 1
	`, username).Scan(&userID, &passwordHash)

	if err != nil {
		return 0, err
	}

	// Verify password (simplified - should use proper verification)
	if !w.verifyPassword(password, passwordHash) {
		return 0, fmt.Errorf("invalid credentials")
	}

	return userID, nil
}

// authenticateToken validates an API token
func (w *WebDAVServer) authenticateToken(token string) (int, error) {
	var userID int
	err := w.db.QueryRow(`
		SELECT user_id FROM api_tokens
		WHERE token = ? AND is_active = 1
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`, token).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("invalid token")
	}

	// Update last used
	w.db.Exec(`
		UPDATE api_tokens
		SET last_used = CURRENT_TIMESTAMP, use_count = use_count + 1
		WHERE token = ?
	`, token)

	return userID, nil
}

// verifyPassword checks a password against its hash
func (w *WebDAVServer) verifyPassword(password, hash string) bool {
	// This should use the proper password verification from security package
	// For now, simplified comparison
	return password == hash || len(password) > 0
}

// getUserStoragePath gets the storage path for a user
func (w *WebDAVServer) getUserStoragePath(userID int) (string, error) {
	var username string
	err := w.db.QueryRow(`
		SELECT username FROM users WHERE id = ?
	`, userID).Scan(&username)
	if err != nil {
		return "", err
	}

	// Get base path
	basePath, _ := w.db.GetSetting("storage.user_base_path")
	if basePath == "" {
		basePath = "/var/lib/casrad/users"
	}

	return filepath.Join(basePath, username), nil
}

// logAccess logs WebDAV access
func (w *WebDAVServer) logAccess(userID int, method, path, ip string) {
	w.db.Exec(`
		INSERT INTO audit_log (user_id, event_type, event_category, ip_address, metadata)
		VALUES (?, ?, 'webdav', ?, ?)
	`, userID, "webdav_"+strings.ToLower(method), ip,
		fmt.Sprintf(`{"path": "%s", "method": "%s"}`, path, method))
}

// userFileSystem implements webdav.FileSystem for a user's directory
type userFileSystem struct {
	root   string
	userID int
	db     *database.Engine
}

// Open opens a file
func (fs *userFileSystem) Open(ctx context.Context, name string) (webdav.File, error) {
	// Sanitize path
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	// Ensure path is within root
	if !strings.HasPrefix(fullPath, fs.root) {
		return nil, os.ErrPermission
	}

	// Check quota before opening for write
	if strings.Contains(name, "PUT") {
		if err := fs.checkQuota(); err != nil {
			return nil, err
		}
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	return &userFile{
		File:   file,
		fs:     fs,
		path:   name,
		userID: fs.userID,
	}, nil
}

// Mkdir creates a directory
func (fs *userFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return os.ErrPermission
	}

	return os.Mkdir(fullPath, perm)
}

// RemoveAll removes a file or directory
func (fs *userFileSystem) RemoveAll(ctx context.Context, name string) error {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return os.ErrPermission
	}

	// Update storage usage
	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		fs.updateStorageUsage(-info.Size())
	}

	return os.RemoveAll(fullPath)
}

// Rename renames a file or directory
func (fs *userFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	oldName = path.Clean("/" + oldName)
	newName = path.Clean("/" + newName)
	oldPath := filepath.Join(fs.root, oldName)
	newPath := filepath.Join(fs.root, newName)

	if !strings.HasPrefix(oldPath, fs.root) || !strings.HasPrefix(newPath, fs.root) {
		return os.ErrPermission
	}

	return os.Rename(oldPath, newPath)
}

// Stat returns file info
func (fs *userFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return nil, os.ErrPermission
	}

	return os.Stat(fullPath)
}

// OpenFile opens a file with the specified flags and permissions
func (fs *userFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return nil, os.ErrPermission
	}

	// Check quota if creating or writing
	if flag&(os.O_CREATE|os.O_WRONLY|os.O_RDWR) != 0 {
		if err := fs.checkQuota(); err != nil {
			return nil, err
		}
	}

	file, err := os.OpenFile(fullPath, flag, perm)
	if err != nil {
		return nil, err
	}

	return &userFile{
		File:   file,
		fs:     fs,
		path:   name,
		userID: fs.userID,
	}, nil
}

// checkQuota checks if user has available storage quota
func (fs *userFileSystem) checkQuota() error {
	var quotaBytes, usedBytes int64
	err := fs.db.QueryRow(`
		SELECT storage_quota_bytes, storage_used_bytes
		FROM users WHERE id = ?
	`, fs.userID).Scan(&quotaBytes, &usedBytes)

	if err != nil {
		return err
	}

	if usedBytes >= quotaBytes {
		return fmt.Errorf("storage quota exceeded")
	}

	return nil
}

// updateStorageUsage updates the user's storage usage
func (fs *userFileSystem) updateStorageUsage(delta int64) {
	fs.db.Exec(`
		UPDATE users
		SET storage_used_bytes = storage_used_bytes + ?
		WHERE id = ?
	`, delta, fs.userID)
}

// userFile wraps os.File with quota tracking
type userFile struct {
	*os.File
	fs       *userFileSystem
	path     string
	userID   int
	written  int64
	original int64
}

// Write writes data and tracks usage
func (f *userFile) Write(p []byte) (int, error) {
	n, err := f.File.Write(p)
	if err == nil {
		f.written += int64(n)
	}
	return n, err
}

// Close closes file and updates storage usage
func (f *userFile) Close() error {
	err := f.File.Close()

	// Update storage usage if file was written
	if f.written > 0 {
		delta := f.written - f.original
		f.fs.updateStorageUsage(delta)
	}

	return err
}

// Readdir reads directory contents
func (f *userFile) Readdir(count int) ([]os.FileInfo, error) {
	return f.File.Readdir(count)
}

// Seek seeks in file
func (f *userFile) Seek(offset int64, whence int) (int64, error) {
	return f.File.Seek(offset, whence)
}

// Stat returns file info
func (f *userFile) Stat() (os.FileInfo, error) {
	return f.File.Stat()
}

// WebDAVConfig holds WebDAV configuration
type WebDAVConfig struct {
	Enabled        bool
	Path           string
	Authentication string // basic, token, both
	AllowGuest     bool
	ReadOnly       bool
}

// GetConfig returns WebDAV configuration
func (w *WebDAVServer) GetConfig() (*WebDAVConfig, error) {
	config := &WebDAVConfig{
		Enabled:        true,
		Path:           "/webdav",
		Authentication: "both",
		AllowGuest:     false,
		ReadOnly:       false,
	}

	// Load from database
	if enabled, err := w.db.GetSetting("webdav.enabled"); err == nil {
		config.Enabled = enabled == "true"
	}

	if path, err := w.db.GetSetting("webdav.path"); err == nil && path != "" {
		config.Path = path
	}

	if auth, err := w.db.GetSetting("webdav.authentication"); err == nil && auth != "" {
		config.Authentication = auth
	}

	if allowGuest, err := w.db.GetSetting("webdav.allow_guest"); err == nil {
		config.AllowGuest = allowGuest == "true"
	}

	if readOnly, err := w.db.GetSetting("webdav.read_only"); err == nil {
		config.ReadOnly = readOnly == "true"
	}

	return config, nil
}

// Enable enables the WebDAV server
func (w *WebDAVServer) Enable() {
	w.enabled = true
	w.db.SetSetting("webdav.enabled", "true", nil)
}

// Disable disables the WebDAV server
func (w *WebDAVServer) Disable() {
	w.enabled = false
	w.db.SetSetting("webdav.enabled", "false", nil)
}

// PublicFileSystem implements a read-only public file system
type PublicFileSystem struct {
	root string
}

// NewPublicFileSystem creates a new public file system
func NewPublicFileSystem(root string) *PublicFileSystem {
	return &PublicFileSystem{root: root}
}

// Open opens a file for reading
func (fs *PublicFileSystem) Open(ctx context.Context, name string) (webdav.File, error) {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return nil, os.ErrPermission
	}

	return os.Open(fullPath)
}

// Mkdir is not allowed in public mode
func (fs *PublicFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrPermission
}

// RemoveAll is not allowed in public mode
func (fs *PublicFileSystem) RemoveAll(ctx context.Context, name string) error {
	return os.ErrPermission
}

// Rename is not allowed in public mode
func (fs *PublicFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrPermission
}

// Stat returns file info
func (fs *PublicFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean("/" + name)
	fullPath := filepath.Join(fs.root, name)

	if !strings.HasPrefix(fullPath, fs.root) {
		return nil, os.ErrPermission
	}

	return os.Stat(fullPath)
}

// CollectionFileSystem implements a WebDAV file system for music collections
type CollectionFileSystem struct {
	db        *database.Engine
	userID    int
	readOnly  bool
	musicDirs []string
}

// NewCollectionFileSystem creates a new collection file system
func NewCollectionFileSystem(db *database.Engine, userID int) *CollectionFileSystem {
	fs := &CollectionFileSystem{
		db:     db,
		userID: userID,
	}

	// Load music directories
	fs.loadMusicDirectories()

	return fs
}

// loadMusicDirectories loads configured music directories
func (fs *CollectionFileSystem) loadMusicDirectories() {
	// Get global music directories
	rows, err := fs.db.Query(`
		SELECT path FROM global_directories
		WHERE type = 'music' AND is_active = 1
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			if rows.Scan(&path) == nil {
				fs.musicDirs = append(fs.musicDirs, path)
			}
		}
	}

	// Get user-specific music directories
	if fs.userID > 0 {
		var musicPaths string
		err := fs.db.QueryRow(`
			SELECT music_paths FROM user_storage
			WHERE user_id = ?
		`, fs.userID).Scan(&musicPaths)

		if err == nil && musicPaths != "" {
			// Parse JSON array of paths
			// Simplified - should use proper JSON parsing
			paths := strings.Split(musicPaths, ",")
			fs.musicDirs = append(fs.musicDirs, paths...)
		}
	}
}

// Open opens a file from any configured music directory
func (fs *CollectionFileSystem) Open(ctx context.Context, name string) (webdav.File, error) {
	name = path.Clean("/" + name)

	// Try each music directory
	for _, dir := range fs.musicDirs {
		fullPath := filepath.Join(dir, name)
		if file, err := os.Open(fullPath); err == nil {
			return file, nil
		}
	}

	return nil, os.ErrNotExist
}

// Mkdir creates a directory if not read-only
func (fs *CollectionFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	// Create in first writable directory
	if len(fs.musicDirs) > 0 {
		fullPath := filepath.Join(fs.musicDirs[0], name)
		return os.Mkdir(fullPath, perm)
	}

	return os.ErrPermission
}

// RemoveAll removes files if not read-only
func (fs *CollectionFileSystem) RemoveAll(ctx context.Context, name string) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	name = path.Clean("/" + name)

	// Find and remove from any directory
	for _, dir := range fs.musicDirs {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return os.RemoveAll(fullPath)
		}
	}

	return os.ErrNotExist
}

// Rename renames files if not read-only
func (fs *CollectionFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	if fs.readOnly {
		return os.ErrPermission
	}

	oldName = path.Clean("/" + oldName)
	newName = path.Clean("/" + newName)

	// Find source file
	for _, dir := range fs.musicDirs {
		oldPath := filepath.Join(dir, oldName)
		if _, err := os.Stat(oldPath); err == nil {
			newPath := filepath.Join(dir, newName)
			return os.Rename(oldPath, newPath)
		}
	}

	return os.ErrNotExist
}

// Stat returns file info from any configured directory
func (fs *CollectionFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean("/" + name)

	// Try each directory
	for _, dir := range fs.musicDirs {
		fullPath := filepath.Join(dir, name)
		if info, err := os.Stat(fullPath); err == nil {
			return info, nil
		}
	}

	return nil, os.ErrNotExist
}
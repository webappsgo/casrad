// Package paths handles platform-specific path resolution
// See AI.md PART 4 for path specification
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// Directories holds all platform-specific directory paths
// See AI.md PART 4 for complete path specification per platform
type Directories struct {
	// Configuration files (server.yml)
	Config string
	// Application data
	Data string
	// Cache files
	Cache string
	// Log files
	Log string
	// Backup storage
	Backup string
	// SSL certificates (letsencrypt/, local/)
	SSL string
	// Security databases (geoip/, blocklists/, cve/, trivy/)
	Security string
	// SQLite database files
	Database string
	// PID file path
	PIDFile string
}

// IsPrivileged returns true if running as root/administrator
func IsPrivileged() bool {
	return os.Geteuid() == 0
}

// Get returns the directories for the current platform and privilege level
// See AI.md PART 4 for platform-specific paths
func Get() Directories {
	if IsPrivileged() {
		return getPrivileged()
	}
	return getUnprivileged()
}

// getPrivileged returns paths for privileged (root/administrator) execution
func getPrivileged() Directories {
	switch runtime.GOOS {
	case "darwin":
		base := "/Library/Application Support/casapps/casrad"
		return Directories{
			Config:   base,
			Data:     filepath.Join(base, "data"),
			Cache:    "/Library/Caches/casapps/casrad",
			Log:      "/Library/Logs/casapps/casrad",
			Backup:   "/Library/Backups/casapps/casrad",
			SSL:      filepath.Join(base, "ssl"),
			Security: filepath.Join(base, "security"),
			Database: filepath.Join(base, "db"),
			PIDFile:  "/var/run/casapps/casrad.pid",
		}

	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		base := filepath.Join(programData, "casapps", "casrad")
		return Directories{
			Config:   base,
			Data:     filepath.Join(base, "data"),
			Cache:    filepath.Join(base, "cache"),
			Log:      filepath.Join(base, "logs"),
			Backup:   filepath.Join(programData, "Backups", "casapps", "casrad"),
			SSL:      filepath.Join(base, "ssl"),
			Security: filepath.Join(base, "security"),
			Database: filepath.Join(base, "db"),
			// Windows uses Service Manager, no PID file
			PIDFile: "",
		}

	case "freebsd", "openbsd", "netbsd":
		return Directories{
			Config:   "/usr/local/etc/casapps/casrad",
			Data:     "/var/db/casapps/casrad",
			Cache:    "/var/cache/casapps/casrad",
			Log:      "/var/log/casapps/casrad",
			Backup:   "/var/backups/casapps/casrad",
			SSL:      "/usr/local/etc/casapps/casrad/ssl",
			Security: "/usr/local/etc/casapps/casrad/security",
			Database: "/var/db/casapps/casrad/db",
			PIDFile:  "/var/run/casapps/casrad.pid",
		}

	// Linux
	default:
		return Directories{
			Config:   "/etc/casapps/casrad",
			Data:     "/var/lib/casapps/casrad",
			Cache:    "/var/cache/casapps/casrad",
			Log:      "/var/log/casapps/casrad",
			Backup:   "/mnt/Backups/casapps/casrad",
			SSL:      "/etc/casapps/casrad/ssl",
			Security: "/etc/casapps/casrad/security",
			Database: "/var/lib/casapps/casrad/db",
			PIDFile:  "/var/run/casapps/casrad.pid",
		}
	}
}

// getUnprivileged returns paths for unprivileged (user) execution
func getUnprivileged() Directories {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		base := filepath.Join(home, "Library/Application Support/casapps/casrad")
		return Directories{
			Config:   base,
			Data:     base,
			Cache:    filepath.Join(home, "Library/Caches/casapps/casrad"),
			Log:      filepath.Join(home, "Library/Logs/casapps/casrad"),
			Backup:   filepath.Join(home, "Library/Backups/casapps/casrad"),
			SSL:      filepath.Join(base, "ssl"),
			Security: filepath.Join(base, "security"),
			Database: filepath.Join(base, "db"),
			PIDFile:  filepath.Join(base, "casrad.pid"),
		}

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		configBase := filepath.Join(appData, "casapps", "casrad")
		dataBase := filepath.Join(localAppData, "casapps", "casrad")
		return Directories{
			Config:   configBase,
			Data:     dataBase,
			Cache:    filepath.Join(dataBase, "cache"),
			Log:      filepath.Join(dataBase, "logs"),
			Backup:   filepath.Join(localAppData, "Backups", "casapps", "casrad"),
			SSL:      filepath.Join(configBase, "ssl"),
			Security: filepath.Join(configBase, "security"),
			Database: filepath.Join(dataBase, "db"),
			// No PID file for user-mode Windows
			PIDFile: "",
		}

	case "freebsd", "openbsd", "netbsd":
		// BSD user mode follows XDG-like conventions
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local/share")
		}
		cacheHome := os.Getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(home, ".cache")
		}
		configBase := filepath.Join(configHome, "casapps/casrad")
		dataBase := filepath.Join(dataHome, "casapps/casrad")
		return Directories{
			Config:   configBase,
			Data:     dataBase,
			Cache:    filepath.Join(cacheHome, "casapps/casrad"),
			Log:      filepath.Join(home, ".local/log/casapps/casrad"),
			Backup:   filepath.Join(dataHome, "Backups/casapps/casrad"),
			SSL:      filepath.Join(configBase, "ssl"),
			Security: filepath.Join(configBase, "security"),
			Database: filepath.Join(dataBase, "db"),
			PIDFile:  filepath.Join(dataBase, "casrad.pid"),
		}

	// Linux (XDG Base Directory Specification)
	default:
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local/share")
		}
		cacheHome := os.Getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(home, ".cache")
		}
		configBase := filepath.Join(configHome, "casapps/casrad")
		dataBase := filepath.Join(dataHome, "casapps/casrad")
		return Directories{
			Config:   configBase,
			Data:     dataBase,
			Cache:    filepath.Join(cacheHome, "casapps/casrad"),
			Log:      filepath.Join(home, ".local/log/casapps/casrad"),
			Backup:   filepath.Join(dataHome, "Backups/casapps/casrad"),
			SSL:      filepath.Join(configBase, "ssl"),
			Security: filepath.Join(configBase, "security"),
			Database: filepath.Join(dataBase, "db"),
			PIDFile:  filepath.Join(dataBase, "casrad.pid"),
		}
	}
}

// EnsureDirectories creates all required directories with appropriate permissions
func EnsureDirectories() error {
	dirs := Get()

	// Directories to create (order matters for nested paths)
	paths := []string{
		dirs.Config,
		dirs.Data,
		dirs.Cache,
		dirs.Log,
		dirs.Backup,
		dirs.SSL,
		dirs.Security,
		dirs.Database,
	}

	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(p, 0755); err != nil {
			return err
		}
	}

	// Create PID file directory if needed
	if dirs.PIDFile != "" {
		pidDir := filepath.Dir(dirs.PIDFile)
		if err := os.MkdirAll(pidDir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ConfigFile returns the path to the main config file (server.yml)
func ConfigFile() string {
	return filepath.Join(Get().Config, "server.yml")
}

// LogFile returns the path to the main log file
func LogFile() string {
	return filepath.Join(Get().Log, "server.log")
}

// ServerDB returns the path to the server database
func ServerDB() string {
	return filepath.Join(Get().Database, "server.db")
}

// UsersDB returns the path to the users database
func UsersDB() string {
	return filepath.Join(Get().Database, "users.db")
}

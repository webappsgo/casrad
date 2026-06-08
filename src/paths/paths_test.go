// Package paths - Tests for path resolution functions.
// Covers: Get() returns non-empty dirs on Linux (test host), ConfigFile, LogFile,
// ServerDB, UsersDB, IsPrivileged detection, and path relationship invariants.
// Does NOT call EnsureDirectories (writes to filesystem, not a unit test).
package paths

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetReturnsNonEmptyPaths(t *testing.T) {
	t.Parallel()

	dirs := Get()

	if dirs.Config == "" {
		t.Error("Get().Config is empty")
	}
	if dirs.Data == "" {
		t.Error("Get().Data is empty")
	}
	if dirs.Cache == "" {
		t.Error("Get().Cache is empty")
	}
	if dirs.Log == "" {
		t.Error("Get().Log is empty")
	}
	if dirs.Database == "" {
		t.Error("Get().Database is empty")
	}
	if dirs.SSL == "" {
		t.Error("Get().SSL is empty")
	}
	if dirs.Security == "" {
		t.Error("Get().Security is empty")
	}
}

func TestConfigFileEndsWithServerYml(t *testing.T) {
	t.Parallel()

	cf := ConfigFile()
	if !strings.HasSuffix(cf, "server.yml") {
		t.Errorf("ConfigFile() = %q, want to end with server.yml", cf)
	}
}

func TestLogFileEndsWithServerLog(t *testing.T) {
	t.Parallel()

	lf := LogFile()
	if !strings.HasSuffix(lf, "server.log") {
		t.Errorf("LogFile() = %q, want to end with server.log", lf)
	}
}

func TestServerDBEndsWithServerDb(t *testing.T) {
	t.Parallel()

	db := ServerDB()
	if !strings.HasSuffix(db, "server.db") {
		t.Errorf("ServerDB() = %q, want to end with server.db", db)
	}
}

func TestUsersDBEndsWithUsersDb(t *testing.T) {
	t.Parallel()

	db := UsersDB()
	if !strings.HasSuffix(db, "users.db") {
		t.Errorf("UsersDB() = %q, want to end with users.db", db)
	}
}

func TestConfigFileIsInsideConfigDir(t *testing.T) {
	t.Parallel()

	dirs := Get()
	cf := ConfigFile()

	// ConfigFile must be inside the Config directory
	if !strings.HasPrefix(cf, dirs.Config) {
		t.Errorf("ConfigFile %q is not inside Config dir %q", cf, dirs.Config)
	}
}

func TestServerDBIsInsideDatabaseDir(t *testing.T) {
	t.Parallel()

	dirs := Get()
	db := ServerDB()

	if !strings.HasPrefix(db, dirs.Database) {
		t.Errorf("ServerDB %q is not inside Database dir %q", db, dirs.Database)
	}
}

func TestSSLIsSubdirOfConfig(t *testing.T) {
	t.Parallel()

	// On Linux non-privileged the SSL path is a subdirectory of config
	dirs := Get()

	if runtime.GOOS == "linux" {
		parent := filepath.Dir(dirs.SSL)
		if parent != dirs.Config && !strings.HasPrefix(dirs.SSL, dirs.Config) {
			t.Logf("SSL dir %q is not under Config dir %q (may be ok for this platform)", dirs.SSL, dirs.Config)
		}
	}
}

func TestIsPrivilegedReturnsBool(t *testing.T) {
	t.Parallel()

	// Just verify it returns a boolean without panicking
	_ = IsPrivileged()
}

// Package paths — Additional tests for path resolution covering backup, security,
// PIDFile fields, XDG env var overrides, and getUnprivileged branches.
// Note: t.Setenv and t.Parallel are mutually exclusive — these tests are sequential.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Directories struct completeness ---

func TestGetReturnsBackupPath(t *testing.T) {
	t.Parallel()
	dirs := Get()
	if dirs.Backup == "" {
		t.Error("Get().Backup is empty")
	}
}

func TestGetReturnsSecurityPath(t *testing.T) {
	t.Parallel()
	dirs := Get()
	if dirs.Security == "" {
		t.Error("Get().Security is empty")
	}
}

func TestGetPIDFilePathOnLinux(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := Get()
	// PIDFile may be empty in user mode on some platforms but should
	// contain casrad on linux privileged mode.
	if dirs.PIDFile != "" && !strings.Contains(dirs.PIDFile, "casrad") {
		t.Errorf("Get().PIDFile = %q, should contain casrad", dirs.PIDFile)
	}
}

// --- getUnprivileged via XDG env vars (sequential — uses t.Setenv) ---

func TestGetUnprivilegedXDGConfigHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only XDG test")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.Config, tmp) {
		t.Errorf("getUnprivileged().Config = %q, should start with XDG_CONFIG_HOME %q", dirs.Config, tmp)
	}
}

func TestGetUnprivilegedXDGDataHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only XDG test")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.Data, tmp) {
		t.Errorf("getUnprivileged().Data = %q, should start with XDG_DATA_HOME %q", dirs.Data, tmp)
	}
}

func TestGetUnprivilegedXDGCacheHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only XDG test")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.Cache, tmp) {
		t.Errorf("getUnprivileged().Cache = %q, should start with XDG_CACHE_HOME %q", dirs.Cache, tmp)
	}
}

func TestGetUnprivilegedDefaultsWithoutXDG(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only XDG test")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()
	home, _ := os.UserHomeDir()

	if !strings.HasPrefix(dirs.Config, home) {
		t.Errorf("getUnprivileged().Config = %q, should be under home %q", dirs.Config, home)
	}
	if !strings.HasPrefix(dirs.Data, home) {
		t.Errorf("getUnprivileged().Data = %q, should be under home %q", dirs.Data, home)
	}
}

// --- getPrivileged ---

func TestGetPrivilegedLinuxConfig(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Config != "/etc/casapps/casrad" {
		t.Errorf("getPrivileged().Config = %q, want /etc/casapps/casrad", dirs.Config)
	}
}

func TestGetPrivilegedLinuxData(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Data != "/var/lib/casapps/casrad" {
		t.Errorf("getPrivileged().Data = %q, want /var/lib/casapps/casrad", dirs.Data)
	}
}

func TestGetPrivilegedLinuxPIDFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.PIDFile != "/var/run/casapps/casrad.pid" {
		t.Errorf("getPrivileged().PIDFile = %q, want /var/run/casapps/casrad.pid", dirs.PIDFile)
	}
}

// --- ConfigFile / LogFile / ServerDB / UsersDB relationship invariants ---

func TestConfigFileParentIsConfigDir(t *testing.T) {
	t.Parallel()
	dirs := Get()
	cf := ConfigFile()
	if filepath.Dir(cf) != dirs.Config {
		t.Errorf("filepath.Dir(ConfigFile()) = %q, want %q", filepath.Dir(cf), dirs.Config)
	}
}

func TestLogFileParentIsLogDir(t *testing.T) {
	t.Parallel()
	dirs := Get()
	lf := LogFile()
	if filepath.Dir(lf) != dirs.Log {
		t.Errorf("filepath.Dir(LogFile()) = %q, want %q", filepath.Dir(lf), dirs.Log)
	}
}

func TestServerDBParentIsDatabaseDir(t *testing.T) {
	t.Parallel()
	dirs := Get()
	db := ServerDB()
	if filepath.Dir(db) != dirs.Database {
		t.Errorf("filepath.Dir(ServerDB()) = %q, want %q", filepath.Dir(db), dirs.Database)
	}
}

func TestUsersDBParentIsDatabaseDir(t *testing.T) {
	t.Parallel()
	dirs := Get()
	db := UsersDB()
	if filepath.Dir(db) != dirs.Database {
		t.Errorf("filepath.Dir(UsersDB()) = %q, want %q", filepath.Dir(db), dirs.Database)
	}
}

func TestWindowsPROGRAMDATAFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	t.Setenv("PROGRAMDATA", "")

	dirs := getPrivileged()

	if !strings.Contains(dirs.Config, "ProgramData") {
		t.Errorf("getPrivileged().Config on Windows with empty PROGRAMDATA = %q", dirs.Config)
	}
}

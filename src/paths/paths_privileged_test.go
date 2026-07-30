// Package paths — Tests for privileged paths, EnsureDirectories, and unprivileged
// path construction with HOME env overrides.
//
// Coverage targets:
//   - getPrivileged(): Linux cache, log, backup, SSL, security, database fields (20% → ~90%)
//   - getUnprivileged(): HOME-based defaults on Linux (37% → ~80%)
//   - EnsureDirectories(): happy path as root, creates dirs, cleans up (0% → 100%)
//
// All t.Setenv tests are NOT parallel (t.Setenv and t.Parallel are incompatible).
// Pure value tests are parallel.
package paths

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// --- getPrivileged: verify every field on Linux ---

func TestGetPrivilegedLinuxCache(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Cache != "/var/cache/casapps/casrad" {
		t.Errorf("getPrivileged().Cache = %q, want /var/cache/casapps/casrad", dirs.Cache)
	}
}

func TestGetPrivilegedLinuxLog(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Log != "/var/log/casapps/casrad" {
		t.Errorf("getPrivileged().Log = %q, want /var/log/casapps/casrad", dirs.Log)
	}
}

func TestGetPrivilegedLinuxBackup(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Backup != "/mnt/Backups/casapps/casrad" {
		t.Errorf("getPrivileged().Backup = %q, want /mnt/Backups/casapps/casrad", dirs.Backup)
	}
}

func TestGetPrivilegedLinuxSSL(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.SSL != "/etc/casapps/casrad/ssl" {
		t.Errorf("getPrivileged().SSL = %q, want /etc/casapps/casrad/ssl", dirs.SSL)
	}
}

func TestGetPrivilegedLinuxSecurity(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Security != "/etc/casapps/casrad/security" {
		t.Errorf("getPrivileged().Security = %q, want /etc/casapps/casrad/security", dirs.Security)
	}
}

func TestGetPrivilegedLinuxDatabase(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	dirs := getPrivileged()
	if dirs.Database != "/var/lib/casapps/casrad/db" {
		t.Errorf("getPrivileged().Database = %q, want /var/lib/casapps/casrad/db", dirs.Database)
	}
}

// Verify all privileged fields are non-empty in one pass
func TestGetPrivilegedAllFieldsNonEmpty(t *testing.T) {
	t.Parallel()
	dirs := getPrivileged()
	fields := map[string]string{
		"Config":   dirs.Config,
		"Data":     dirs.Data,
		"Cache":    dirs.Cache,
		"Log":      dirs.Log,
		"Backup":   dirs.Backup,
		"SSL":      dirs.SSL,
		"Security": dirs.Security,
		"Database": dirs.Database,
	}
	for name, val := range fields {
		if val == "" {
			t.Errorf("getPrivileged().%s is empty", name)
		}
	}
}

// --- getUnprivileged: test with explicit HOME and no XDG vars ---

// TestGetUnprivilegedHomeBasedConfig verifies that without XDG vars set,
// the config path falls back to $HOME/.config/casapps/casrad.
func TestGetUnprivilegedHomeBasedConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	want := tmp + "/.config/casapps/casrad"
	if dirs.Config != want {
		t.Errorf("getUnprivileged().Config = %q, want %q", dirs.Config, want)
	}
}

// TestGetUnprivilegedHomeBasedData verifies $HOME/.local/share/casapps/casrad.
func TestGetUnprivilegedHomeBasedData(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	want := tmp + "/.local/share/casapps/casrad"
	if dirs.Data != want {
		t.Errorf("getUnprivileged().Data = %q, want %q", dirs.Data, want)
	}
}

// TestGetUnprivilegedHomeBasedCache verifies $HOME/.cache/casapps/casrad.
func TestGetUnprivilegedHomeBasedCache(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	want := tmp + "/.cache/casapps/casrad"
	if dirs.Cache != want {
		t.Errorf("getUnprivileged().Cache = %q, want %q", dirs.Cache, want)
	}
}

// TestGetUnprivilegedHomeBasedLog verifies $HOME/.local/log/casapps/casrad.
func TestGetUnprivilegedHomeBasedLog(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	want := tmp + "/.local/log/casapps/casrad"
	if dirs.Log != want {
		t.Errorf("getUnprivileged().Log = %q, want %q", dirs.Log, want)
	}
}

// TestGetUnprivilegedPIDFileUnderDataDir verifies PIDFile sits inside the data dir.
func TestGetUnprivilegedPIDFileUnderDataDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if dirs.PIDFile == "" {
		t.Error("getUnprivileged().PIDFile is empty on Linux")
	}
	if !strings.HasPrefix(dirs.PIDFile, dirs.Data) {
		t.Errorf("getUnprivileged().PIDFile = %q not under Data = %q", dirs.PIDFile, dirs.Data)
	}
	if !strings.HasSuffix(dirs.PIDFile, "casrad.pid") {
		t.Errorf("getUnprivileged().PIDFile = %q should end with casrad.pid", dirs.PIDFile)
	}
}

// TestGetUnprivilegedSSLUnderConfigDir verifies SSL dir is inside config dir.
func TestGetUnprivilegedSSLUnderConfigDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.SSL, dirs.Config) {
		t.Errorf("getUnprivileged().SSL = %q not under Config = %q", dirs.SSL, dirs.Config)
	}
}

// TestGetUnprivilegedSecurityUnderConfigDir verifies Security dir is inside config dir.
func TestGetUnprivilegedSecurityUnderConfigDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.Security, dirs.Config) {
		t.Errorf("getUnprivileged().Security = %q not under Config = %q", dirs.Security, dirs.Config)
	}
}

// TestGetUnprivilegedDatabaseUnderDataDir verifies Database dir is inside data dir.
func TestGetUnprivilegedDatabaseUnderDataDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dirs := getUnprivileged()

	if !strings.HasPrefix(dirs.Database, dirs.Data) {
		t.Errorf("getUnprivileged().Database = %q not under Data = %q", dirs.Database, dirs.Data)
	}
}

// --- EnsureDirectories ---

// TestEnsureDirectoriesCreatesAllDirs runs EnsureDirectories as the current user
// and verifies all directories are actually created on disk.
// When running as root in the container this tests the privileged Linux paths.
// Cleanup removes the top-level directories created under /etc/casapps, /var/lib/casapps, etc.
func TestEnsureDirectoriesCreatesAllDirs(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	err := EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories() returned error: %v", err)
	}

	dirs := Get()

	// Verify each directory was created
	for _, p := range []string{
		dirs.Config,
		dirs.Data,
		dirs.Cache,
		dirs.Log,
		dirs.SSL,
		dirs.Security,
		dirs.Database,
	} {
		if p == "" {
			continue
		}
		info, statErr := os.Stat(p)
		if statErr != nil {
			t.Errorf("EnsureDirectories did not create %q: %v", p, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("EnsureDirectories created %q but it is not a directory", p)
		}
	}

	// Verify PID directory was created
	if dirs.PIDFile != "" {
		pidDir := "/var/run/casapps"
		info, statErr := os.Stat(pidDir)
		if statErr != nil {
			t.Errorf("EnsureDirectories did not create PID dir %q: %v", pidDir, statErr)
		} else if !info.IsDir() {
			t.Errorf("%q is not a directory", pidDir)
		}
	}

	// Cleanup: remove the top-level directories this test created.
	// Only remove casapps subtrees, not host system dirs.
	t.Cleanup(func() {
		for _, root := range []string{
			"/etc/casapps",
			"/var/lib/casapps",
			"/var/cache/casapps",
			"/var/log/casapps",
			"/var/run/casapps",
		} {
			os.RemoveAll(root)
		}
	})
}

// TestEnsureDirectoriesIdempotent verifies that calling EnsureDirectories twice
// does not return an error (directories already exist).
func TestEnsureDirectoriesIdempotent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	// First call
	if err := EnsureDirectories(); err != nil {
		t.Fatalf("first EnsureDirectories() error: %v", err)
	}
	// Second call must also succeed
	if err := EnsureDirectories(); err != nil {
		t.Fatalf("second EnsureDirectories() (idempotent) error: %v", err)
	}

	t.Cleanup(func() {
		for _, root := range []string{
			"/etc/casapps",
			"/var/lib/casapps",
			"/var/cache/casapps",
			"/var/log/casapps",
			"/var/run/casapps",
		} {
			os.RemoveAll(root)
		}
	})
}

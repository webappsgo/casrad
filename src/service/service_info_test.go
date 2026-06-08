// Package service — Tests for Manager info, path helpers, and findAvailableUID.
// Covers: systemdUnitPath, launchdPlistPath, findAvailableUID, Info, IsInstalled,
// IsRunning (no running service → false).
package service

import (
	"path/filepath"
	"strings"
	"testing"
)

// --- systemdUnitPath ---

func TestSystemdUnitPathUsesServiceName(t *testing.T) {
	t.Parallel()
	m := NewManager("myapp")
	got := m.systemdUnitPath()
	want := "/etc/systemd/system/myapp.service"
	if got != want {
		t.Errorf("systemdUnitPath() = %q, want %q", got, want)
	}
}

func TestSystemdUnitPathDefaultName(t *testing.T) {
	t.Parallel()
	m := NewManager("")
	got := m.systemdUnitPath()
	if !strings.HasSuffix(got, "casrad.service") {
		t.Errorf("systemdUnitPath() = %q, should end with casrad.service", got)
	}
}

// --- launchdPlistPath ---

func TestLaunchdPlistPathContainsName(t *testing.T) {
	t.Parallel()
	m := NewManager("myapp")
	got := m.launchdPlistPath()
	if !strings.Contains(got, "myapp") {
		t.Errorf("launchdPlistPath() = %q, should contain myapp", got)
	}
}

func TestLaunchdPlistPathIsPlist(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	got := m.launchdPlistPath()
	if filepath.Ext(got) != ".plist" {
		t.Errorf("launchdPlistPath() = %q, should have .plist extension", got)
	}
}

func TestLaunchdPlistPathContainsCasapps(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	got := m.launchdPlistPath()
	if !strings.Contains(got, "casapps") {
		t.Errorf("launchdPlistPath() = %q, should contain casapps prefix", got)
	}
}

// --- findAvailableUID ---

func TestFindAvailableUIDReturnsValidRange(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	uid := m.findAvailableUID(999, 100)
	if uid != 0 && (uid < 100 || uid > 999) {
		t.Errorf("findAvailableUID(999, 100) = %d, out of range [100, 999]", uid)
	}
}

func TestFindAvailableUIDImpossibleRangeReturnsZero(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	// max < min → should return 0 (loop never executes)
	uid := m.findAvailableUID(50, 100)
	if uid != 0 {
		t.Errorf("findAvailableUID(50, 100) = %d, want 0 (inverted range)", uid)
	}
}

// --- IsInstalled ---

func TestIsInstalledReturnsBool(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad-nonexistent-test-service")
	// A service named casrad-nonexistent-test-service should not be installed.
	// IsInstalled returns false when the service file doesn't exist.
	got := m.IsInstalled()
	if got {
		t.Error("IsInstalled() for non-existent service should return false")
	}
}

// --- IsRunning ---

func TestIsRunningReturnsFalseForNonexistentService(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad-nonexistent-test-service")
	// Status() will fail for a non-existent service, so IsRunning returns false.
	got := m.IsRunning()
	if got {
		t.Error("IsRunning() for non-existent service should return false")
	}
}

// --- Info ---

func TestInfoContainsExpectedKeys(t *testing.T) {
	t.Parallel()
	m := NewManager("infoapp")
	info := m.Info()

	expected := []string{
		"name", "display_name", "description", "binary_path",
		"user", "group", "work_dir", "log_dir", "data_dir",
		"config_dir", "type", "installed", "running",
	}
	for _, key := range expected {
		if _, ok := info[key]; !ok {
			t.Errorf("Info() missing key %q", key)
		}
	}
}

func TestInfoNameMatchesConfig(t *testing.T) {
	t.Parallel()
	m := NewManager("infoapp")
	info := m.Info()
	if info["name"] != "infoapp" {
		t.Errorf("Info()[name] = %q, want infoapp", info["name"])
	}
}

func TestInfoTypeIsString(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	info := m.Info()
	if _, ok := info["type"].(string); !ok {
		t.Errorf("Info()[type] = %T, want string", info["type"])
	}
}

func TestInfoInstalledIsBool(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	info := m.Info()
	if _, ok := info["installed"].(bool); !ok {
		t.Errorf("Info()[installed] = %T, want bool", info["installed"])
	}
}

func TestInfoRunningIsBool(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	info := m.Info()
	if _, ok := info["running"].(bool); !ok {
		t.Errorf("Info()[running] = %T, want bool", info["running"])
	}
}

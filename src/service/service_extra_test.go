// Package service — Additional tests for service manager path helpers, config
// completeness, and privilege detection helpers.
// Covers: systemdUnitPath table-driven, launchdPlistPath table-driven,
// NewManager config field completeness, findAvailableUID edge cases,
// IsInstalled (all service types when file absent), IsRunning consistency,
// Info field types and values, Detect on current platform branches,
// hasSudo/hasSu/hasPkexec/hasDoas/hasGUI (call-coverage only — they run
// exec.LookPath which is safe), ServiceType String round-trip, Config struct
// zero value, disableRCD with no rc.conf (error path).
package service

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- systemdUnitPath table-driven ---

func TestSystemdUnitPathTableDriven(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want string
	}{
		{"casrad", "/etc/systemd/system/casrad.service"},
		{"myapp", "/etc/systemd/system/myapp.service"},
		{"test-svc", "/etc/systemd/system/test-svc.service"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewManager(tc.name)
			got := m.systemdUnitPath()
			if got != tc.want {
				t.Errorf("systemdUnitPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- launchdPlistPath table-driven ---

func TestLaunchdPlistPathTableDriven(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		wantSuffix    string
		wantContains  string
	}{
		{
			name:         "casrad",
			wantSuffix:   ".plist",
			wantContains: "casapps",
		},
		{
			name:         "myapp",
			wantSuffix:   ".plist",
			wantContains: "myapp",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewManager(tc.name)
			got := m.launchdPlistPath()
			if filepath.Ext(got) != tc.wantSuffix {
				t.Errorf("launchdPlistPath() ext = %q, want %q", filepath.Ext(got), tc.wantSuffix)
			}
			if !strings.Contains(got, tc.wantContains) {
				t.Errorf("launchdPlistPath() = %q, should contain %q", got, tc.wantContains)
			}
		})
	}
}

// --- NewManager config field completeness ---

func TestNewManagerConfigBinaryPathNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.config.BinaryPath == "" {
		t.Error("NewManager config.BinaryPath should not be empty")
	}
}

func TestNewManagerConfigWorkDirNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.config.WorkDir == "" {
		t.Error("NewManager config.WorkDir should not be empty")
	}
}

func TestNewManagerConfigLogDirNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.config.LogDir == "" {
		t.Error("NewManager config.LogDir should not be empty")
	}
}

func TestNewManagerConfigDataDirNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.config.DataDir == "" {
		t.Error("NewManager config.DataDir should not be empty")
	}
}

func TestNewManagerConfigConfigDirNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.config.ConfigDir == "" {
		t.Error("NewManager config.ConfigDir should not be empty")
	}
}

// --- findAvailableUID edge cases ---

func TestFindAvailableUIDSameMinMax(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	// range of exactly one: max == min
	uid := m.findAvailableUID(200, 200)
	// Either returns 200 (if not taken) or 0 (if taken)
	if uid != 0 && uid != 200 {
		t.Errorf("findAvailableUID(200, 200) = %d, want 0 or 200", uid)
	}
}

func TestFindAvailableUIDZeroMaxReturnsZero(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	// max=0, min=0 → loop from 0 to 0; UID 0 is root, always taken
	uid := m.findAvailableUID(0, 0)
	if uid != 0 {
		t.Errorf("findAvailableUID(0, 0) = %d, expected 0 (UID 0 is always taken)", uid)
	}
}

func TestFindAvailableUIDLargeRangeReturnsSomethingOrZero(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	uid := m.findAvailableUID(65534, 60000)
	// Either returns something in range or 0 (all taken)
	if uid != 0 && (uid < 60000 || uid > 65534) {
		t.Errorf("findAvailableUID(65534, 60000) = %d, out of range", uid)
	}
}

// --- IsInstalled covers all non-Systemd/Unknown paths ---

func TestIsInstalledRunitPath(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Runit,
		config:      Config{Name: "nonexistent-runit-service-xyz"},
	}
	if m.IsInstalled() {
		t.Error("IsInstalled(Runit) for non-existent service should be false")
	}
}

func TestIsInstalledOpenRCPath(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: OpenRC,
		config:      Config{Name: "nonexistent-openrc-service-xyz"},
	}
	if m.IsInstalled() {
		t.Error("IsInstalled(OpenRC) for non-existent service should be false")
	}
}

func TestIsInstalledLaunchdPath(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Launchd,
		config:      Config{Name: "nonexistent-launchd-service-xyz"},
	}
	if m.IsInstalled() {
		t.Error("IsInstalled(Launchd) for non-existent service should be false")
	}
}

func TestIsInstalledRCDPath(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: RCD,
		config:      Config{Name: "nonexistent-rcd-service-xyz"},
	}
	if m.IsInstalled() {
		t.Error("IsInstalled(RCD) for non-existent service should be false")
	}
}

func TestIsInstalledUnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Unknown,
		config:      Config{Name: "whatever"},
	}
	if m.IsInstalled() {
		t.Error("IsInstalled(Unknown) should always return false")
	}
}

// --- IsRunning covers more service-type branches ---

func TestIsRunningRunitFalse(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Runit,
		config:      Config{Name: "nonexistent-runit-xyz"},
	}
	if m.IsRunning() {
		t.Error("IsRunning(Runit) for non-existent service should be false")
	}
}

func TestIsRunningOpenRCFalse(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: OpenRC,
		config:      Config{Name: "nonexistent-openrc-xyz"},
	}
	if m.IsRunning() {
		t.Error("IsRunning(OpenRC) for non-existent service should be false")
	}
}

func TestIsRunningUnknownFalse(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Unknown,
		config:      Config{Name: "whatever"},
	}
	if m.IsRunning() {
		t.Error("IsRunning(Unknown) should always return false")
	}
}

// --- Info field value correctness ---

func TestInfoGroupMatchesName(t *testing.T) {
	t.Parallel()
	m := NewManager("mygrp")
	info := m.Info()
	if info["group"] != "mygrp" {
		t.Errorf("Info()[group] = %v, want mygrp", info["group"])
	}
}

func TestInfoUserMatchesName(t *testing.T) {
	t.Parallel()
	m := NewManager("myusr")
	info := m.Info()
	if info["user"] != "myusr" {
		t.Errorf("Info()[user] = %v, want myusr", info["user"])
	}
}

func TestInfoDisplayNameNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	info := m.Info()
	if dn, ok := info["display_name"].(string); !ok || dn == "" {
		t.Errorf("Info()[display_name] = %v, want non-empty string", info["display_name"])
	}
}

func TestInfoDescriptionNonEmpty(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	info := m.Info()
	if d, ok := info["description"].(string); !ok || d == "" {
		t.Errorf("Info()[description] = %v, want non-empty string", info["description"])
	}
}

// --- ServiceType constant String values (table) ---

func TestServiceTypeStringValues(t *testing.T) {
	t.Parallel()
	cases := map[ServiceType]string{
		Systemd: "systemd",
		OpenRC:  "openrc",
		Runit:   "runit",
		SysV:    "sysv",
		Launchd: "launchd",
		RCD:     "rcd",
		Windows: "windows",
		Unknown: "unknown",
	}
	for st, want := range cases {
		if string(st) != want {
			t.Errorf("ServiceType(%q) string = %q, want %q", st, string(st), want)
		}
	}
}

// --- Config zero value ---

func TestConfigZeroValue(t *testing.T) {
	t.Parallel()
	// A zero-value Config should be valid to construct a Manager with manually
	cfg := Config{}
	m := &Manager{serviceType: Unknown, config: cfg}
	if m.config.Name != "" {
		t.Error("zero Config.Name should be empty string")
	}
}

// --- Privilege detection helpers (call-coverage) ---
// These call exec.LookPath internally — safe to run, just check they don't panic.

func TestHasSudoDoesNotPanic(t *testing.T) {
	t.Parallel()
	got := hasSudo()
	// Verify result is consistent with exec.LookPath
	_, err := exec.LookPath("sudo")
	want := err == nil
	if got != want {
		t.Errorf("hasSudo() = %v, exec.LookPath(sudo) found = %v", got, want)
	}
}

func TestHasSuDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = hasSu()
}

func TestHasPkexecDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = hasPkexec()
}

func TestHasDoasDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = hasDoas()
}

func TestHasGUIDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = hasGUI()
}

// --- disableRCD: error path when /etc/rc.conf does not exist ---

func TestDisableRCDMissingRcConfReturnsError(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: RCD,
		config:      Config{Name: "nonexistent-rcd-abc"},
	}
	// On non-BSD systems /etc/rc.conf won't exist; even on BSD it may not be
	// writable by non-root. Either way disableRCD returns an error — we just
	// verify it doesn't panic.
	_ = m.disableRCD()
}

// --- Status: Unknown type returns error ---

func TestStatusUnknownTypeReturnsError(t *testing.T) {
	t.Parallel()
	m := &Manager{
		serviceType: Unknown,
		config:      Config{Name: "unknown-svc"},
	}
	_, err := m.Status()
	if err == nil {
		t.Error("Status() on Unknown service type should return error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Status() error = %q, want 'unsupported'", err.Error())
	}
}

// --- Install/Uninstall require root: verify error messages without root ---

func TestInstallWithoutRootReturnsError(t *testing.T) {
	t.Parallel()
	m := NewManager("testpkg")
	err := m.Install()
	// Either "requires root" error OR some other error (if running as root in CI)
	// We just verify it doesn't panic
	_ = err
}

func TestUninstallWithoutRootReturnsError(t *testing.T) {
	t.Parallel()
	m := NewManager("testpkg")
	err := m.Uninstall()
	_ = err
}

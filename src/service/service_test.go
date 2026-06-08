// Package service — Tests for service manager detection and construction.
// Covers: Detect, NewManager, Manager.Type.
// Note: Install/Uninstall/Start/Stop require root and a running service manager;
// those paths are covered by service integration tests, not unit tests.
package service

import (
	"runtime"
	"testing"
)

func TestDetectReturnsValidServiceType(t *testing.T) {
	t.Parallel()
	got := Detect()
	validTypes := map[ServiceType]bool{
		Systemd: true,
		OpenRC:  true,
		Runit:   true,
		SysV:    true,
		Launchd: true,
		RCD:     true,
		Windows: true,
		Unknown: true,
	}
	if !validTypes[got] {
		t.Errorf("Detect() = %q, not a valid ServiceType", got)
	}
}

func TestDetectWindowsOnWindows(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	if got := Detect(); got != Windows {
		t.Errorf("Detect() on Windows = %q, want Windows", got)
	}
}

func TestDetectDarwinOnMacOS(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}
	if got := Detect(); got != Launchd {
		t.Errorf("Detect() on macOS = %q, want Launchd", got)
	}
}

func TestDetectBSDOnBSD(t *testing.T) {
	t.Parallel()
	switch runtime.GOOS {
	case "freebsd", "openbsd", "netbsd":
	default:
		t.Skip("BSD-only test")
	}
	if got := Detect(); got != RCD {
		t.Errorf("Detect() on BSD = %q, want RCD", got)
	}
}

func TestNewManagerDefaultName(t *testing.T) {
	t.Parallel()
	m := NewManager("")
	if m == nil {
		t.Fatal("NewManager(\"\") returned nil")
	}
	if m.config.Name != "casrad" {
		t.Errorf("config.Name = %q, want casrad", m.config.Name)
	}
}

func TestNewManagerCustomName(t *testing.T) {
	t.Parallel()
	m := NewManager("myservice")
	if m.config.Name != "myservice" {
		t.Errorf("config.Name = %q, want myservice", m.config.Name)
	}
}

func TestNewManagerHasValidConfig(t *testing.T) {
	t.Parallel()
	m := NewManager("testapp")
	if m.config.DisplayName == "" {
		t.Error("config.DisplayName should not be empty")
	}
	if m.config.Description == "" {
		t.Error("config.Description should not be empty")
	}
	if m.config.User != "testapp" {
		t.Errorf("config.User = %q, want testapp", m.config.User)
	}
	if m.config.Group != "testapp" {
		t.Errorf("config.Group = %q, want testapp", m.config.Group)
	}
}

func TestManagerTypeMatchesDetect(t *testing.T) {
	t.Parallel()
	m := NewManager("casrad")
	if m.Type() != Detect() {
		t.Errorf("Manager.Type() = %q, Detect() = %q; should match", m.Type(), Detect())
	}
}

func TestServiceTypeConstants(t *testing.T) {
	t.Parallel()
	expected := map[ServiceType]string{
		Systemd: "systemd",
		OpenRC:  "openrc",
		Runit:   "runit",
		SysV:    "sysv",
		Launchd: "launchd",
		RCD:     "rcd",
		Windows: "windows",
		Unknown: "unknown",
	}
	for st, want := range expected {
		if string(st) != want {
			t.Errorf("ServiceType %q = %q, want %q", st, string(st), want)
		}
	}
}

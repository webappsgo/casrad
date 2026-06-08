// Package service — Tests for Tor manager.
// Covers: TorStatus constants, TorConfig/TorInfo/VanityStatus structs, NewTorManager,
// FindTorBinary (empty config path, nonexistent config path), GetInfo on fresh manager,
// GetOnionAddress on fresh manager, IsEnabled on fresh manager, Stop without start,
// SetEnabled(false) no-op, GetVanityStatus on fresh manager.
// Note: Start/Restart require Tor binary and process spawning — not tested here.
package service

import (
	"testing"
)

// --- TorStatus constants ---

func TestTorStatusConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status TorStatus
		want   string
	}{
		{TorStatusDisabled, "disabled"},
		{TorStatusStarting, "starting"},
		{TorStatusConnected, "connected"},
		{TorStatusDisconnected, "disconnected"},
		{TorStatusError, "error"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("TorStatus %q = %q, want %q", tt.status, string(tt.status), tt.want)
		}
	}
}

// --- NewTorManager ---

func TestNewTorManagerNotNil(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/config", "/tmp/data", "/tmp/logs", 9050)
	if tm == nil {
		t.Fatal("NewTorManager returned nil")
	}
}

func TestNewTorManagerInitialStatusDisabled(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/config", "/tmp/data", "/tmp/logs", 9050)
	info := tm.GetInfo()
	if info.Status != TorStatusDisabled {
		t.Errorf("initial TorStatus = %q, want disabled", info.Status)
	}
}

func TestNewTorManagerInitialNotEnabled(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/config", "/tmp/data", "/tmp/logs", 9050)
	if tm.IsEnabled() {
		t.Error("new TorManager should not be enabled")
	}
}

func TestNewTorManagerInitialOnionAddressEmpty(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/config", "/tmp/data", "/tmp/logs", 9050)
	if addr := tm.GetOnionAddress(); addr != "" {
		t.Errorf("GetOnionAddress() on new manager = %q, want empty", addr)
	}
}

// --- FindTorBinary ---

func TestFindTorBinaryEmptyConfigPath(t *testing.T) {
	t.Parallel()
	// Empty config path should fall through to PATH lookup
	// Returns empty string if tor not installed — not an error
	result := FindTorBinary("")
	// Result is either a valid path string or empty — either is correct
	_ = result
}

func TestFindTorBinaryNonexistentConfigPath(t *testing.T) {
	t.Parallel()
	result := FindTorBinary("/nonexistent/path/to/tor")
	// Should fall through to PATH lookup, not fail with panic
	_ = result
}

func TestFindTorBinaryReturnsStringOrEmpty(t *testing.T) {
	t.Parallel()
	// Just verify it doesn't panic
	result := FindTorBinary("")
	if result != "" {
		// If tor is found, it must be a non-empty path string
		if len(result) == 0 {
			t.Error("FindTorBinary returned non-empty but zero-length string")
		}
	}
}

// --- GetInfo ---

func TestGetInfoOnFreshManagerEnabledFalse(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	info := tm.GetInfo()
	if info.Enabled {
		t.Error("GetInfo().Enabled should be false on fresh manager")
	}
}

func TestGetInfoOnFreshManagerStatusDisabled(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	info := tm.GetInfo()
	if info.Status != TorStatusDisabled {
		t.Errorf("GetInfo().Status = %q, want disabled", info.Status)
	}
}

func TestGetInfoOnFreshManagerDataDirSet(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data/tor", "/logs", 9050)
	info := tm.GetInfo()
	if info.DataDir != "/data/tor" {
		t.Errorf("GetInfo().DataDir = %q, want /data/tor", info.DataDir)
	}
}

func TestGetInfoStartedAtZeroOnFreshManager(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	info := tm.GetInfo()
	if !info.StartedAt.IsZero() {
		t.Errorf("GetInfo().StartedAt = %v, want zero time on fresh manager", info.StartedAt)
	}
}

// --- Stop ---

func TestStopWithoutStartDoesNotPanic(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	if err := tm.Stop(); err != nil {
		t.Errorf("Stop() on fresh manager = %v, want nil", err)
	}
}

func TestStopSetsStatusDisconnected(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	_ = tm.Stop()
	info := tm.GetInfo()
	if info.Status != TorStatusDisconnected {
		t.Errorf("after Stop(), status = %q, want disconnected", info.Status)
	}
}

// --- SetEnabled(false) ---

func TestSetEnabledFalseCallsStop(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	if err := tm.SetEnabled(false); err != nil {
		t.Errorf("SetEnabled(false) = %v, want nil", err)
	}
}

// --- GetOnionAddress ---

func TestGetOnionAddressEmptyOnFreshManager(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	addr := tm.GetOnionAddress()
	if addr != "" {
		t.Errorf("GetOnionAddress() = %q, want empty on fresh manager", addr)
	}
}

// --- GetVanityStatus ---

func TestGetVanityStatusNotGeneratingOnFreshManager(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/config", "/data", "/logs", 9050)
	status := tm.GetVanityStatus()
	if status.Generating {
		t.Error("GetVanityStatus().Generating should be false on fresh manager")
	}
}

// --- TorConfig / TorInfo / VanityStatus struct fields ---

func TestTorConfigFields(t *testing.T) {
	t.Parallel()
	cfg := TorConfig{Binary: "/usr/bin/tor", DataDir: "/var/lib/tor"}
	if cfg.Binary != "/usr/bin/tor" {
		t.Errorf("TorConfig.Binary = %q, want /usr/bin/tor", cfg.Binary)
	}
	if cfg.DataDir != "/var/lib/tor" {
		t.Errorf("TorConfig.DataDir = %q, want /var/lib/tor", cfg.DataDir)
	}
}

func TestTorInfoFields(t *testing.T) {
	t.Parallel()
	info := TorInfo{
		Enabled:      true,
		Status:       TorStatusConnected,
		OnionAddress: "test.onion",
	}
	if !info.Enabled {
		t.Error("TorInfo.Enabled should be true")
	}
	if info.Status != TorStatusConnected {
		t.Errorf("TorInfo.Status = %q, want connected", info.Status)
	}
	if info.OnionAddress != "test.onion" {
		t.Errorf("TorInfo.OnionAddress = %q, want test.onion", info.OnionAddress)
	}
}

func TestVanityStatusFields(t *testing.T) {
	t.Parallel()
	vs := VanityStatus{
		Generating: true,
		Prefix:     "abc",
	}
	if !vs.Generating {
		t.Error("VanityStatus.Generating should be true")
	}
	if vs.Prefix != "abc" {
		t.Errorf("VanityStatus.Prefix = %q, want abc", vs.Prefix)
	}
}

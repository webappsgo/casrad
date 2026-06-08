// Package handler — Tests for admin handler pure helper functions and admin route validation.
// Covers: formatUptime, formatMemory, formatInt, itoa, ftoa,
// ValidateAdminRoute, ValidateAdminPath, AdminRoutePaths, AdminAPIRoutePaths,
// NewAdminHandler.
package handler

import (
	"testing"
	"time"
)

// --- formatUptime ---

func TestFormatUptimeMinutesOnly(t *testing.T) {
	t.Parallel()
	got := formatUptime(45 * time.Minute)
	want := "45m"
	if got != want {
		t.Errorf("formatUptime(45m) = %q, want %q", got, want)
	}
}

func TestFormatUptimeZero(t *testing.T) {
	t.Parallel()
	got := formatUptime(0)
	want := "0m"
	if got != want {
		t.Errorf("formatUptime(0) = %q, want %q", got, want)
	}
}

func TestFormatUptimeHoursAndMinutes(t *testing.T) {
	t.Parallel()
	got := formatUptime(2*time.Hour + 30*time.Minute)
	want := "2h 30m"
	if got != want {
		t.Errorf("formatUptime(2h30m) = %q, want %q", got, want)
	}
}

func TestFormatUptimeDays(t *testing.T) {
	t.Parallel()
	got := formatUptime(25 * time.Hour)
	want := "1d 1h"
	if got != want {
		t.Errorf("formatUptime(25h) = %q, want %q", got, want)
	}
}

// --- formatMemory ---

func TestFormatMemoryBytes(t *testing.T) {
	t.Parallel()
	got := formatMemory(512)
	want := "512 B"
	if got != want {
		t.Errorf("formatMemory(512) = %q, want %q", got, want)
	}
}

func TestFormatMemoryKilobytes(t *testing.T) {
	t.Parallel()
	got := formatMemory(1024)
	want := "1.0 KB"
	if got != want {
		t.Errorf("formatMemory(1024) = %q, want %q", got, want)
	}
}

func TestFormatMemoryMegabytes(t *testing.T) {
	t.Parallel()
	got := formatMemory(1024 * 1024)
	want := "1.0 MB"
	if got != want {
		t.Errorf("formatMemory(1MB) = %q, want %q", got, want)
	}
}

func TestFormatMemoryZero(t *testing.T) {
	t.Parallel()
	got := formatMemory(0)
	want := "0 B"
	if got != want {
		t.Errorf("formatMemory(0) = %q, want %q", got, want)
	}
}

// --- formatInt ---

func TestFormatIntPositive(t *testing.T) {
	t.Parallel()
	got := formatInt(42)
	want := "42"
	if got != want {
		t.Errorf("formatInt(42) = %q, want %q", got, want)
	}
}

func TestFormatIntZero(t *testing.T) {
	t.Parallel()
	got := formatInt(0)
	want := "0"
	if got != want {
		t.Errorf("formatInt(0) = %q, want %q", got, want)
	}
}

// --- itoa ---

func TestItoaPositive(t *testing.T) {
	t.Parallel()
	got := itoa(123)
	want := "123"
	if got != want {
		t.Errorf("itoa(123) = %q, want %q", got, want)
	}
}

func TestItoaZero(t *testing.T) {
	t.Parallel()
	got := itoa(0)
	want := "0"
	if got != want {
		t.Errorf("itoa(0) = %q, want %q", got, want)
	}
}

func TestItoaNegative(t *testing.T) {
	t.Parallel()
	got := itoa(-7)
	want := "-7"
	if got != want {
		t.Errorf("itoa(-7) = %q, want %q", got, want)
	}
}

// --- ftoa ---

func TestFtoaOneDecimal(t *testing.T) {
	t.Parallel()
	got := ftoa(1.5)
	want := "1.5"
	if got != want {
		t.Errorf("ftoa(1.5) = %q, want %q", got, want)
	}
}

func TestFtoaZeroDecimal(t *testing.T) {
	t.Parallel()
	got := ftoa(2.0)
	want := "2.0"
	if got != want {
		t.Errorf("ftoa(2.0) = %q, want %q", got, want)
	}
}

// --- ValidateAdminRoute ---

func TestValidateAdminRouteRootIsValid(t *testing.T) {
	t.Parallel()
	err := ValidateAdminRoute("")
	if err != nil {
		t.Errorf("ValidateAdminRoute(\"\") error = %v, want nil", err)
	}
}

func TestValidateAdminRouteProfileIsValid(t *testing.T) {
	t.Parallel()
	err := ValidateAdminRoute("profile")
	if err != nil {
		t.Errorf("ValidateAdminRoute(profile) error = %v, want nil", err)
	}
}

func TestValidateAdminRouteServerIsValid(t *testing.T) {
	t.Parallel()
	err := ValidateAdminRoute("server/settings")
	if err != nil {
		t.Errorf("ValidateAdminRoute(server/settings) error = %v, want nil", err)
	}
}

func TestValidateAdminRouteInvalidSegmentReturnsError(t *testing.T) {
	t.Parallel()
	err := ValidateAdminRoute("unknown")
	if err == nil {
		t.Error("ValidateAdminRoute(unknown) should return error")
	}
}

// --- ValidateAdminPath ---

func TestValidateAdminPathValidPath(t *testing.T) {
	t.Parallel()
	err := ValidateAdminPath("mgmt")
	if err != nil {
		t.Errorf("ValidateAdminPath(mgmt) error = %v, want nil", err)
	}
}

func TestValidateAdminPathTooShort(t *testing.T) {
	t.Parallel()
	err := ValidateAdminPath("a")
	if err == nil {
		t.Error("ValidateAdminPath(a) should return error (too short)")
	}
}

func TestValidateAdminPathTooLong(t *testing.T) {
	t.Parallel()
	err := ValidateAdminPath("this-path-is-way-too-long-to-be-valid-here-12345678")
	if err == nil {
		t.Error("ValidateAdminPath(too long) should return error")
	}
}

func TestValidateAdminPathReservedWordReturnsError(t *testing.T) {
	t.Parallel()
	for _, reserved := range ReservedAdminPaths {
		err := ValidateAdminPath(reserved)
		if err == nil {
			t.Errorf("ValidateAdminPath(%q) should return error (reserved)", reserved)
		}
	}
}

func TestValidateAdminPathLeadingHyphenReturnsError(t *testing.T) {
	t.Parallel()
	err := ValidateAdminPath("-bad")
	if err == nil {
		t.Error("ValidateAdminPath(-bad) should return error")
	}
}

func TestValidateAdminPathUppercaseNormalized(t *testing.T) {
	t.Parallel()
	// Uppercase is normalized to lowercase before validation
	err := ValidateAdminPath("MGMT")
	if err != nil {
		t.Errorf("ValidateAdminPath(MGMT) error = %v, want nil (normalizes to mgmt)", err)
	}
}

// --- AdminRoutePaths ---

func TestAdminRoutePathsContainsDashboard(t *testing.T) {
	t.Parallel()
	paths := AdminRoutePaths("admin")
	if _, ok := paths["dashboard"]; !ok {
		t.Error("AdminRoutePaths should contain 'dashboard' key")
	}
}

func TestAdminRoutePathsUsesAdminPath(t *testing.T) {
	t.Parallel()
	paths := AdminRoutePaths("mgmt")
	if paths["dashboard"] != "/mgmt" {
		t.Errorf("dashboard path = %q, want /mgmt", paths["dashboard"])
	}
}

func TestAdminRoutePathsContainsServerUsers(t *testing.T) {
	t.Parallel()
	paths := AdminRoutePaths("admin")
	if _, ok := paths["server_users"]; !ok {
		t.Error("AdminRoutePaths should contain 'server_users'")
	}
}

// --- AdminAPIRoutePaths ---

func TestAdminAPIRoutePathsContainsProfile(t *testing.T) {
	t.Parallel()
	paths := AdminAPIRoutePaths("admin", "v1")
	if _, ok := paths["api_profile"]; !ok {
		t.Error("AdminAPIRoutePaths should contain 'api_profile'")
	}
}

func TestAdminAPIRoutePathsUsesVersionAndAdminPath(t *testing.T) {
	t.Parallel()
	paths := AdminAPIRoutePaths("mgmt", "v1")
	want := "/api/v1/mgmt/profile"
	if paths["api_profile"] != want {
		t.Errorf("api_profile = %q, want %q", paths["api_profile"], want)
	}
}

// --- NewAdminHandler ---

func TestNewAdminHandlerReturnsNonNil(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	if h == nil {
		t.Error("NewAdminHandler returned nil")
	}
}

// Package admin — Tests for pure helper functions.
// Covers: formatDuration, formatBytes, selectedIf, New, Path.
package admin

import (
	"testing"
	"time"
)

// --- formatDuration ---

func TestFormatDurationDays(t *testing.T) {
	t.Parallel()
	got := formatDuration(50 * time.Hour)
	want := "2d 2h 0m"
	if got != want {
		t.Errorf("formatDuration(50h) = %q, want %q", got, want)
	}
}

func TestFormatDurationHoursOnly(t *testing.T) {
	t.Parallel()
	got := formatDuration(3 * time.Hour)
	want := "3h 0m"
	if got != want {
		t.Errorf("formatDuration(3h) = %q, want %q", got, want)
	}
}

func TestFormatDurationMinutesOnly(t *testing.T) {
	t.Parallel()
	got := formatDuration(45 * time.Minute)
	want := "45m"
	if got != want {
		t.Errorf("formatDuration(45m) = %q, want %q", got, want)
	}
}

func TestFormatDurationZero(t *testing.T) {
	t.Parallel()
	got := formatDuration(0)
	want := "0m"
	if got != want {
		t.Errorf("formatDuration(0) = %q, want %q", got, want)
	}
}

func TestFormatDurationHoursWithMinutes(t *testing.T) {
	t.Parallel()
	got := formatDuration(2*time.Hour + 30*time.Minute)
	want := "2h 30m"
	if got != want {
		t.Errorf("formatDuration(2h30m) = %q, want %q", got, want)
	}
}

func TestFormatDurationDaysWithHoursAndMinutes(t *testing.T) {
	t.Parallel()
	got := formatDuration(25*time.Hour + 30*time.Minute)
	want := "1d 1h 30m"
	if got != want {
		t.Errorf("formatDuration(25h30m) = %q, want %q", got, want)
	}
}

// --- formatBytes ---

func TestFormatBytesBytes(t *testing.T) {
	t.Parallel()
	got := formatBytes(512)
	want := "512 B"
	if got != want {
		t.Errorf("formatBytes(512) = %q, want %q", got, want)
	}
}

func TestFormatBytesKilobytes(t *testing.T) {
	t.Parallel()
	got := formatBytes(1024)
	want := "1.0 KB"
	if got != want {
		t.Errorf("formatBytes(1024) = %q, want %q", got, want)
	}
}

func TestFormatBytesMegabytes(t *testing.T) {
	t.Parallel()
	got := formatBytes(1024 * 1024)
	want := "1.0 MB"
	if got != want {
		t.Errorf("formatBytes(1MB) = %q, want %q", got, want)
	}
}

func TestFormatBytesGigabytes(t *testing.T) {
	t.Parallel()
	got := formatBytes(1024 * 1024 * 1024)
	want := "1.0 GB"
	if got != want {
		t.Errorf("formatBytes(1GB) = %q, want %q", got, want)
	}
}

func TestFormatBytesZero(t *testing.T) {
	t.Parallel()
	got := formatBytes(0)
	want := "0 B"
	if got != want {
		t.Errorf("formatBytes(0) = %q, want %q", got, want)
	}
}

func TestFormatBytesPartialKB(t *testing.T) {
	t.Parallel()
	got := formatBytes(1536)
	want := "1.5 KB"
	if got != want {
		t.Errorf("formatBytes(1536) = %q, want %q", got, want)
	}
}

// --- selectedIf ---

func TestSelectedIfTrue(t *testing.T) {
	t.Parallel()
	got := selectedIf(true)
	if got != " selected" {
		t.Errorf("selectedIf(true) = %q, want \" selected\"", got)
	}
}

func TestSelectedIfFalse(t *testing.T) {
	t.Parallel()
	got := selectedIf(false)
	if got != "" {
		t.Errorf("selectedIf(false) = %q, want \"\"", got)
	}
}

// --- New and Path ---

func TestNewAdminDefaultsPath(t *testing.T) {
	t.Parallel()
	a := New(Config{AdminPath: ""})
	if a == nil {
		t.Fatal("New returned nil")
	}
	if a.Path() != "admin" {
		t.Errorf("New(empty path).Path() = %q, want admin", a.Path())
	}
}

func TestNewAdminCustomPath(t *testing.T) {
	t.Parallel()
	a := New(Config{AdminPath: "mgmt"})
	if a.Path() != "mgmt" {
		t.Errorf("New(mgmt).Path() = %q, want mgmt", a.Path())
	}
}

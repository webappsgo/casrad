// Package service — Tests for UpdateService pure constructors and configuration.
// Covers: DefaultUpdateConfig, NewUpdateService, Configure, SetBranch, GetBranch,
// GetLastCheckTime (initial), GetLastInfo, IsUpdateInProgress, GetCurrentVersion,
// GetBinaryPath, GetAssetName, ShouldAutoCheck, ValidateBranch, GetBinaryDir,
// matchesBranch (stable, beta, daily), isNewerVersion, findAsset (no match).
package service

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- DefaultUpdateConfig ---

func TestDefaultUpdateConfigReturnsNonNil(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	if cfg == nil {
		t.Fatal("DefaultUpdateConfig returned nil")
	}
}

func TestDefaultUpdateConfigBranchIsStable(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	if cfg.Branch != BranchStable {
		t.Errorf("Branch = %q, want stable", cfg.Branch)
	}
}

func TestDefaultUpdateConfigAutoCheckEnabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	if !cfg.AutoCheck {
		t.Error("AutoCheck should be true by default")
	}
}

func TestDefaultUpdateConfigCheckInterval24h(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	if cfg.CheckInterval != 24*time.Hour {
		t.Errorf("CheckInterval = %v, want 24h", cfg.CheckInterval)
	}
}

func TestDefaultUpdateConfigRepoOwnerIsOrg(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	if cfg.RepoOwner != "casapps" {
		t.Errorf("RepoOwner = %q, want casapps", cfg.RepoOwner)
	}
}

// --- NewUpdateService ---

func TestNewUpdateServiceReturnsNonNil(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/usr/local/bin/casrad", nil)
	if svc == nil {
		t.Fatal("NewUpdateService returned nil")
	}
}

func TestNewUpdateServiceNilConfigUsesDefaults(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if svc.config == nil {
		t.Error("NewUpdateService(nil config) should use default config")
	}
}

func TestNewUpdateServicePreservesVersion(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("2.3.4", "/bin/casrad", nil)
	if svc.GetCurrentVersion() != "2.3.4" {
		t.Errorf("GetCurrentVersion() = %q, want 2.3.4", svc.GetCurrentVersion())
	}
}

func TestNewUpdateServicePreservesBinaryPath(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/usr/bin/casrad", nil)
	if svc.GetBinaryPath() != "/usr/bin/casrad" {
		t.Errorf("GetBinaryPath() = %q, want /usr/bin/casrad", svc.GetBinaryPath())
	}
}

// --- Configure ---

func TestConfigureSetsNewBranch(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	svc.Configure(&UpdateConfig{
		Branch:        BranchBeta,
		AutoCheck:     false,
		CheckInterval: 48 * time.Hour,
		RepoOwner:     "casapps",
		RepoName:      "casrad",
	})
	if svc.GetBranch() != BranchBeta {
		t.Errorf("after Configure, Branch = %q, want beta", svc.GetBranch())
	}
}

// --- SetBranch / GetBranch ---

func TestSetBranchStable(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if err := svc.SetBranch("stable"); err != nil {
		t.Fatalf("SetBranch(stable) error = %v", err)
	}
	if svc.GetBranch() != BranchStable {
		t.Errorf("GetBranch() = %q, want stable", svc.GetBranch())
	}
}

func TestSetBranchBeta(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if err := svc.SetBranch("beta"); err != nil {
		t.Fatalf("SetBranch(beta) error = %v", err)
	}
	if svc.GetBranch() != BranchBeta {
		t.Errorf("GetBranch() = %q, want beta", svc.GetBranch())
	}
}

func TestSetBranchDaily(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if err := svc.SetBranch("daily"); err != nil {
		t.Fatalf("SetBranch(daily) error = %v", err)
	}
	if svc.GetBranch() != BranchDaily {
		t.Errorf("GetBranch() = %q, want daily", svc.GetBranch())
	}
}

func TestSetBranchInvalidReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	err := svc.SetBranch("nightly")
	if err != ErrInvalidBranch {
		t.Errorf("SetBranch(nightly) = %v, want ErrInvalidBranch", err)
	}
}

// --- GetLastCheckTime (initial zero) ---

func TestGetLastCheckTimeInitiallyZero(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if !svc.GetLastCheckTime().IsZero() {
		t.Error("GetLastCheckTime() should be zero before any check")
	}
}

// --- GetLastInfo ---

func TestGetLastInfoInitiallyNil(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if svc.GetLastInfo() != nil {
		t.Error("GetLastInfo() should be nil before any check")
	}
}

// --- IsUpdateInProgress ---

func TestIsUpdateInProgressInitiallyFalse(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if svc.IsUpdateInProgress() {
		t.Error("IsUpdateInProgress() should be false initially")
	}
}

// --- GetAssetName ---

func TestGetAssetNameContainsPlatform(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	name := svc.GetAssetName()
	if !strings.Contains(name, runtime.GOOS) {
		t.Errorf("GetAssetName() = %q, should contain %q", name, runtime.GOOS)
	}
	if !strings.Contains(name, runtime.GOARCH) {
		t.Errorf("GetAssetName() = %q, should contain %q", name, runtime.GOARCH)
	}
}

func TestGetAssetNameWindowsHasExe(t *testing.T) {
	t.Parallel()
	// Only verifiable on windows; on other platforms verify it does NOT have .exe
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	name := svc.GetAssetName()
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(name, ".exe") {
			t.Errorf("GetAssetName() on windows should end with .exe, got %q", name)
		}
	} else {
		if strings.HasSuffix(name, ".exe") {
			t.Errorf("GetAssetName() on %s should not end with .exe, got %q", runtime.GOOS, name)
		}
	}
}

// --- ShouldAutoCheck ---

func TestShouldAutoCheckFalseWhenDisabled(t *testing.T) {
	t.Parallel()
	cfg := &UpdateConfig{
		Branch:        BranchStable,
		AutoCheck:     false,
		CheckInterval: 24 * time.Hour,
		RepoOwner:     "casapps",
		RepoName:      "casrad",
	}
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	if svc.ShouldAutoCheck() {
		t.Error("ShouldAutoCheck() should be false when AutoCheck disabled")
	}
}

func TestShouldAutoCheckTrueWhenNeverChecked(t *testing.T) {
	t.Parallel()
	cfg := &UpdateConfig{
		Branch:        BranchStable,
		AutoCheck:     true,
		CheckInterval: 1 * time.Millisecond,
		RepoOwner:     "casapps",
		RepoName:      "casrad",
	}
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	if !svc.ShouldAutoCheck() {
		t.Error("ShouldAutoCheck() should be true when never checked and interval passed")
	}
}

// --- ValidateBranch ---

func TestValidateBranchStable(t *testing.T) {
	t.Parallel()
	if err := ValidateBranch("stable"); err != nil {
		t.Errorf("ValidateBranch(stable) = %v, want nil", err)
	}
}

func TestValidateBranchBeta(t *testing.T) {
	t.Parallel()
	if err := ValidateBranch("beta"); err != nil {
		t.Errorf("ValidateBranch(beta) = %v, want nil", err)
	}
}

func TestValidateBranchDaily(t *testing.T) {
	t.Parallel()
	if err := ValidateBranch("daily"); err != nil {
		t.Errorf("ValidateBranch(daily) = %v, want nil", err)
	}
}

func TestValidateBranchInvalidReturnsError(t *testing.T) {
	t.Parallel()
	if err := ValidateBranch("nightly"); err != ErrInvalidBranch {
		t.Errorf("ValidateBranch(nightly) = %v, want ErrInvalidBranch", err)
	}
}

func TestValidateBranchCaseInsensitive(t *testing.T) {
	t.Parallel()
	if err := ValidateBranch("STABLE"); err != nil {
		t.Errorf("ValidateBranch(STABLE) = %v, want nil (case insensitive)", err)
	}
}

// --- GetBinaryDir ---

func TestGetBinaryDirReturnsDirectory(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/usr/local/bin/casrad", nil)
	dir := svc.GetBinaryDir()
	if dir != "/usr/local/bin" {
		t.Errorf("GetBinaryDir() = %q, want /usr/local/bin", dir)
	}
}

// --- matchesBranch ---

func TestMatchesBranchStableRelease(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	release := &GitHubRelease{TagName: "v1.2.3", Prerelease: false}
	if !svc.matchesBranch(release) {
		t.Error("matchesBranch(stable, v1.2.3) should match")
	}
}

func TestMatchesBranchStableIgnoresPrerelease(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	release := &GitHubRelease{TagName: "v1.2.3-beta", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(stable, prerelease) should not match")
	}
}

func TestMatchesBranchBeta(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchBeta
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "v1.3.0-beta", Prerelease: true}
	if !svc.matchesBranch(release) {
		t.Error("matchesBranch(beta, v1.3.0-beta) should match")
	}
}

func TestMatchesBranchBetaNonBetaDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchBeta
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "v1.3.0-rc1", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(beta, v1.3.0-rc1) should not match")
	}
}

func TestMatchesBranchDailyTimestamp(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchDaily
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "20250101120000", Prerelease: true}
	if !svc.matchesBranch(release) {
		t.Error("matchesBranch(daily, 20250101120000) should match")
	}
}

func TestMatchesBranchDailyNonTimestampDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchDaily
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "v1.0.0-daily", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(daily, non-timestamp) should not match")
	}
}

// --- isNewerVersion ---

func TestIsNewerVersionTrue(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if !svc.isNewerVersion("1.1.0") {
		t.Error("isNewerVersion(1.1.0) should be true when current is 1.0.0")
	}
}

func TestIsNewerVersionFalse(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.1.0", "/bin/casrad", nil)
	if svc.isNewerVersion("1.0.0") {
		t.Error("isNewerVersion(1.0.0) should be false when current is 1.1.0")
	}
}

func TestIsNewerVersionSameVersionFalse(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	if svc.isNewerVersion("1.0.0") {
		t.Error("isNewerVersion(same) should be false")
	}
}

func TestIsNewerVersionStripsVPrefix(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("v1.0.0", "/bin/casrad", nil)
	if !svc.isNewerVersion("v1.1.0") {
		t.Error("isNewerVersion with v-prefix should compare correctly")
	}
}

// --- findAsset (no matching platform asset) ---

func TestFindAssetNoMatchReturnsEmpty(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	assets := []GitHubAsset{
		{Name: "casrad-wrongos-wrongarch", BrowserDownloadURL: "https://example.com/casrad"},
	}
	url, size := svc.findAsset(assets)
	if url != "" || size != 0 {
		t.Errorf("findAsset(no match) = (%q, %d), want (\"\", 0)", url, size)
	}
}

func TestFindAssetMatchReturnsURL(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	expected := svc.GetAssetName()
	assets := []GitHubAsset{
		{Name: expected, BrowserDownloadURL: "https://example.com/" + expected, Size: 12345},
	}
	url, size := svc.findAsset(assets)
	if url == "" {
		t.Errorf("findAsset(matching %q) returned empty URL", expected)
	}
	if size != 12345 {
		t.Errorf("findAsset size = %d, want 12345", size)
	}
}

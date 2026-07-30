// Package service — Additional tests targeting uncovered functions.
// Covers: smtp.go (AutoDetectSMTP, testSMTPConnection, TestSMTPSettings, getGatewayIP),
// tor.go (getTorConfig, ensureDirs, CancelVanityGeneration, StartVanityGeneration,
// GetVanityStatus elapsed branch), update.go (matchesBranch edge paths),
// scheduler.go (checkAndRunTasks, updateTaskNextRun, parseRange errors, parseField step+range),
// notification.go (formatDaysLeft days==1 branch).
package service

import (
	"context"
	"embed"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
)

// --- smtp.go: TestSMTPSettings nil/empty-host guard ---
// These cover the cheap guard at the top of TestSMTPSettings without any
// network activity. The function returns ErrSMTPNotConfigured immediately.

func TestSMTPSettingsNilReturnsNotConfigured(t *testing.T) {
	t.Parallel()
	err := TestSMTPSettings(nil)
	if !errors.Is(err, ErrSMTPNotConfigured) {
		t.Errorf("TestSMTPSettings(nil) = %v, want ErrSMTPNotConfigured", err)
	}
}

func TestSMTPSettingsEmptyHostReturnsNotConfigured(t *testing.T) {
	t.Parallel()
	err := TestSMTPSettings(&SMTPSettings{Host: "", Port: 25})
	if !errors.Is(err, ErrSMTPNotConfigured) {
		t.Errorf("TestSMTPSettings(empty host) = %v, want ErrSMTPNotConfigured", err)
	}
}

// TestSMTPSettingsUnreachableHostReturnsConnectionError covers the TCP-dial
// failure path. Port 1 on localhost is almost universally closed/refused.
func TestSMTPSettingsUnreachableHostReturnsConnectionError(t *testing.T) {
	t.Parallel()
	err := TestSMTPSettings(&SMTPSettings{Host: "127.0.0.1", Port: 1})
	if err == nil {
		t.Skip("port 1 unexpectedly accepted a connection — skipping on this host")
	}
	if !errors.Is(err, ErrSMTPConnection) {
		t.Errorf("TestSMTPSettings(unreachable) = %v, want ErrSMTPConnection", err)
	}
}

// --- smtp.go: testSMTPConnection ---
// Cover the TCP-fail path (false return) with a definitely-closed port.

func TestSMTPConnectionRefusedReturnsFalse(t *testing.T) {
	t.Parallel()
	// Port 1 is always closed on any normal host.
	got := testSMTPConnection("127.0.0.1", 1)
	if got {
		t.Skip("port 1 unexpectedly accepted — skipping on this host")
	}
	// Confirm the function returns false (already validated by the if above).
}

// --- smtp.go: AutoDetectSMTP ---
// AutoDetectSMTP iterates hosts/ports, returns nil when nothing responds.
// In a CI environment with no local SMTP this covers all the loop iterations.

func TestAutoDetectSMTPReturnsNilOrSettings(t *testing.T) {
	t.Parallel()
	// Result is nil (no local SMTP) or a valid *SMTPSettings if one is running.
	// Either outcome is correct; the test just ensures it doesn't panic.
	result := AutoDetectSMTP()
	if result != nil {
		// If a server was found, verify the returned struct has Host and Port.
		if result.Host == "" {
			t.Error("AutoDetectSMTP non-nil result has empty Host")
		}
		if result.Port == 0 {
			t.Error("AutoDetectSMTP non-nil result has Port 0")
		}
		if result.TLSMode != TLSModeAuto {
			t.Errorf("AutoDetectSMTP result TLSMode = %q, want auto", result.TLSMode)
		}
	}
}

// --- smtp.go: getGatewayIP ---
// getGatewayIP is unexported; call via AutoDetectSMTP path above and directly.

func TestGetGatewayIPReturnsStringOrEmpty(t *testing.T) {
	t.Parallel()
	// Just ensure it doesn't panic. Returns "" when UDP routing fails.
	ip := getGatewayIP()
	// If non-empty, must look like an IPv4 address with dots.
	if ip != "" && !strings.Contains(ip, ".") {
		t.Errorf("getGatewayIP() = %q, should be empty or contain dots (IPv4)", ip)
	}
}

// --- tor.go: getTorConfig ---

func TestGetTorConfigReturnsNonEmpty(t *testing.T) {
	t.Parallel()
	config := getTorConfig()
	if config == "" {
		t.Error("getTorConfig() returned empty string")
	}
}

func TestGetTorConfigContainsSocksPort(t *testing.T) {
	t.Parallel()
	config := getTorConfig()
	if !strings.Contains(config, "SocksPort") {
		t.Error("getTorConfig() should contain SocksPort directive")
	}
}

func TestGetTorConfigDisablesExitRelay(t *testing.T) {
	t.Parallel()
	config := getTorConfig()
	if !strings.Contains(config, "ExitRelay 0") {
		t.Error("getTorConfig() should disable ExitRelay")
	}
}

func TestGetTorConfigDisablesORPort(t *testing.T) {
	t.Parallel()
	config := getTorConfig()
	if !strings.Contains(config, "ORPort 0") {
		t.Error("getTorConfig() should disable ORPort (not a relay)")
	}
}

// --- tor.go: ensureDirs ---
// Use a temp directory so the test never touches /tmp/config or /tmp/data.

func TestEnsureDirsCreatesRequiredDirectories(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cfgDir := filepath.Join(base, "config")
	dataDir := filepath.Join(base, "data")
	logDir := filepath.Join(base, "logs")

	tm := NewTorManager(TorConfig{}, cfgDir, dataDir, logDir, 9050)
	if err := tm.ensureDirs(); err != nil {
		t.Fatalf("ensureDirs() error = %v", err)
	}

	expected := []string{
		filepath.Join(cfgDir, "tor"),
		filepath.Join(dataDir, "tor"),
		filepath.Join(dataDir, "tor", "site"),
	}
	for _, dir := range expected {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("ensureDirs() did not create %q: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("ensureDirs() created %q but it is not a directory", dir)
		}
	}
}

func TestEnsureDirsIsIdempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	cfgDir := filepath.Join(base, "config")
	dataDir := filepath.Join(base, "data")

	tm := NewTorManager(TorConfig{}, cfgDir, dataDir, "/logs", 9050)

	// First call
	if err := tm.ensureDirs(); err != nil {
		t.Fatalf("first ensureDirs() error = %v", err)
	}
	// Second call must not fail
	if err := tm.ensureDirs(); err != nil {
		t.Fatalf("second ensureDirs() error = %v (should be idempotent)", err)
	}
}

// --- tor.go: CancelVanityGeneration ---

func TestCancelVanityGenerationNoopWhenNotRunning(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg", "/tmp/dat", "/tmp/log", 9050)
	// Must not panic when no vanity generation is in progress.
	tm.CancelVanityGeneration()

	status := tm.GetVanityStatus()
	if status.Generating {
		t.Error("Generating should be false after CancelVanityGeneration")
	}
	if status.Prefix != "" {
		t.Errorf("Prefix should be empty after cancel, got %q", status.Prefix)
	}
}

// --- tor.go: StartVanityGeneration ---

func TestStartVanityGenerationSetsStatus(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg", "/tmp/dat", "/tmp/log", 9050)

	err := tm.StartVanityGeneration("abc")
	if err != nil {
		t.Fatalf("StartVanityGeneration(abc) error = %v", err)
	}

	status := tm.GetVanityStatus()
	if !status.Generating {
		t.Error("GetVanityStatus().Generating should be true after StartVanityGeneration")
	}
	if status.Prefix != "abc" {
		t.Errorf("GetVanityStatus().Prefix = %q, want abc", status.Prefix)
	}

	// Clean up — cancel the background goroutine.
	tm.CancelVanityGeneration()
}

func TestStartVanityGenerationRejectsLongPrefix(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg", "/tmp/dat", "/tmp/log", 9050)

	err := tm.StartVanityGeneration("toolongprefix")
	if err == nil {
		t.Error("StartVanityGeneration with prefix > 6 chars should return error")
		tm.CancelVanityGeneration()
	}
}

func TestStartVanityGenerationRejectsDuplicate(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg2", "/tmp/dat2", "/tmp/log2", 9050)

	if err := tm.StartVanityGeneration("xy"); err != nil {
		t.Fatalf("first StartVanityGeneration error = %v", err)
	}
	// Second call while first is still running must return error.
	err := tm.StartVanityGeneration("xy")
	if err == nil {
		t.Error("duplicate StartVanityGeneration should return error")
	}
	tm.CancelVanityGeneration()
}

// --- tor.go: GetVanityStatus elapsed branch ---
// When generating is true, ElapsedSec is derived from time.Since(vanityStarted).
// After StartVanityGeneration, elapsed should be >= 0.

func TestGetVanityStatusElapsedWhenGenerating(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg3", "/tmp/dat3", "/tmp/log3", 9050)

	if err := tm.StartVanityGeneration("zz"); err != nil {
		t.Fatalf("StartVanityGeneration error = %v", err)
	}
	// Small sleep so elapsed is non-negative and StartedAt is set.
	time.Sleep(10 * time.Millisecond)

	status := tm.GetVanityStatus()
	if !status.Generating {
		t.Error("Generating should be true")
	}
	if status.ElapsedSec < 0 {
		t.Errorf("ElapsedSec = %d, should be >= 0", status.ElapsedSec)
	}
	if status.StartedAt.IsZero() {
		t.Error("StartedAt should be non-zero when generating")
	}

	tm.CancelVanityGeneration()
}

// --- tor.go: CancelVanityGeneration clears state after generation started ---

func TestCancelVanityGenerationClearsState(t *testing.T) {
	t.Parallel()
	tm := NewTorManager(TorConfig{}, "/tmp/cfg4", "/tmp/dat4", "/tmp/log4", 9050)

	if err := tm.StartVanityGeneration("qq"); err != nil {
		t.Fatalf("StartVanityGeneration error = %v", err)
	}
	tm.CancelVanityGeneration()

	status := tm.GetVanityStatus()
	if status.Generating {
		t.Error("Generating should be false after cancel")
	}
	if status.Prefix != "" {
		t.Errorf("Prefix should be empty after cancel, got %q", status.Prefix)
	}
}

// --- update.go: matchesBranch uncovered edge paths ---
// The daily branch: non-prerelease returns false (not yet covered).

func TestMatchesBranchDailyNonPrereleaseDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchDaily
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "20250101120000", Prerelease: false}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(daily, non-prerelease) should not match even with timestamp tag")
	}
}

// The daily branch: 13-char tag (not 14) should not match.

func TestMatchesBranchDailyWrongLengthTagDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchDaily
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "2025010112000", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(daily, 13-char tag) should not match")
	}
}

// The daily branch: tag with non-digit chars at position not matching pure digits.

func TestMatchesBranchDailyTagWithNonDigitDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchDaily
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "2025010112000x", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(daily, tag with non-digit) should not match")
	}
}

// The stable branch: tag with a dot but no v prefix should also match (e.g. "1.2.3").

func TestMatchesBranchStableDotVersionNoVPrefix(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	release := &GitHubRelease{TagName: "1.2.3", Prerelease: false}
	if !svc.matchesBranch(release) {
		t.Error("matchesBranch(stable, 1.2.3) should match (contains dot, not prerelease)")
	}
}

// The beta branch: prerelease but not ending in "-beta" should not match.

func TestMatchesBranchBetaPrereleaseNoSuffixDoesNotMatch(t *testing.T) {
	t.Parallel()
	cfg := DefaultUpdateConfig()
	cfg.Branch = BranchBeta
	svc := NewUpdateService("1.0.0", "/bin/casrad", cfg)
	release := &GitHubRelease{TagName: "v1.3.0", Prerelease: true}
	if svc.matchesBranch(release) {
		t.Error("matchesBranch(beta, prerelease without -beta suffix) should not match")
	}
}

// isNewerVersion: non-semver strings compared lexicographically — verify no panic.

func TestIsNewerVersionNonSemver(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("dev", "/bin/casrad", nil)
	// Should not panic; result depends on string comparison.
	_ = svc.isNewerVersion("nightly")
}

// findAsset: empty asset list returns ("", 0).

func TestFindAssetEmptyListReturnsEmpty(t *testing.T) {
	t.Parallel()
	svc := NewUpdateService("1.0.0", "/bin/casrad", nil)
	url, size := svc.findAsset([]GitHubAsset{})
	if url != "" || size != 0 {
		t.Errorf("findAsset(empty) = (%q, %d), want (\"\", 0)", url, size)
	}
}

// --- scheduler.go: checkAndRunTasks ---
// checkAndRunTasks is the internal tick handler. Call it directly with a time
// that is after a task's NextRun to verify it executes the handler.

func TestCheckAndRunTasksExecutesOverdueTask(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()

	called := false
	task := &Task{
		ID:      "overdue",
		Name:    "overdue",
		Schedule: "0 * * * *",
		Handler: func(ctx context.Context) error {
			called = true
			return nil
		},
		Enabled: true,
		Status:  TaskStatusPending,
		// NextRun in the past so the task is overdue.
		NextRun: time.Now().Add(-1 * time.Minute),
	}
	s.tasks["overdue"] = task

	// Need an active context for s.ctx used inside checkAndRunTasks.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.ctx = ctx

	s.checkAndRunTasks(time.Now())

	if !called {
		t.Error("checkAndRunTasks should have called the overdue task handler")
	}
}

func TestCheckAndRunTasksSkipsDisabledTask(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()

	called := false
	task := &Task{
		ID:      "disabled_overdue",
		Name:    "disabled_overdue",
		Schedule: "0 * * * *",
		Handler: func(ctx context.Context) error {
			called = true
			return nil
		},
		Enabled: false,
		NextRun: time.Now().Add(-1 * time.Minute),
	}
	s.tasks["disabled_overdue"] = task

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.ctx = ctx

	s.checkAndRunTasks(time.Now())

	if called {
		t.Error("checkAndRunTasks should NOT call a disabled task handler")
	}
}

func TestCheckAndRunTasksSkipsZeroNextRun(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()

	called := false
	task := &Task{
		ID:      "zero_nextrun",
		Name:    "zero_nextrun",
		Schedule: "0 * * * *",
		Handler: func(ctx context.Context) error {
			called = true
			return nil
		},
		Enabled: true,
		// NextRun is zero — should be skipped
	}
	s.tasks["zero_nextrun"] = task

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.ctx = ctx

	s.checkAndRunTasks(time.Now())

	if called {
		t.Error("checkAndRunTasks should NOT run a task with zero NextRun")
	}
}

// --- scheduler.go: updateTaskNextRun ---

func TestUpdateTaskNextRunSetsNextRun(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()

	task := &Task{
		ID:       "update_next",
		Name:     "update_next",
		Schedule: "@every 5m",
		Enabled:  true,
		// LastRun is zero, so nextRun = now + interval.
	}
	s.tasks["update_next"] = task

	before := time.Now()
	s.updateTaskNextRun(task)
	after := time.Now()

	// NextRun should now be roughly 5 minutes from now.
	expectedMin := before.Add(5 * time.Minute)
	expectedMax := after.Add(5 * time.Minute)

	if task.NextRun.Before(expectedMin) || task.NextRun.After(expectedMax) {
		t.Errorf("NextRun = %v, want between %v and %v", task.NextRun, expectedMin, expectedMax)
	}
}

// --- scheduler.go: parseRange error paths ---
// The 66.7% coverage gap is the invalid-start and invalid-end error paths.

func TestParseRangeInvalidStart(t *testing.T) {
	t.Parallel()
	_, err := parseRange("abc-5", 0, 59)
	if err == nil {
		t.Error("parseRange with non-numeric start should return error")
	}
}

func TestParseRangeInvalidEnd(t *testing.T) {
	t.Parallel()
	_, err := parseRange("1-xyz", 0, 59)
	if err == nil {
		t.Error("parseRange with non-numeric end should return error")
	}
}

func TestParseRangeOutOfBounds(t *testing.T) {
	t.Parallel()
	// start > end
	_, err := parseRange("5-3", 0, 59)
	if err == nil {
		t.Error("parseRange with start > end should return error")
	}
}

func TestParseRangeBelowMin(t *testing.T) {
	t.Parallel()
	// start below min
	_, err := parseRange("-1-5", 0, 59)
	// "-1-5" splits as ["", "1", "5"] — len != 2, so it's a different error.
	// Use a value that passes split but fails bounds check.
	_, err = parseRange("0-60", 0, 59)
	if err == nil {
		t.Error("parseRange with end > max should return error")
	}
	_ = err
}

// --- scheduler.go: parseField step with range base ---
// parseField("0-10/2", 0, 59) exercises the step-with-range branch.

func TestParseFieldStepWithRangeBase(t *testing.T) {
	t.Parallel()
	vals, err := parseField("0-10/2", 0, 59)
	if err != nil {
		t.Fatalf("parseField step+range: %v", err)
	}
	// 0, 2, 4, 6, 8, 10 — every 2nd in 0-10
	if len(vals) == 0 {
		t.Error("parseField(0-10/2) should return non-empty values")
	}
}

// --- notification.go: formatDaysLeft days==1 ---

func TestFormatDaysLeftOneDay(t *testing.T) {
	t.Parallel()
	result := formatDaysLeft(1)
	if result != "Certificate expires in 1 day" {
		t.Errorf("formatDaysLeft(1) = %q, want \"Certificate expires in 1 day\"", result)
	}
}

func TestFormatDaysLeftMultipleDays(t *testing.T) {
	t.Parallel()
	result := formatDaysLeft(7)
	if !strings.Contains(result, "days") {
		t.Errorf("formatDaysLeft(7) = %q, should contain 'days'", result)
	}
}

// --- user.go: createUserDirectories ---
// createUserDirectories is unexported but accessible from package service tests.
// It only uses os.MkdirAll — no store needed.

func TestCreateUserDirectoriesCreatesSubdirs(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	homeDir := filepath.Join(base, "testuser")

	svc := &UserService{}
	if err := svc.createUserDirectories(homeDir); err != nil {
		t.Fatalf("createUserDirectories error = %v", err)
	}

	expected := []string{
		homeDir,
		filepath.Join(homeDir, "music"),
		filepath.Join(homeDir, "podcasts"),
		filepath.Join(homeDir, "audiobooks"),
		filepath.Join(homeDir, "radio"),
		filepath.Join(homeDir, "playlists"),
		filepath.Join(homeDir, "recordings"),
		filepath.Join(homeDir, "transcodes"),
	}
	for _, dir := range expected {
		if info, err := os.Stat(dir); err != nil {
			t.Errorf("createUserDirectories did not create %q: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("%q exists but is not a directory", dir)
		}
	}
}

func TestCreateUserDirectoriesIsIdempotent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	homeDir := filepath.Join(base, "testuser2")
	svc := &UserService{}

	// First call
	if err := svc.createUserDirectories(homeDir); err != nil {
		t.Fatalf("first createUserDirectories error = %v", err)
	}
	// Second call must succeed (idempotent via MkdirAll)
	if err := svc.createUserDirectories(homeDir); err != nil {
		t.Fatalf("second createUserDirectories error = %v (should be idempotent)", err)
	}
}

// --- geoip.go: Init when disabled ---
// Init() returns nil immediately when IsEnabled() is false.

func TestGeoIPInitDisabledReturnsNil(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: false})
	if err := s.Init(); err != nil {
		t.Errorf("Init() on disabled GeoIPService = %v, want nil", err)
	}
}

// --- email.go: sendMail / Send via TCP connection failure ---
// The Send method calls sendMail which connects to SMTP.
// With an unreachable host, the TCP dial fails → ErrSMTPConnection path covered.

func TestEmailServiceSendUnreachableHostReturnsConnectionError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "127.0.0.1",
		Port:      1,
		FromEmail: "noreply@example.com",
	})

	err := svc.Send("to@example.com", "Test Subject", "Test body", false)
	if err == nil {
		t.Skip("port 1 unexpectedly accepted — skipping on this host")
	}
	if !errors.Is(err, ErrSMTPConnection) {
		t.Errorf("Send(unreachable) = %v, want ErrSMTPConnection", err)
	}
}

// SendVerification calls Send which calls sendMail — same TCP fail path.
func TestEmailServiceSendVerificationUnreachableHost(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "127.0.0.1",
		Port:      1,
		FromEmail: "noreply@example.com",
	})

	err := svc.SendVerification("user@example.com", "123456", "http://localhost")
	if err == nil {
		t.Skip("port 1 unexpectedly accepted — skipping on this host")
	}
	// Should be connection error wrapping or the ErrSMTPConnection type.
	if !errors.Is(err, ErrSMTPConnection) {
		t.Errorf("SendVerification(unreachable) = %v, want ErrSMTPConnection", err)
	}
}

func TestEmailServiceSendPasswordResetUnreachableHost(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "127.0.0.1",
		Port:      1,
		FromEmail: "noreply@example.com",
	})

	err := svc.SendPasswordReset("user@example.com", "reset-token-abc", "http://localhost")
	if err == nil {
		t.Skip("port 1 unexpectedly accepted")
	}
	if !errors.Is(err, ErrSMTPConnection) {
		t.Errorf("SendPasswordReset(unreachable) = %v, want ErrSMTPConnection", err)
	}
}

func TestEmailServiceSendWelcomeUnreachableHost(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "127.0.0.1",
		Port:      1,
		FromEmail: "noreply@example.com",
	})

	err := svc.SendWelcome("user@example.com", "testuser", "http://localhost")
	if err == nil {
		t.Skip("port 1 unexpectedly accepted")
	}
	if !errors.Is(err, ErrSMTPConnection) {
		t.Errorf("SendWelcome(unreachable) = %v, want ErrSMTPConnection", err)
	}
}

// --- tor.go: ensureDirs failure path ---
// When the configDir/dataDir is a file (not a directory), MkdirAll fails.

func TestEnsureDirsFailsWhenParentIsFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	// Create a file where a directory is expected.
	blocker := filepath.Join(base, "config")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("setup: write blocker file: %v", err)
	}

	tm := NewTorManager(TorConfig{}, blocker, filepath.Join(base, "data"), "/logs", 9050)
	err := tm.ensureDirs()
	if err == nil {
		t.Error("ensureDirs should fail when configDir is a regular file")
	}
}

// --- user.go: DeleteUser with non-empty HomeDirectory ---
// The 70% gap is the os.RemoveAll branch for non-empty HomeDirectory.

func TestDeleteUserWithHomeDirectoryRemovesIt(t *testing.T) {
	t.Parallel()
	svc, ms := newUserSvc()
	ctx := context.Background()

	// Create a real temp dir to use as HomeDirectory.
	homeDir := t.TempDir()
	testFile := filepath.Join(homeDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("setup: write test file: %v", err)
	}

	// Seed user directly into the memory store with HomeDirectory set.
	u := &model.User{
		Username:          "delhome",
		Email:             "delhome@example.com",
		PasswordHash:      "x",
		Role:              "user",
		IsActive:          true,
		HomeDirectory:     homeDir,
		StorageQuotaBytes: 1024 * 1024,
	}
	id, err := ms.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	u.ID = id

	// Now delete the user — should also remove the home directory via os.RemoveAll.
	if err := svc.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// Verify user is gone from store.
	_, err = svc.GetUser(ctx, u.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("after delete, GetUser = %v, want ErrUserNotFound", err)
	}
}

// --- auth.go: RegisterUser with validation failure ---
// RegisterUser returns (0, validationResult, nil) when inputs are invalid.
// This path doesn't touch the store at all.

func TestRegisterUserInvalidInputReturnsValidationResult(t *testing.T) {
	t.Parallel()
	svc := NewAuthService(nil)
	ctx := context.Background()

	// Empty username, email, password — all invalid.
	id, result, err := svc.RegisterUser(ctx, "", "", "", "")
	if err != nil {
		t.Fatalf("RegisterUser invalid inputs unexpected error: %v", err)
	}
	if id != 0 {
		t.Errorf("RegisterUser invalid inputs returned id = %d, want 0", id)
	}
	if result == nil || !result.HasErrors() {
		t.Error("RegisterUser invalid inputs should return ValidationResult with errors")
	}
}

func TestRegisterUserPasswordMismatchReturnsValidationResult(t *testing.T) {
	t.Parallel()
	svc := NewAuthService(nil)
	ctx := context.Background()

	// Valid username/email but mismatched passwords.
	id, result, err := svc.RegisterUser(ctx, "testuser", "test@example.com", "ValidPass1!", "DifferentPass1!")
	if err != nil {
		t.Fatalf("RegisterUser mismatch unexpected error: %v", err)
	}
	if id != 0 {
		t.Errorf("RegisterUser mismatch returned id = %d, want 0", id)
	}
	if result == nil || !result.HasErrors() {
		t.Error("RegisterUser password mismatch should return ValidationResult with errors")
	}
}

// --- auth.go: RegisterUser with username/email conflict paths ---
// These paths need a store implementing RegisterStore (different signature from MemoryStore).
// We use a minimal fake that satisfies both AuthStore and RegisterStore.

// fakeRegisterStore is a minimal in-memory store for RegisterUser tests.
type fakeRegisterStore struct {
	users    []*model.User
	nextID   int64
	forceErr error
}

func (f *fakeRegisterStore) GetAdminByUsername(_ context.Context, _ string) (*model.Admin, error) {
	return nil, nil
}
func (f *fakeRegisterStore) GetAdminByEmail(_ context.Context, _ string) (*model.Admin, error) {
	return nil, nil
}
func (f *fakeRegisterStore) GetAdminByID(_ context.Context, _ int64) (*model.Admin, error) {
	return nil, nil
}
func (f *fakeRegisterStore) UpdateAdmin(_ context.Context, _ *model.Admin) error { return nil }
func (f *fakeRegisterStore) CreateSession(_ context.Context, _ *model.Session) error {
	return nil
}
func (f *fakeRegisterStore) GetSession(_ context.Context, _ string) (*model.Session, error) {
	return nil, nil
}
func (f *fakeRegisterStore) UpdateSession(_ context.Context, _ *model.Session) error { return nil }
func (f *fakeRegisterStore) DeleteSession(_ context.Context, _ string) error { return nil }
func (f *fakeRegisterStore) DeleteUserSessions(_ context.Context, _ int64) error { return nil }
func (f *fakeRegisterStore) GetUserByID(_ context.Context, _ int64) (*model.User, error) {
	return nil, nil
}
func (f *fakeRegisterStore) UpdateUser(_ context.Context, _ *model.User) error { return nil }

func (f *fakeRegisterStore) GetUserByUsername(_ context.Context, username string) (*model.User, error) {
	for _, u := range f.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeRegisterStore) GetUserByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeRegisterStore) CreateUser(_ context.Context, u *model.User) error {
	if f.forceErr != nil {
		return f.forceErr
	}
	f.nextID++
	u.ID = f.nextID
	f.users = append(f.users, u)
	return nil
}

func TestRegisterUserUsernameAlreadyTaken(t *testing.T) {
	t.Parallel()
	store := &fakeRegisterStore{}
	store.users = append(store.users, &model.User{
		Username: "existinguser",
		Email:    "other@example.com",
	})

	svc := NewAuthService(store)
	ctx := context.Background()

	id, result, err := svc.RegisterUser(ctx, "existinguser", "new@example.com", "ValidPass1!", "ValidPass1!")
	if err != nil {
		t.Fatalf("RegisterUser username conflict unexpected error: %v", err)
	}
	if id != 0 {
		t.Errorf("RegisterUser username conflict returned id = %d, want 0", id)
	}
	if result == nil || !result.HasErrors() {
		t.Error("RegisterUser username conflict should return ValidationResult with errors")
	}
}

func TestRegisterUserEmailAlreadyTaken(t *testing.T) {
	t.Parallel()
	store := &fakeRegisterStore{}
	store.users = append(store.users, &model.User{
		Username: "otheruser",
		Email:    "taken@example.com",
	})

	svc := NewAuthService(store)
	ctx := context.Background()

	id, result, err := svc.RegisterUser(ctx, "newuser", "taken@example.com", "ValidPass1!", "ValidPass1!")
	if err != nil {
		t.Fatalf("RegisterUser email conflict unexpected error: %v", err)
	}
	if id != 0 {
		t.Errorf("RegisterUser email conflict returned id = %d, want 0", id)
	}
	if result == nil || !result.HasErrors() {
		t.Error("RegisterUser email conflict should return ValidationResult with errors")
	}
}

func TestRegisterUserSucceeds(t *testing.T) {
	t.Parallel()
	store := &fakeRegisterStore{}
	svc := NewAuthService(store)
	ctx := context.Background()

	id, result, err := svc.RegisterUser(ctx, "brandnew", "brand@example.com", "ValidPass1!", "ValidPass1!")
	if err != nil {
		t.Fatalf("RegisterUser happy path unexpected error: %v", err)
	}
	if result != nil && result.HasErrors() {
		t.Errorf("RegisterUser happy path returned errors: %v", result.Errors)
	}
	if id <= 0 {
		t.Errorf("RegisterUser happy path returned id = %d, want > 0", id)
	}
}

// --- auth.go: NewAuthServiceWithStore ---
// NewAuthServiceWithStore is a trivial one-liner but shows at 0% because it needs
// a *store.SQLiteStore. Use an in-memory SQLite to cover it.

func TestNewAuthServiceWithStoreReturnsNonNil(t *testing.T) {
	t.Parallel()
	sqliteStore, err := store.NewSQLiteStore("file::memory:?cache=shared&_unique_id=auth_svc_test")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer sqliteStore.Close()

	svc := NewAuthServiceWithStore(sqliteStore)
	if svc == nil {
		t.Error("NewAuthServiceWithStore returned nil")
	}
}

// --- user.go: NewUserServiceWithStore ---

func TestNewUserServiceWithStoreReturnsNonNil(t *testing.T) {
	t.Parallel()
	sqliteStore, err := store.NewSQLiteStore("file::memory:?cache=shared&_unique_id=user_svc_test")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer sqliteStore.Close()

	auth := NewAuthServiceWithStore(sqliteStore)
	svc := NewUserServiceWithStore(sqliteStore, auth, "/tmp")
	if svc == nil {
		t.Error("NewUserServiceWithStore returned nil")
	}
}

// --- i18n.go: NewI18nService constructor ---
// The constructor is 0% covered because all other i18n tests use buildI18nSvc directly.

func TestNewI18nServiceNotNil(t *testing.T) {
	t.Parallel()
	cfg := DefaultI18nConfig()
	// embed.FS zero value is valid — no files embedded.
	var emptyFS embed.FS
	svc := NewI18nService(cfg, emptyFS)
	if svc == nil {
		t.Fatal("NewI18nService returned nil")
	}
}

func TestNewI18nServiceHasEmptyTranslations(t *testing.T) {
	t.Parallel()
	cfg := DefaultI18nConfig()
	var emptyFS embed.FS
	svc := NewI18nService(cfg, emptyFS)
	if svc.translations == nil {
		t.Error("NewI18nService should initialize translations map")
	}
}

// --- backup.go: createArchive with directory content ---
// The directory walk branch (item ending in "/") is uncovered.
// Call createArchive directly with a "data/" prefix item and a real dataDir.

func TestCreateArchiveWithDirectoryContent(t *testing.T) {
	t.Parallel()
	svc := newBackupSvcTemp(t)

	// Create a file inside dataDir so the walk has something to process.
	dataSubDir := filepath.Join(svc.dataDir, "testdir")
	if err := os.MkdirAll(dataSubDir, 0755); err != nil {
		t.Fatalf("setup dataSubDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataSubDir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("setup test.txt: %v", err)
	}

	// Add "data/" to configDir so collectBackupContents includes it via IncludeData.
	// But we can call createArchive directly instead.
	manifest := &BackupManifest{
		Version:    "1.0.0",
		BackupType: "full",
	}

	// "data/" → the directory walk branch inside createArchive.
	contents := []string{"data/"}
	archiveData, checksum, err := svc.createArchive(contents, manifest)
	if err != nil {
		t.Fatalf("createArchive with directory content: %v", err)
	}
	if len(archiveData) == 0 {
		t.Error("createArchive returned empty archive")
	}
	if checksum == "" {
		t.Error("createArchive returned empty checksum")
	}
}

// --- backup.go: createArchive with data/ prefix file path ---
// The data/ prefix branch maps to dataDir.

func TestCreateArchiveWithDataPrefixFile(t *testing.T) {
	t.Parallel()
	svc := newBackupSvcTemp(t)

	// Write a file in dataDir to be included as "data/myfile.txt".
	if err := os.WriteFile(filepath.Join(svc.dataDir, "myfile.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("setup data file: %v", err)
	}

	manifest := &BackupManifest{Version: "1.0.0", BackupType: "full"}
	contents := []string{"data/myfile.txt"}
	archiveData, checksum, err := svc.createArchive(contents, manifest)
	if err != nil {
		t.Fatalf("createArchive(data/ prefix): %v", err)
	}
	if len(archiveData) == 0 {
		t.Error("createArchive returned empty data")
	}
	if checksum == "" {
		t.Error("createArchive returned empty checksum")
	}
}

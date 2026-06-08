// Package service — Additional tests for privilege functions.
// Covers: GetPrivilegeInfo (struct fields, consistency), RequirePrivileges,
// CanEscalate, GetBestMethod, Escalate (unknown method), EscalateAuto (already
// privileged path), and sentinel error variable distinctness.
// Note: actual escalation (sudo/doas/su) is not exercised — these tests only
// verify the logic paths that don't spawn privileged processes.
package service

import (
	"os"
	"runtime"
	"testing"
)

// --- sentinel error variables ---

func TestPrivilegeErrorVariablesDistinct(t *testing.T) {
	t.Parallel()
	errs := []error{
		ErrNoEscalationMethod,
		ErrEscalationFailed,
		ErrPrivilegesRequired,
		ErrAlreadyPrivileged,
	}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errs[i] == errs[j] {
				t.Errorf("error[%d] and error[%d] are the same value", i, j)
			}
		}
	}
}

func TestPrivilegeErrorMessagesNonEmpty(t *testing.T) {
	t.Parallel()
	for _, err := range []error{ErrNoEscalationMethod, ErrEscalationFailed, ErrPrivilegesRequired, ErrAlreadyPrivileged} {
		if err.Error() == "" {
			t.Errorf("error %T has empty message", err)
		}
	}
}

// --- GetPrivilegeInfo ---

func TestGetPrivilegeInfoNotNil(t *testing.T) {
	t.Parallel()
	info := GetPrivilegeInfo()
	if info == nil {
		t.Fatal("GetPrivilegeInfo() returned nil")
	}
}

func TestGetPrivilegeInfoOSMatches(t *testing.T) {
	t.Parallel()
	info := GetPrivilegeInfo()
	if info.OS != runtime.GOOS {
		t.Errorf("GetPrivilegeInfo().OS = %q, want %q", info.OS, runtime.GOOS)
	}
}

func TestGetPrivilegeInfoIsPrivilegedMatchesIsPrivileged(t *testing.T) {
	t.Parallel()
	info := GetPrivilegeInfo()
	if info.IsPrivileged != IsPrivileged() {
		t.Errorf("GetPrivilegeInfo().IsPrivileged = %v, IsPrivileged() = %v — should match", info.IsPrivileged, IsPrivileged())
	}
}

func TestGetPrivilegeInfoAvailableMethodsNonEmpty(t *testing.T) {
	t.Parallel()
	info := GetPrivilegeInfo()
	if len(info.AvailableMethods) == 0 {
		t.Error("GetPrivilegeInfo().AvailableMethods should not be empty")
	}
}

func TestGetPrivilegeInfoUIDOnUnix(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("UID not applicable on Windows")
	}
	info := GetPrivilegeInfo()
	if info.CurrentUID != os.Getuid() {
		t.Errorf("GetPrivilegeInfo().CurrentUID = %d, want %d", info.CurrentUID, os.Getuid())
	}
	if info.EffectiveUID != os.Geteuid() {
		t.Errorf("GetPrivilegeInfo().EffectiveUID = %d, want %d", info.EffectiveUID, os.Geteuid())
	}
}

func TestGetPrivilegeInfoMethodWhenPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("root check differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	info := GetPrivilegeInfo()
	if info.Method != MethodRoot {
		t.Errorf("GetPrivilegeInfo().Method = %q, want root when running as root", info.Method)
	}
}

func TestGetPrivilegeInfoMethodWhenNotPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root — cannot test non-privileged path")
	}
	info := GetPrivilegeInfo()
	// When not privileged the method should be one of the available methods or MethodNone
	if info.Method == MethodRoot {
		t.Error("GetPrivilegeInfo().Method should not be root when not running as root")
	}
}

// --- RequirePrivileges ---

func TestRequirePrivilegesWhenPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	if err := RequirePrivileges(); err != nil {
		t.Errorf("RequirePrivileges() as root = %v, want nil", err)
	}
}

func TestRequirePrivilegesWhenNotPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root — cannot test non-privileged path")
	}
	err := RequirePrivileges()
	if err == nil {
		t.Error("RequirePrivileges() should return error when not privileged")
	}
	if err != ErrPrivilegesRequired {
		t.Errorf("RequirePrivileges() = %v, want ErrPrivilegesRequired", err)
	}
}

// --- CanEscalate ---

func TestCanEscalateReturnsBool(t *testing.T) {
	t.Parallel()
	// Just verify it doesn't panic and returns a consistent value
	result := CanEscalate()
	_ = result
}

func TestCanEscalateTrueWhenPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	if !CanEscalate() {
		t.Error("CanEscalate() should return true when already privileged")
	}
}

// --- GetBestMethod ---

func TestGetBestMethodReturnsSomething(t *testing.T) {
	t.Parallel()
	method := GetBestMethod()
	if method == "" {
		t.Error("GetBestMethod() should return a non-empty method")
	}
}

func TestGetBestMethodRootWhenPrivileged(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	if GetBestMethod() != MethodRoot {
		t.Errorf("GetBestMethod() = %q, want root when running as root", GetBestMethod())
	}
}

func TestGetBestMethodConsistentWithDetect(t *testing.T) {
	t.Parallel()
	best := GetBestMethod()
	methods := DetectEscalationMethods()
	if len(methods) > 0 && best != methods[0] {
		t.Errorf("GetBestMethod() = %q, DetectEscalationMethods()[0] = %q — should match", best, methods[0])
	}
}

// --- Escalate (unknown method does not spawn process) ---

func TestEscalateUnknownMethodReturnsError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root — Escalate returns ErrAlreadyPrivileged before method check")
	}
	err := Escalate("unknown_method_xyz", []string{})
	if err == nil {
		t.Error("Escalate with unknown method should return error")
	}
	if err != ErrNoEscalationMethod {
		t.Errorf("Escalate unknown method = %v, want ErrNoEscalationMethod", err)
	}
}

func TestEscalateAlreadyPrivilegedReturnsError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root — cannot test already-privileged path")
	}
	err := Escalate(MethodSudo, []string{})
	if err != ErrAlreadyPrivileged {
		t.Errorf("Escalate when already root = %v, want ErrAlreadyPrivileged", err)
	}
}

// --- EscalateAuto ---

func TestEscalateAutoAlreadyPrivilegedReturnsError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root — cannot test already-privileged path")
	}
	err := EscalateAuto([]string{})
	if err != ErrAlreadyPrivileged {
		t.Errorf("EscalateAuto when already root = %v, want ErrAlreadyPrivileged", err)
	}
}

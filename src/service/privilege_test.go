// Package service — Tests for privilege detection.
// Covers: IsPrivileged, DetectEscalationMethods, EscalationMethod constants.
// Note: actual privilege state is environment-dependent; tests verify
// behavior consistency, not specific return values.
package service

import (
	"os"
	"runtime"
	"testing"
)

func TestIsPrivilegedDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = IsPrivileged()
}

func TestIsPrivilegedMatchesGeteuid(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("geteuid not applicable on Windows")
	}
	wantPrivileged := os.Geteuid() == 0
	if IsPrivileged() != wantPrivileged {
		t.Errorf("IsPrivileged() = %v, os.Geteuid()==0 = %v", IsPrivileged(), wantPrivileged)
	}
}

func TestDetectEscalationMethodsNonEmpty(t *testing.T) {
	t.Parallel()
	methods := DetectEscalationMethods()
	if len(methods) == 0 {
		t.Error("DetectEscalationMethods should always return at least one method")
	}
}

func TestDetectEscalationMethodsRootReturnsRoot(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("root check not applicable on Windows")
	}
	if os.Geteuid() != 0 {
		t.Skip("not running as root")
	}
	methods := DetectEscalationMethods()
	if len(methods) != 1 || methods[0] != MethodRoot {
		t.Errorf("root DetectEscalationMethods = %v, want [root]", methods)
	}
}

func TestDetectEscalationMethodsNonRootReturnsNoneOrMethods(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("privilege model differs on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root, skip non-root test")
	}
	methods := DetectEscalationMethods()
	if len(methods) == 0 {
		t.Error("should return at least [none] when no method available")
	}
}

func TestEscalationMethodConstants(t *testing.T) {
	t.Parallel()
	expected := map[EscalationMethod]string{
		MethodRoot:      "root",
		MethodSudo:      "sudo",
		MethodSu:        "su",
		MethodPkexec:    "pkexec",
		MethodDoas:      "doas",
		MethodOsascript: "osascript",
		MethodUAC:       "uac",
		MethodRunas:     "runas",
		MethodNone:      "none",
	}
	for method, want := range expected {
		if string(method) != want {
			t.Errorf("EscalationMethod %q = %q, want %q", method, string(method), want)
		}
	}
}

func TestPrivilegeInfoStructFields(t *testing.T) {
	t.Parallel()
	// Ensure PrivilegeInfo struct compiles correctly and has expected field types
	info := PrivilegeInfo{
		IsPrivileged:     false,
		Method:           MethodSudo,
		AvailableMethods: []EscalationMethod{MethodSudo, MethodNone},
		CanEscalate:      true,
		CurrentUID:       1000,
		EffectiveUID:     1000,
		OS:               "linux",
	}
	if info.OS != "linux" {
		t.Errorf("PrivilegeInfo.OS = %q, want linux", info.OS)
	}
	if info.Method != MethodSudo {
		t.Errorf("PrivilegeInfo.Method = %q, want sudo", info.Method)
	}
}

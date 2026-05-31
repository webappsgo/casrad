// Package service - Privilege escalation handling
// See AI.md PART 24 for privilege escalation specification
package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	ErrNoEscalationMethod   = errors.New("no privilege escalation method available")
	ErrEscalationFailed     = errors.New("privilege escalation failed")
	ErrPrivilegesRequired   = errors.New("root/administrator privileges required")
	ErrAlreadyPrivileged    = errors.New("already running with privileges")
)

// EscalationMethod represents a method of privilege escalation
type EscalationMethod string

const (
	MethodRoot     EscalationMethod = "root"     // Already root
	MethodSudo     EscalationMethod = "sudo"     // sudo command
	MethodSu       EscalationMethod = "su"       // su command
	MethodPkexec   EscalationMethod = "pkexec"   // PolicyKit
	MethodDoas     EscalationMethod = "doas"     // OpenBSD doas
	// macOS GUI prompt
	MethodOsascript EscalationMethod = "osascript"
	MethodUAC      EscalationMethod = "uac"      // Windows UAC
	MethodRunas    EscalationMethod = "runas"    // Windows runas
	MethodNone     EscalationMethod = "none"     // No method available
)

// PrivilegeInfo contains information about current privilege state
type PrivilegeInfo struct {
	IsPrivileged      bool             `json:"is_privileged"`
	Method            EscalationMethod `json:"method"`
	AvailableMethods  []EscalationMethod `json:"available_methods"`
	CanEscalate       bool             `json:"can_escalate"`
	CurrentUID        int              `json:"current_uid"`
	EffectiveUID      int              `json:"effective_uid"`
	OS                string           `json:"os"`
}

// IsPrivileged returns true if running as root/administrator
func IsPrivileged() bool {
	switch runtime.GOOS {
	case "windows":
		return isWindowsAdmin()
	default:
		return os.Geteuid() == 0
	}
}

// isWindowsAdmin checks if running as administrator on Windows
func isWindowsAdmin() bool {
	// Try to run a command that requires admin privileges
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

// DetectEscalationMethods detects available privilege escalation methods per PART 24
func DetectEscalationMethods() []EscalationMethod {
	var methods []EscalationMethod

	// If already privileged, return just that
	if IsPrivileged() {
		return []EscalationMethod{MethodRoot}
	}

	switch runtime.GOOS {
	case "linux":
		methods = detectLinuxMethods()
	case "darwin":
		methods = detectMacOSMethods()
	case "freebsd", "openbsd", "netbsd":
		methods = detectBSDMethods()
	case "windows":
		methods = detectWindowsMethods()
	}

	if len(methods) == 0 {
		methods = []EscalationMethod{MethodNone}
	}

	return methods
}

// detectLinuxMethods detects escalation methods on Linux per PART 24
// Order: sudo, su, pkexec, doas
func detectLinuxMethods() []EscalationMethod {
	var methods []EscalationMethod

	// Check sudo
	if hasSudo() {
		methods = append(methods, MethodSudo)
	}

	// Check su
	if hasSu() {
		methods = append(methods, MethodSu)
	}

	// Check pkexec (PolicyKit)
	if hasPkexec() {
		methods = append(methods, MethodPkexec)
	}

	// Check doas
	if hasDoas() {
		methods = append(methods, MethodDoas)
	}

	return methods
}

// detectMacOSMethods detects escalation methods on macOS per PART 24
// Order: sudo, osascript
func detectMacOSMethods() []EscalationMethod {
	var methods []EscalationMethod

	// Check sudo (user must be in admin group)
	if hasSudo() {
		methods = append(methods, MethodSudo)
	}

	// osascript with administrator privileges (GUI prompt)
	// Always available on macOS with GUI
	if hasGUI() {
		methods = append(methods, MethodOsascript)
	}

	return methods
}

// detectBSDMethods detects escalation methods on BSD per PART 24
// Order: doas, sudo, su
func detectBSDMethods() []EscalationMethod {
	var methods []EscalationMethod

	// Check doas (OpenBSD default)
	if hasDoas() {
		methods = append(methods, MethodDoas)
	}

	// Check sudo
	if hasSudo() {
		methods = append(methods, MethodSudo)
	}

	// Check su
	if hasSu() {
		methods = append(methods, MethodSu)
	}

	return methods
}

// detectWindowsMethods detects escalation methods on Windows per PART 24
// Order: UAC, runas
func detectWindowsMethods() []EscalationMethod {
	var methods []EscalationMethod

	// UAC prompt (requires GUI)
	if hasGUI() {
		methods = append(methods, MethodUAC)
	}

	// runas (command line)
	methods = append(methods, MethodRunas)

	return methods
}

// hasSudo checks if sudo is available
func hasSudo() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

// hasSu checks if su is available
func hasSu() bool {
	_, err := exec.LookPath("su")
	return err == nil
}

// hasPkexec checks if pkexec is available
func hasPkexec() bool {
	_, err := exec.LookPath("pkexec")
	return err == nil
}

// hasDoas checks if doas is available and configured
func hasDoas() bool {
	path, err := exec.LookPath("doas")
	if err != nil {
		return false
	}
	// Check if doas.conf exists
	if _, err := os.Stat("/etc/doas.conf"); err == nil {
		return true
	}
	// OpenBSD has doas built-in
	if runtime.GOOS == "openbsd" {
		return path != ""
	}
	return false
}

// hasGUI checks if a GUI environment is available
func hasGUI() bool {
	switch runtime.GOOS {
	case "darwin":
		// macOS always has GUI unless in SSH session without X forwarding
		return os.Getenv("SSH_CLIENT") == "" || os.Getenv("DISPLAY") != ""
	case "linux":
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	case "windows":
		// Windows typically has GUI
		return true
	default:
		return os.Getenv("DISPLAY") != ""
	}
}

// GetPrivilegeInfo returns detailed privilege information
func GetPrivilegeInfo() *PrivilegeInfo {
	methods := DetectEscalationMethods()

	info := &PrivilegeInfo{
		IsPrivileged:     IsPrivileged(),
		AvailableMethods: methods,
		CanEscalate:      len(methods) > 0 && methods[0] != MethodNone,
		OS:               runtime.GOOS,
	}

	if IsPrivileged() {
		info.Method = MethodRoot
	} else if len(methods) > 0 {
		info.Method = methods[0]
	} else {
		info.Method = MethodNone
	}

	// Get UID info (Unix only)
	if runtime.GOOS != "windows" {
		info.CurrentUID = os.Getuid()
		info.EffectiveUID = os.Geteuid()
	}

	return info
}

// Escalate attempts to re-execute the current binary with elevated privileges
func Escalate(method EscalationMethod, args []string) error {
	if IsPrivileged() {
		return ErrAlreadyPrivileged
	}

	binary, err := os.Executable()
	if err != nil {
		return err
	}

	switch method {
	case MethodSudo:
		return escalateWithSudo(binary, args)
	case MethodDoas:
		return escalateWithDoas(binary, args)
	case MethodSu:
		return escalateWithSu(binary, args)
	case MethodPkexec:
		return escalateWithPkexec(binary, args)
	case MethodOsascript:
		return escalateWithOsascript(binary, args)
	case MethodUAC:
		return escalateWithUAC(binary, args)
	case MethodRunas:
		return escalateWithRunas(binary, args)
	default:
		return ErrNoEscalationMethod
	}
}

// EscalateAuto automatically escalates using the best available method
func EscalateAuto(args []string) error {
	if IsPrivileged() {
		return ErrAlreadyPrivileged
	}

	methods := DetectEscalationMethods()
	if len(methods) == 0 || methods[0] == MethodNone {
		return ErrNoEscalationMethod
	}

	return Escalate(methods[0], args)
}

// escalateWithSudo uses sudo to escalate privileges
func escalateWithSudo(binary string, args []string) error {
	cmdArgs := append([]string{binary}, args...)
	cmd := exec.Command("sudo", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithDoas uses doas to escalate privileges
func escalateWithDoas(binary string, args []string) error {
	cmdArgs := append([]string{binary}, args...)
	cmd := exec.Command("doas", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithSu uses su to escalate privileges
func escalateWithSu(binary string, args []string) error {
	command := binary + " " + strings.Join(args, " ")
	cmd := exec.Command("su", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithPkexec uses pkexec (PolicyKit) to escalate privileges
func escalateWithPkexec(binary string, args []string) error {
	cmdArgs := append([]string{binary}, args...)
	cmd := exec.Command("pkexec", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithOsascript uses macOS osascript for GUI prompt
func escalateWithOsascript(binary string, args []string) error {
	script := fmt.Sprintf(`do shell script "%s %s" with administrator privileges`,
		binary, strings.Join(args, " "))
	cmd := exec.Command("osascript", "-e", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithUAC uses Windows UAC for elevation
func escalateWithUAC(binary string, args []string) error {
	// Use PowerShell's Start-Process with -Verb RunAs for UAC prompt
	cmdArgs := strings.Join(args, " ")
	psScript := fmt.Sprintf(`Start-Process -FilePath "%s" -ArgumentList "%s" -Verb RunAs -Wait`,
		binary, cmdArgs)
	cmd := exec.Command("powershell", "-Command", psScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// escalateWithRunas uses Windows runas command
func escalateWithRunas(binary string, args []string) error {
	cmdArgs := append([]string{"/user:Administrator", binary}, args...)
	cmd := exec.Command("runas", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RequirePrivileges returns an error if not running with privileges
func RequirePrivileges() error {
	if !IsPrivileged() {
		return ErrPrivilegesRequired
	}
	return nil
}

// CanEscalate returns true if privilege escalation is possible
func CanEscalate() bool {
	if IsPrivileged() {
		return true
	}
	methods := DetectEscalationMethods()
	return len(methods) > 0 && methods[0] != MethodNone
}

// GetBestMethod returns the best available escalation method
func GetBestMethod() EscalationMethod {
	if IsPrivileged() {
		return MethodRoot
	}
	methods := DetectEscalationMethods()
	if len(methods) > 0 {
		return methods[0]
	}
	return MethodNone
}

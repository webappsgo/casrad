// Package service handles systemd/service management
// See AI.md PART 25 for service support specification
package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/casapps/casrad/src/paths"
)

// ServiceType represents the type of service manager
type ServiceType string

const (
	Systemd  ServiceType = "systemd"
	OpenRC   ServiceType = "openrc"
	Runit    ServiceType = "runit"
	SysV     ServiceType = "sysv"
	Launchd  ServiceType = "launchd"
	RCD      ServiceType = "rcd"     // BSD rc.d
	Windows  ServiceType = "windows"
	Unknown  ServiceType = "unknown"
)

// Config holds service configuration
type Config struct {
	// Service name (default: casrad)
	Name string
	// Display name
	DisplayName string
	// Service description
	Description string
	// Path to binary
	BinaryPath string
	// User to run as
	User string
	// Group to run as
	Group string
	// Working directory
	WorkDir string
	// Log directory
	LogDir string
	// Data directory
	DataDir string
	// Config directory
	ConfigDir string
}

// Manager handles service installation and management
type Manager struct {
	serviceType ServiceType
	config      Config
}

// NewManager creates a new service manager
func NewManager(name string) *Manager {
	if name == "" {
		name = "casrad"
	}

	dirs := paths.Get()

	// Get binary path
	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "/usr/local/bin/casrad"
	}

	return &Manager{
		serviceType: Detect(),
		config: Config{
			Name:        name,
			DisplayName: "CASRAD Audio Streaming Server",
			Description: "Complete Audio Streaming, Radio, and Distribution server",
			BinaryPath:  binaryPath,
			User:        name,
			Group:       name,
			WorkDir:     dirs.Data,
			LogDir:      dirs.Log,
			DataDir:     dirs.Data,
			ConfigDir:   dirs.Config,
		},
	}
}

// Detect detects the service manager type
func Detect() ServiceType {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "darwin":
		return Launchd
	case "freebsd", "openbsd", "netbsd":
		return RCD
	case "linux":
		// Check for systemd
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			return Systemd
		}
		// Check for runit
		if _, err := os.Stat("/etc/runit"); err == nil {
			return Runit
		}
		// Check for OpenRC
		if _, err := os.Stat("/sbin/openrc"); err == nil {
			return OpenRC
		}
		// Check for SysV
		if _, err := os.Stat("/etc/init.d"); err == nil {
			return SysV
		}
	}
	return Unknown
}

// Type returns the detected service type
func (m *Manager) Type() ServiceType {
	return m.serviceType
}

// Install installs the application as a service
func (m *Manager) Install() error {
	// Check if running as root/admin
	if os.Geteuid() != 0 {
		return errors.New("service installation requires root/administrator privileges")
	}

	// Create service user if needed
	if err := m.createServiceUser(); err != nil {
		return fmt.Errorf("failed to create service user: %w", err)
	}

	// Create required directories
	if err := m.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Install based on service type
	switch m.serviceType {
	case Systemd:
		return m.installSystemd()
	case Runit:
		return m.installRunit()
	case OpenRC:
		return m.installOpenRC()
	case Launchd:
		return m.installLaunchd()
	case RCD:
		return m.installRCD()
	case Windows:
		return m.installWindows()
	default:
		return errors.New("unsupported service type")
	}
}

// Uninstall removes the service
func (m *Manager) Uninstall() error {
	if os.Geteuid() != 0 {
		return errors.New("service uninstallation requires root/administrator privileges")
	}

	switch m.serviceType {
	case Systemd:
		return m.uninstallSystemd()
	case Runit:
		return m.uninstallRunit()
	case OpenRC:
		return m.uninstallOpenRC()
	case Launchd:
		return m.uninstallLaunchd()
	case RCD:
		return m.uninstallRCD()
	case Windows:
		return m.uninstallWindows()
	default:
		return errors.New("unsupported service type")
	}
}

// Start starts the service
func (m *Manager) Start() error {
	switch m.serviceType {
	case Systemd:
		return exec.Command("systemctl", "start", m.config.Name).Run()
	case Runit:
		return exec.Command("sv", "start", m.config.Name).Run()
	case OpenRC:
		return exec.Command("rc-service", m.config.Name, "start").Run()
	case Launchd:
		return exec.Command("launchctl", "load", "-w", m.launchdPlistPath()).Run()
	case RCD:
		return exec.Command("service", m.config.Name, "start").Run()
	case Windows:
		return exec.Command("sc", "start", m.config.Name).Run()
	default:
		return errors.New("unsupported service type")
	}
}

// Stop stops the service
func (m *Manager) Stop() error {
	switch m.serviceType {
	case Systemd:
		return exec.Command("systemctl", "stop", m.config.Name).Run()
	case Runit:
		return exec.Command("sv", "stop", m.config.Name).Run()
	case OpenRC:
		return exec.Command("rc-service", m.config.Name, "stop").Run()
	case Launchd:
		return exec.Command("launchctl", "unload", m.launchdPlistPath()).Run()
	case RCD:
		return exec.Command("service", m.config.Name, "stop").Run()
	case Windows:
		return exec.Command("sc", "stop", m.config.Name).Run()
	default:
		return errors.New("unsupported service type")
	}
}

// Restart restarts the service
func (m *Manager) Restart() error {
	switch m.serviceType {
	case Systemd:
		return exec.Command("systemctl", "restart", m.config.Name).Run()
	case Runit:
		return exec.Command("sv", "restart", m.config.Name).Run()
	case OpenRC:
		return exec.Command("rc-service", m.config.Name, "restart").Run()
	case Launchd:
		m.Stop()
		return m.Start()
	case RCD:
		return exec.Command("service", m.config.Name, "restart").Run()
	case Windows:
		exec.Command("sc", "stop", m.config.Name).Run()
		return exec.Command("sc", "start", m.config.Name).Run()
	default:
		return errors.New("unsupported service type")
	}
}

// Enable enables the service to start on boot
func (m *Manager) Enable() error {
	switch m.serviceType {
	case Systemd:
		return exec.Command("systemctl", "enable", m.config.Name).Run()
	case OpenRC:
		return exec.Command("rc-update", "add", m.config.Name, "default").Run()
	case RCD:
		return m.enableRCD()
	case Launchd:
		// RunAtLoad in plist handles this
		return nil
	case Windows:
		return exec.Command("sc", "config", m.config.Name, "start=", "auto").Run()
	default:
		return nil
	}
}

// Disable disables the service from starting on boot
func (m *Manager) Disable() error {
	switch m.serviceType {
	case Systemd:
		return exec.Command("systemctl", "disable", m.config.Name).Run()
	case OpenRC:
		return exec.Command("rc-update", "del", m.config.Name).Run()
	case RCD:
		return m.disableRCD()
	case Launchd:
		return exec.Command("launchctl", "unload", "-w", m.launchdPlistPath()).Run()
	case Windows:
		return exec.Command("sc", "config", m.config.Name, "start=", "disabled").Run()
	default:
		return nil
	}
}

// Status returns the service status
func (m *Manager) Status() (string, error) {
	switch m.serviceType {
	case Systemd:
		out, err := exec.Command("systemctl", "status", m.config.Name).Output()
		return string(out), err
	case Runit:
		out, err := exec.Command("sv", "status", m.config.Name).Output()
		return string(out), err
	case OpenRC:
		out, err := exec.Command("rc-service", m.config.Name, "status").Output()
		return string(out), err
	case Launchd:
		out, err := exec.Command("launchctl", "list", m.config.Name).Output()
		return string(out), err
	case RCD:
		out, err := exec.Command("service", m.config.Name, "status").Output()
		return string(out), err
	case Windows:
		out, err := exec.Command("sc", "query", m.config.Name).Output()
		return string(out), err
	default:
		return "", errors.New("unsupported service type")
	}
}

// IsInstalled checks if the service is installed
func (m *Manager) IsInstalled() bool {
	switch m.serviceType {
	case Systemd:
		_, err := os.Stat(m.systemdUnitPath())
		return err == nil
	case Runit:
		_, err := os.Stat(filepath.Join("/etc/sv", m.config.Name, "run"))
		return err == nil
	case OpenRC:
		_, err := os.Stat(filepath.Join("/etc/init.d", m.config.Name))
		return err == nil
	case Launchd:
		_, err := os.Stat(m.launchdPlistPath())
		return err == nil
	case RCD:
		_, err := os.Stat(filepath.Join("/usr/local/etc/rc.d", m.config.Name))
		return err == nil
	case Windows:
		out, _ := exec.Command("sc", "query", m.config.Name).Output()
		return strings.Contains(string(out), m.config.Name)
	default:
		return false
	}
}

// IsRunning checks if the service is running
func (m *Manager) IsRunning() bool {
	status, err := m.Status()
	if err != nil {
		return false
	}

	switch m.serviceType {
	case Systemd:
		return strings.Contains(status, "active (running)")
	case Runit:
		return strings.Contains(status, "run:")
	case OpenRC:
		return strings.Contains(status, "started")
	case Launchd:
		return !strings.Contains(status, "unknown response")
	case RCD:
		return strings.Contains(status, "is running")
	case Windows:
		return strings.Contains(status, "RUNNING")
	default:
		return false
	}
}

// createServiceUser creates the service user if it doesn't exist
func (m *Manager) createServiceUser() error {
	// Check if user already exists
	if _, err := user.Lookup(m.config.User); err == nil {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return m.createLinuxUser()
	case "darwin":
		return m.createMacOSUser()
	case "freebsd":
		return m.createFreeBSDUser()
	case "windows":
		// Windows uses Virtual Service Account
		return nil
	default:
		return nil
	}
}

// createLinuxUser creates a system user on Linux
func (m *Manager) createLinuxUser() error {
	// Find available UID/GID in system range (100-999)
	uid := m.findAvailableUID(999, 100)
	if uid == 0 {
		return errors.New("no available system UID")
	}

	// Create group
	if err := exec.Command("groupadd", "-g", strconv.Itoa(uid), "-r", m.config.Group).Run(); err != nil {
		// Ignore if group exists
	}

	// Create user
	args := []string{
		"-r",
		"-u", strconv.Itoa(uid),
		"-g", m.config.Group,
		"-d", m.config.DataDir,
		"-s", "/usr/sbin/nologin",
		"-c", m.config.Description,
		m.config.User,
	}

	return exec.Command("useradd", args...).Run()
}

// createMacOSUser creates a system user on macOS
func (m *Manager) createMacOSUser() error {
	// Find available UID/GID in macOS system range (100-499)
	uid := m.findAvailableUID(499, 100)
	if uid == 0 {
		return errors.New("no available system UID")
	}

	uidStr := strconv.Itoa(uid)

	commands := [][]string{
		// Create group
		{"dscl", ".", "-create", "/Groups/" + m.config.Group},
		{"dscl", ".", "-create", "/Groups/" + m.config.Group, "PrimaryGroupID", uidStr},
		{"dscl", ".", "-create", "/Groups/" + m.config.Group, "Password", "*"},
		// Create user
		{"dscl", ".", "-create", "/Users/" + m.config.User},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "UniqueID", uidStr},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "PrimaryGroupID", uidStr},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "UserShell", "/usr/bin/false"},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "RealName", m.config.Description},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "NFSHomeDirectory", m.config.DataDir},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "Password", "*"},
		{"dscl", ".", "-create", "/Users/" + m.config.User, "IsHidden", "1"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			// Continue on error, some may already exist
		}
	}

	return nil
}

// createFreeBSDUser creates a system user on FreeBSD
func (m *Manager) createFreeBSDUser() error {
	uid := m.findAvailableUID(999, 100)
	if uid == 0 {
		return errors.New("no available system UID")
	}

	uidStr := strconv.Itoa(uid)

	// Create group
	exec.Command("pw", "groupadd", "-n", m.config.Group, "-g", uidStr).Run()

	// Create user
	return exec.Command("pw", "useradd",
		"-n", m.config.User,
		"-u", uidStr,
		"-g", uidStr,
		"-d", m.config.DataDir,
		"-s", "/usr/sbin/nologin",
		"-c", m.config.Description,
	).Run()
}

// findAvailableUID finds an available UID/GID
func (m *Manager) findAvailableUID(max, min int) int {
	for id := max; id >= min; id-- {
		if _, err := user.LookupId(strconv.Itoa(id)); err == nil {
			continue
		}
		if _, err := user.LookupGroupId(strconv.Itoa(id)); err == nil {
			continue
		}
		return id
	}
	return 0
}

// createDirectories creates required directories
func (m *Manager) createDirectories() error {
	dirs := []string{m.config.DataDir, m.config.LogDir, m.config.ConfigDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Set ownership on non-Windows
		if runtime.GOOS != "windows" {
			if u, err := user.Lookup(m.config.User); err == nil {
				uid, _ := strconv.Atoi(u.Uid)
				gid, _ := strconv.Atoi(u.Gid)
				os.Chown(dir, uid, gid)
			}
		}
	}

	return nil
}

// systemd implementation

const systemdTemplate = `[Unit]
Description={{.Description}}
Documentation=https://github.com/casapps/casrad
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{.User}}
Group={{.Group}}
ExecStart={{.BinaryPath}}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadWritePaths={{.DataDir}}
ReadWritePaths={{.LogDir}}

[Install]
WantedBy=multi-user.target
`

func (m *Manager) systemdUnitPath() string {
	return filepath.Join("/etc/systemd/system", m.config.Name+".service")
}

func (m *Manager) installSystemd() error {
	tmpl, err := template.New("systemd").Parse(systemdTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(m.systemdUnitPath())
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, m.config); err != nil {
		return err
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}

	// Enable service
	return exec.Command("systemctl", "enable", m.config.Name).Run()
}

func (m *Manager) uninstallSystemd() error {
	// Stop and disable
	exec.Command("systemctl", "stop", m.config.Name).Run()
	exec.Command("systemctl", "disable", m.config.Name).Run()

	// Remove unit file
	os.Remove(m.systemdUnitPath())

	// Reload systemd
	return exec.Command("systemctl", "daemon-reload").Run()
}

// runit implementation

const runitRunTemplate = `#!/bin/sh
exec chpst -u {{.User}}:{{.Group}} {{.BinaryPath}} 2>&1
`

const runitLogTemplate = `#!/bin/sh
exec svlogd -tt {{.LogDir}}
`

func (m *Manager) installRunit() error {
	svDir := filepath.Join("/etc/sv", m.config.Name)
	logDir := filepath.Join(svDir, "log")

	// Create directories
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Create run script
	runTmpl, _ := template.New("run").Parse(runitRunTemplate)
	runFile, err := os.OpenFile(filepath.Join(svDir, "run"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	runTmpl.Execute(runFile, m.config)
	runFile.Close()

	// Create log/run script
	logTmpl, _ := template.New("log").Parse(runitLogTemplate)
	logFile, err := os.OpenFile(filepath.Join(logDir, "run"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	logTmpl.Execute(logFile, m.config)
	logFile.Close()

	// Create symlink to enable service
	return os.Symlink(svDir, filepath.Join("/var/service", m.config.Name))
}

func (m *Manager) uninstallRunit() error {
	// Stop service
	exec.Command("sv", "stop", m.config.Name).Run()

	// Remove symlink
	os.Remove(filepath.Join("/var/service", m.config.Name))

	// Remove service directory
	return os.RemoveAll(filepath.Join("/etc/sv", m.config.Name))
}

// OpenRC implementation

const openrcTemplate = `#!/sbin/openrc-run

name="{{.Name}}"
description="{{.Description}}"
command="{{.BinaryPath}}"
command_user="{{.User}}:{{.Group}}"
pidfile="/run/${RC_SVCNAME}.pid"
command_background=true

depend() {
    need net
    after firewall
}
`

func (m *Manager) installOpenRC() error {
	tmpl, err := template.New("openrc").Parse(openrcTemplate)
	if err != nil {
		return err
	}

	initPath := filepath.Join("/etc/init.d", m.config.Name)
	f, err := os.OpenFile(initPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, m.config); err != nil {
		return err
	}

	// Add to default runlevel
	return exec.Command("rc-update", "add", m.config.Name, "default").Run()
}

func (m *Manager) uninstallOpenRC() error {
	// Stop and remove from runlevel
	exec.Command("rc-service", m.config.Name, "stop").Run()
	exec.Command("rc-update", "del", m.config.Name).Run()

	// Remove init script
	return os.Remove(filepath.Join("/etc/init.d", m.config.Name))
}

// launchd implementation

const launchdTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.casapps.{{.Name}}</string>

    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
    </array>

    <key>UserName</key>
    <string>{{.User}}</string>

    <key>GroupName</key>
    <string>{{.Group}}</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>WorkingDirectory</key>
    <string>{{.WorkDir}}</string>

    <key>StandardOutPath</key>
    <string>{{.LogDir}}/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/stderr.log</string>
</dict>
</plist>
`

func (m *Manager) launchdPlistPath() string {
	return filepath.Join("/Library/LaunchDaemons", "com.casapps."+m.config.Name+".plist")
}

func (m *Manager) installLaunchd() error {
	tmpl, err := template.New("launchd").Parse(launchdTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(m.launchdPlistPath())
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, m.config); err != nil {
		return err
	}

	// Set proper permissions
	os.Chmod(m.launchdPlistPath(), 0644)

	// Load the service
	return exec.Command("launchctl", "load", "-w", m.launchdPlistPath()).Run()
}

func (m *Manager) uninstallLaunchd() error {
	// Unload service
	exec.Command("launchctl", "unload", "-w", m.launchdPlistPath()).Run()

	// Remove plist
	return os.Remove(m.launchdPlistPath())
}

// BSD rc.d implementation

const rcdTemplate = `#!/bin/sh

# PROVIDE: {{.Name}}
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="{{.Name}}"
rcvar="{{.Name}}_enable"
command="{{.BinaryPath}}"
{{.Name}}_user="{{.User}}"

load_rc_config $name
run_rc_command "$1"
`

func (m *Manager) installRCD() error {
	tmpl, err := template.New("rcd").Parse(rcdTemplate)
	if err != nil {
		return err
	}

	rcPath := filepath.Join("/usr/local/etc/rc.d", m.config.Name)
	f, err := os.OpenFile(rcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, m.config); err != nil {
		return err
	}

	// Enable in rc.conf
	return m.enableRCD()
}

func (m *Manager) uninstallRCD() error {
	// Stop service
	exec.Command("service", m.config.Name, "stop").Run()

	// Disable
	m.disableRCD()

	// Remove rc script
	return os.Remove(filepath.Join("/usr/local/etc/rc.d", m.config.Name))
}

func (m *Manager) enableRCD() error {
	// Add to /etc/rc.conf
	entry := fmt.Sprintf("%s_enable=\"YES\"\n", m.config.Name)

	f, err := os.OpenFile("/etc/rc.conf", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

func (m *Manager) disableRCD() error {
	// Read rc.conf
	data, err := os.ReadFile("/etc/rc.conf")
	if err != nil {
		return err
	}

	// Remove the enable line
	lines := strings.Split(string(data), "\n")
	var newLines []string
	prefix := m.config.Name + "_enable"

	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), prefix) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile("/etc/rc.conf", []byte(strings.Join(newLines, "\n")), 0644)
}

// Windows implementation

func (m *Manager) installWindows() error {
	// Create service using sc command
	// Windows Virtual Service Account is used automatically when ServiceStartName is empty
	args := []string{
		"create", m.config.Name,
		"binPath=", m.config.BinaryPath,
		"DisplayName=", m.config.DisplayName,
		"start=", "auto",
	}

	if err := exec.Command("sc", args...).Run(); err != nil {
		return err
	}

	// Set description
	return exec.Command("sc", "description", m.config.Name, m.config.Description).Run()
}

func (m *Manager) uninstallWindows() error {
	// Stop service first
	exec.Command("sc", "stop", m.config.Name).Run()

	// Delete service
	return exec.Command("sc", "delete", m.config.Name).Run()
}

// Info returns information about the service configuration
func (m *Manager) Info() map[string]interface{} {
	return map[string]interface{}{
		"name":         m.config.Name,
		"display_name": m.config.DisplayName,
		"description":  m.config.Description,
		"binary_path":  m.config.BinaryPath,
		"user":         m.config.User,
		"group":        m.config.Group,
		"work_dir":     m.config.WorkDir,
		"log_dir":      m.config.LogDir,
		"data_dir":     m.config.DataDir,
		"config_dir":   m.config.ConfigDir,
		"type":         string(m.serviceType),
		"installed":    m.IsInstalled(),
		"running":      m.IsRunning(),
	}
}

package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OSHandler provides platform-specific functionality
type OSHandler interface {
	GetDefaultDataPath() string
	GetDefaultPort() int
	GetDatabasePath(dataPath string) string
	CreateDirectories(dataPath string) error
	HasPrivileges() bool
	IsRunningAsService() bool
	InstallService() error
	GetServiceName() string
	GetSystemUserUID() int
	GetSystemUserGID() int
}

// detectOS detects the operating system and returns appropriate handler
func (c *CASRAD) detectOS() OSHandler {
	switch runtime.GOOS {
	case "linux":
		return detectLinuxDistro()
	case "darwin":
		return &MacOSHandler{}
	case "windows":
		return &WindowsHandler{}
	case "freebsd", "openbsd", "netbsd":
		return &BSDHandler{variant: runtime.GOOS}
	default:
		return &GenericUnixHandler{}
	}
}

// detectLinuxDistro detects Linux distribution and init system
func detectLinuxDistro() OSHandler {
	// Check if running in container
	if isRunningInContainer() {
		return &ContainerHandler{}
	}

	// Detect init system
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &SystemdLinuxHandler{}
	}
	if _, err := os.Stat("/sbin/openrc"); err == nil {
		return &OpenRCLinuxHandler{}
	}
	if _, err := os.Stat("/etc/init.d"); err == nil {
		return &SysVLinuxHandler{}
	}

	return &GenericLinuxHandler{}
}

// isRunningInContainer detects if running inside a container
func isRunningInContainer() bool {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// Check for Podman
	if os.Getenv("container") == "podman" {
		return true
	}

	// Check PID 1 process
	if data, err := os.ReadFile("/proc/1/comm"); err == nil {
		pid1 := strings.TrimSpace(string(data))
		containerInits := []string{"tini", "docker-init", "containerd", "sh", "bash"}
		for _, init := range containerInits {
			if pid1 == init {
				return true
			}
		}
	}

	// Check cgroup for container signatures
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		cgroupData := string(data)
		if strings.Contains(cgroupData, "docker") ||
		   strings.Contains(cgroupData, "kubepods") ||
		   strings.Contains(cgroupData, "containerd") ||
		   strings.Contains(cgroupData, "lxc") {
			return true
		}
	}

	return false
}

// BaseOSHandler provides common functionality
type BaseOSHandler struct{}

func (b *BaseOSHandler) GetDefaultPort() int {
	if b.HasPrivileges() {
		return 80
	}
	// Auto-select from range 64000-64999 as per spec
	for port := 64000; port <= 64999; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	return 8080
}

func (b *BaseOSHandler) HasPrivileges() bool {
	if runtime.GOOS != "windows" {
		return os.Geteuid() == 0
	}
	return false
}

func (b *BaseOSHandler) IsRunningAsService() bool {
	return os.Getenv("INVOCATION_ID") != "" || // systemd
		os.Getenv("SERVICE_NAME") != "" // Windows Service
}

func (b *BaseOSHandler) GetServiceName() string {
	return "casrad"
}

func (b *BaseOSHandler) GetSystemUserUID() int {
	return 963 // As per spec
}

func (b *BaseOSHandler) GetSystemUserGID() int {
	return 963 // As per spec
}

func (b *BaseOSHandler) InstallService() error {
	return fmt.Errorf("service installation not implemented")
}

// SystemdLinuxHandler handles systemd-based Linux systems
type SystemdLinuxHandler struct {
	BaseOSHandler
}

func (s *SystemdLinuxHandler) GetDefaultDataPath() string {
	if s.HasPrivileges() {
		return "/var/lib/casrad"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "casrad")
}

func (s *SystemdLinuxHandler) GetDatabasePath(dataPath string) string {
	if s.HasPrivileges() {
		return "/etc/casrad/server.db"
	}
	return filepath.Join(dataPath, "server.db")
}

func (s *SystemdLinuxHandler) CreateDirectories(dataPath string) error {
	dirs := []string{
		dataPath,
		filepath.Join(dataPath, "users"),
	}

	if s.HasPrivileges() {
		dirs = append(dirs,
			"/etc/casrad",
			"/etc/casrad/backups",
			"/etc/casrad/backups/auto",
			"/etc/casrad/certs",
			"/etc/casrad/certs/letsencrypt",
			"/etc/casrad/security",
			"/etc/casrad/security/geoip",
			"/etc/casrad/security/blocklists",
			"/etc/casrad/security/wordlists",
			"/var/cache/casrad",
			"/var/log/casrad",
			"/tmp/casrad",
		)
	} else {
		home, _ := os.UserHomeDir()
		dirs = append(dirs,
			filepath.Join(home, ".cache", "casrad"),
			filepath.Join(home, ".config", "casrad"),
			filepath.Join(home, ".config", "casrad", "backups"),
			filepath.Join(home, ".config", "casrad", "security"),
			filepath.Join(home, ".config", "casrad", "security", "geoip"),
			filepath.Join(home, ".config", "casrad", "security", "blocklists"),
			filepath.Join(home, ".config", "casrad", "security", "wordlists"),
			filepath.Join(home, ".local", "state", "casrad"),
			filepath.Join(home, ".local", "state", "casrad", "logs"),
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}

func (s *SystemdLinuxHandler) InstallService() error {
	if !s.HasPrivileges() {
		return fmt.Errorf("requires root privileges")
	}

	// Create system user
	exec.Command("useradd", "-r", "-u", "963", "-g", "963", "-s", "/bin/false", "-d", "/var/lib/casrad", "casrad").Run()

	// Create systemd service file
	serviceContent := `[Unit]
Description=CASRAD - Complete Audio Streaming, Radio, and Distribution
After=network.target

[Service]
Type=simple
User=casrad
Group=casrad
ExecStart=/usr/local/bin/casrad
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=casrad

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/casrad /etc/casrad /var/cache/casrad /var/log/casrad

[Install]
WantedBy=multi-user.target`

	servicePath := "/etc/systemd/system/casrad.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	// Reload systemd and enable service
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "casrad").Run()
	exec.Command("systemctl", "start", "casrad").Run()

	return nil
}

// OpenRCLinuxHandler handles OpenRC-based Linux systems
type OpenRCLinuxHandler struct {
	BaseOSHandler
}

func (o *OpenRCLinuxHandler) GetDefaultDataPath() string {
	if o.HasPrivileges() {
		return "/var/lib/casrad"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "casrad")
}

func (o *OpenRCLinuxHandler) GetDatabasePath(dataPath string) string {
	if o.HasPrivileges() {
		return "/etc/casrad/server.db"
	}
	return filepath.Join(dataPath, "server.db")
}

func (o *OpenRCLinuxHandler) CreateDirectories(dataPath string) error {
	return (&SystemdLinuxHandler{}).CreateDirectories(dataPath)
}

func (o *OpenRCLinuxHandler) InstallService() error {
	if !o.HasPrivileges() {
		return fmt.Errorf("requires root privileges")
	}

	// Create OpenRC service script
	serviceContent := `#!/sbin/openrc-run

name="CASRAD"
description="Complete Audio Streaming, Radio, and Distribution"
command="/usr/local/bin/casrad"
command_user="casrad:casrad"
pidfile="/run/${RC_SVCNAME}.pid"
command_background=true

depend() {
    need net
    after firewall
}

start_pre() {
    checkpath -d -m 0755 -o casrad:casrad /var/lib/casrad
    checkpath -d -m 0755 -o casrad:casrad /etc/casrad
}`

	servicePath := "/etc/init.d/casrad"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0755); err != nil {
		return err
	}

	// Add to default runlevel
	exec.Command("rc-update", "add", "casrad", "default").Run()
	exec.Command("rc-service", "casrad", "start").Run()

	return nil
}

// SysVLinuxHandler handles SysV init systems
type SysVLinuxHandler struct {
	BaseOSHandler
}

func (s *SysVLinuxHandler) GetDefaultDataPath() string {
	if s.HasPrivileges() {
		return "/var/lib/casrad"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "casrad")
}

func (s *SysVLinuxHandler) GetDatabasePath(dataPath string) string {
	if s.HasPrivileges() {
		return "/etc/casrad/server.db"
	}
	return filepath.Join(dataPath, "server.db")
}

func (s *SysVLinuxHandler) CreateDirectories(dataPath string) error {
	return (&SystemdLinuxHandler{}).CreateDirectories(dataPath)
}

func (s *SysVLinuxHandler) InstallService() error {
	// SysV init script implementation
	return fmt.Errorf("SysV init not yet implemented")
}

// GenericLinuxHandler for unknown Linux systems
type GenericLinuxHandler struct {
	SystemdLinuxHandler
}

// ContainerHandler for containerized environments
type ContainerHandler struct {
	BaseOSHandler
}

func (c *ContainerHandler) GetDefaultDataPath() string {
	return "/var/lib/casrad"
}

func (c *ContainerHandler) GetDatabasePath(dataPath string) string {
	return filepath.Join(dataPath, "server.db")
}

func (c *ContainerHandler) CreateDirectories(dataPath string) error {
	dirs := []string{
		dataPath,
		filepath.Join(dataPath, "users"),
		"/etc/casrad",
		"/etc/casrad/backups",
		"/etc/casrad/backups/auto",
		"/etc/casrad/certs",
		"/etc/casrad/certs/letsencrypt",
		"/etc/casrad/security",
		"/etc/casrad/security/geoip",
		"/etc/casrad/security/blocklists",
		"/etc/casrad/security/wordlists",
		"/var/cache/casrad",
		"/var/log/casrad",
		"/tmp/casrad",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (c *ContainerHandler) InstallService() error {
	// No service installation in containers
	return nil
}

// MacOSHandler handles macOS systems
type MacOSHandler struct {
	BaseOSHandler
}

func (m *MacOSHandler) GetDefaultDataPath() string {
	if m.HasPrivileges() {
		return "/Library/Application Support/casrad"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "casrad")
}

func (m *MacOSHandler) GetDatabasePath(dataPath string) string {
	return filepath.Join(dataPath, "server.db")
}

func (m *MacOSHandler) CreateDirectories(dataPath string) error {
	dirs := []string{
		dataPath,
		filepath.Join(dataPath, "users"),
		filepath.Join(dataPath, "backups"),
		filepath.Join(dataPath, "security"),
		filepath.Join(dataPath, "security", "geoip"),
		filepath.Join(dataPath, "security", "blocklists"),
		filepath.Join(dataPath, "security", "wordlists"),
		"/Library/Caches/casrad",
		"/Library/Logs/casrad",
		"/private/tmp/casrad",
	}

	if m.HasPrivileges() {
		dirs = append(dirs,
			"/etc/casrad",
			"/etc/casrad/certs",
			"/etc/casrad/certs/letsencrypt",
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (m *MacOSHandler) InstallService() error {
	if !m.HasPrivileges() {
		return fmt.Errorf("requires root privileges")
	}

	// Create launchd plist
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.casrad.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/casrad</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/casrad/casrad.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/casrad/casrad.error.log</string>
</dict>
</plist>`

	plistPath := "/Library/LaunchDaemons/com.casrad.server.plist"
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return err
	}

	// Load service
	exec.Command("launchctl", "load", plistPath).Run()

	return nil
}

// WindowsHandler handles Windows systems
type WindowsHandler struct {
	BaseOSHandler
}

func (w *WindowsHandler) GetDefaultDataPath() string {
	if w.HasPrivileges() {
		return filepath.Join(os.Getenv("PROGRAMDATA"), "casrad")
	}
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "casrad")
}

func (w *WindowsHandler) GetDatabasePath(dataPath string) string {
	return filepath.Join(dataPath, "server.db")
}

func (w *WindowsHandler) CreateDirectories(dataPath string) error {
	dirs := []string{
		dataPath,
		filepath.Join(dataPath, "users"),
		filepath.Join(dataPath, "backups"),
		filepath.Join(dataPath, "certs"),
		filepath.Join(dataPath, "security"),
		filepath.Join(dataPath, "security", "geoip"),
		filepath.Join(dataPath, "security", "blocklists"),
		filepath.Join(dataPath, "security", "wordlists"),
		filepath.Join(dataPath, "logs"),
	}

	// Add cache directory based on user mode
	if w.HasPrivileges() {
		dirs = append(dirs, filepath.Join(os.Getenv("LOCALAPPDATA"), "casrad", "cache"))
	} else {
		dirs = append(dirs, filepath.Join(dataPath, "cache"))
	}

	// Add temp directory
	dirs = append(dirs, filepath.Join(os.Getenv("TEMP"), "casrad"))

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (w *WindowsHandler) HasPrivileges() bool {
	// Check if running as Administrator
	// This is simplified - would need Windows API calls
	return false
}

func (w *WindowsHandler) InstallService() error {
	// Use sc.exe to create Windows service
	cmd := exec.Command("sc", "create", "CASRAD",
		"binPath=", os.Args[0],
		"DisplayName=", "CASRAD Audio Server",
		"start=", "auto")

	if err := cmd.Run(); err != nil {
		return err
	}

	// Start the service
	exec.Command("sc", "start", "CASRAD").Run()

	return nil
}

// BSDHandler handles BSD variants
type BSDHandler struct {
	BaseOSHandler
	variant string
}

func (b *BSDHandler) GetDefaultDataPath() string {
	if b.HasPrivileges() {
		return "/var/db/casrad"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".casrad")
}

func (b *BSDHandler) GetDatabasePath(dataPath string) string {
	return filepath.Join(dataPath, "server.db")
}

func (b *BSDHandler) CreateDirectories(dataPath string) error {
	dirs := []string{
		dataPath,
		filepath.Join(dataPath, "users"),
		filepath.Join(dataPath, "cache"),
		filepath.Join(dataPath, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (b *BSDHandler) InstallService() error {
	// BSD rc.d service script
	return fmt.Errorf("BSD service installation not yet implemented")
}

// GenericUnixHandler for unknown Unix systems
type GenericUnixHandler struct {
	BaseOSHandler
}

func (g *GenericUnixHandler) GetDefaultDataPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".casrad")
}

func (g *GenericUnixHandler) GetDatabasePath(dataPath string) string {
	return filepath.Join(dataPath, "server.db")
}

func (g *GenericUnixHandler) CreateDirectories(dataPath string) error {
	return os.MkdirAll(dataPath, 0755)
}

func (g *GenericUnixHandler) InstallService() error {
	return fmt.Errorf("service installation not supported on this platform")
}

// Helper function to check if port is available
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
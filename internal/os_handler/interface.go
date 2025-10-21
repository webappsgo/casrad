package os_handler

import "runtime"

// OSHandler defines the interface for OS-specific operations
type OSHandler interface {
	SetupDirectories() error
	GetDatabasePath() string
	GetPlatformInfo() string
	HasPrivileges() bool
	IsServiceInstalled() bool
	InstallService() error
}

func NewOSHandler() OSHandler {
	switch runtime.GOOS {
	case "linux":
		return &LinuxHandler{}
	case "windows":
		return &WindowsHandler{}
	case "darwin":
		return &MacOSHandler{}
	default:
		return &GenericHandler{}
	}
}

// Generic implementation for unknown platforms
type GenericHandler struct{}

func (g *GenericHandler) SetupDirectories() error {
	return nil
}

func (g *GenericHandler) GetDatabasePath() string {
	return "./casrad.db"
}

func (g *GenericHandler) GetPlatformInfo() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func (g *GenericHandler) HasPrivileges() bool {
	return false
}

func (g *GenericHandler) IsServiceInstalled() bool {
	return false
}

func (g *GenericHandler) InstallService() error {
	return nil
}

// Linux implementation
type LinuxHandler struct{}

func (l *LinuxHandler) SetupDirectories() error {
	return nil
}

func (l *LinuxHandler) GetDatabasePath() string {
	return "/tmp/casrad/casrad.db"
}

func (l *LinuxHandler) GetPlatformInfo() string {
	return "Linux " + runtime.GOARCH
}

func (l *LinuxHandler) HasPrivileges() bool {
	return false
}

func (l *LinuxHandler) IsServiceInstalled() bool {
	return false
}

func (l *LinuxHandler) InstallService() error {
	return nil
}

// Windows implementation
type WindowsHandler struct{}

func (w *WindowsHandler) SetupDirectories() error {
	return nil
}

func (w *WindowsHandler) GetDatabasePath() string {
	return "C:\\temp\\casrad\\casrad.db"
}

func (w *WindowsHandler) GetPlatformInfo() string {
	return "Windows " + runtime.GOARCH
}

func (w *WindowsHandler) HasPrivileges() bool {
	return false
}

func (w *WindowsHandler) IsServiceInstalled() bool {
	return false
}

func (w *WindowsHandler) InstallService() error {
	return nil
}

// macOS implementation
type MacOSHandler struct{}

func (m *MacOSHandler) SetupDirectories() error {
	return nil
}

func (m *MacOSHandler) GetDatabasePath() string {
	return "/tmp/casrad/casrad.db"
}

func (m *MacOSHandler) GetPlatformInfo() string {
	return "macOS " + runtime.GOARCH
}

func (m *MacOSHandler) HasPrivileges() bool {
	return false
}

func (m *MacOSHandler) IsServiceInstalled() bool {
	return false
}

func (m *MacOSHandler) InstallService() error {
	return nil
}
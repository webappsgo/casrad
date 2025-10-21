package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const (
	preferredUID = 963
	preferredGID = 963
	serviceUser  = "casrad"
	serviceGroup = "casrad"
)

type ServiceInstaller struct {
	osHandler OSHandler
}

func NewServiceInstaller(osHandler OSHandler) *ServiceInstaller {
	return &ServiceInstaller{
		osHandler: osHandler,
	}
}

func (si *ServiceInstaller) Install() error {
	// Skip if already running as service
	if si.osHandler.IsRunningAsService() {
		return nil
	}

	// Skip if in container
	if isRunningInContainer() {
		return fmt.Errorf("service installation not supported in containers")
	}

	// Only attempt if we have privileges
	if !si.osHandler.HasPrivileges() {
		return fmt.Errorf("insufficient privileges for service installation")
	}

	log.Println("Installing CASRAD as system service...")

	// Create service user
	if err := si.createServiceUser(); err != nil {
		log.Printf("Warning: Could not create service user: %v", err)
		// Continue anyway, service might run as root
	}

	// Get current binary path
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Copy binary to system location if needed
	systemBinary := si.getSystemBinaryPath()
	if binary != systemBinary {
		if err := si.copyBinary(binary, systemBinary); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
		binary = systemBinary
	}

	// Install the service
	if err := si.osHandler.InstallService(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	// Enable and start the service
	if err := si.enableService(); err != nil {
		log.Printf("Warning: Could not enable service: %v", err)
	}

	if err := si.startService(); err != nil {
		log.Printf("Warning: Could not start service: %v", err)
	}

	log.Println("Service installation completed successfully")
	return nil
}

func (si *ServiceInstaller) createServiceUser() error {
	// Check if user already exists
	if si.userExists(serviceUser) {
		return nil
	}

	// Try to create user with preferred UID/GID
	cmd := si.buildCreateUserCommand()
	if err := cmd.Run(); err != nil {
		// Try without specifying UID/GID
		cmd = si.buildCreateUserCommandSimple()
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Set up user directories
	return si.setupUserDirectories()
}

func (si *ServiceInstaller) userExists(username string) bool {
	cmd := exec.Command("id", username)
	return cmd.Run() == nil
}

func (si *ServiceInstaller) buildCreateUserCommand() *exec.Cmd {
	switch si.osHandler.(type) {
	case *SystemdLinuxHandler, *GenericLinuxHandler, *OpenRCLinuxHandler, *SysVLinuxHandler:
		// Linux useradd command
		return exec.Command("useradd",
			"-r", // System user
			"-u", strconv.Itoa(preferredUID),
			"-g", strconv.Itoa(preferredGID),
			"-d", "/var/lib/casrad",
			"-s", "/usr/sbin/nologin",
			"-c", "CASRAD Service User",
			serviceUser)
	case *BSDHandler:
		// BSD pw command
		return exec.Command("pw", "useradd",
			serviceUser,
			"-u", strconv.Itoa(preferredUID),
			"-g", strconv.Itoa(preferredGID),
			"-d", "/var/db/casrad",
			"-s", "/usr/sbin/nologin",
			"-c", "CASRAD Service User")
	default:
		// Fallback to simple command
		return si.buildCreateUserCommandSimple()
	}
}

func (si *ServiceInstaller) buildCreateUserCommandSimple() *exec.Cmd {
	switch si.osHandler.(type) {
	case *SystemdLinuxHandler, *GenericLinuxHandler, *OpenRCLinuxHandler, *SysVLinuxHandler:
		return exec.Command("useradd",
			"-r",
			"-d", "/var/lib/casrad",
			"-s", "/usr/sbin/nologin",
			serviceUser)
	case *BSDHandler:
		return exec.Command("pw", "useradd",
			serviceUser,
			"-d", "/var/db/casrad",
			"-s", "/usr/sbin/nologin")
	default:
		return exec.Command("useradd", serviceUser)
	}
}

func (si *ServiceInstaller) setupUserDirectories() error {
	dataPath := si.osHandler.GetDefaultDataPath()

	// Create directories
	if err := si.osHandler.CreateDirectories(dataPath); err != nil {
		return err
	}

	// Change ownership
	cmd := exec.Command("chown", "-R", serviceUser+":"+serviceGroup, dataPath)
	return cmd.Run()
}

func (si *ServiceInstaller) getSystemBinaryPath() string {
	switch si.osHandler.(type) {
	case *WindowsHandler:
		return filepath.Join(os.Getenv("PROGRAMFILES"), "casrad", "casrad.exe")
	default:
		return "/usr/local/bin/casrad"
	}
}

func (si *ServiceInstaller) copyBinary(src, dst string) error {
	// Create destination directory
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to destination with executable permissions
	return os.WriteFile(dst, data, 0755)
}

func (si *ServiceInstaller) enableService() error {
	serviceName := si.osHandler.GetServiceName()

	switch si.osHandler.(type) {
	case *SystemdLinuxHandler:
		cmd := exec.Command("systemctl", "enable", serviceName)
		return cmd.Run()
	case *WindowsHandler:
		// Windows services are enabled by default
		return nil
	case *MacOSHandler:
		cmd := exec.Command("launchctl", "load", "-w",
			fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
		return cmd.Run()
	default:
		// Try systemctl first, then service command
		if cmd := exec.Command("systemctl", "enable", serviceName); cmd.Run() == nil {
			return nil
		}
		// Try update-rc.d for SysV/OpenRC
		if cmd := exec.Command("update-rc.d", serviceName, "enable"); cmd.Run() == nil {
			return nil
		}
		// Try rc-update for OpenRC
		if cmd := exec.Command("rc-update", "add", serviceName, "default"); cmd.Run() == nil {
			return nil
		}
		return fmt.Errorf("could not enable service")
	}
}

func (si *ServiceInstaller) startService() error {
	serviceName := si.osHandler.GetServiceName()

	switch si.osHandler.(type) {
	case *SystemdLinuxHandler:
		cmd := exec.Command("systemctl", "start", serviceName)
		return cmd.Run()
	case *WindowsHandler:
		cmd := exec.Command("net", "start", serviceName)
		return cmd.Run()
	case *MacOSHandler:
		cmd := exec.Command("launchctl", "start", serviceName)
		return cmd.Run()
	default:
		// Try systemctl first, then service command
		if cmd := exec.Command("systemctl", "start", serviceName); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("service", serviceName, "start"); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("rc-service", serviceName, "start"); cmd.Run() == nil {
			return nil
		}
		return fmt.Errorf("could not start service")
	}
}

func (si *ServiceInstaller) Uninstall() error {
	if !si.osHandler.HasPrivileges() {
		return fmt.Errorf("insufficient privileges for service uninstallation")
	}

	serviceName := si.osHandler.GetServiceName()

	// Stop the service
	si.stopService()

	// Disable the service
	si.disableService()

	// Remove service files
	switch si.osHandler.(type) {
	case *SystemdLinuxHandler:
		os.Remove(fmt.Sprintf("/etc/systemd/system/%s.service", serviceName))
		exec.Command("systemctl", "daemon-reload").Run()
	case *WindowsHandler:
		exec.Command("sc", "delete", serviceName).Run()
	case *MacOSHandler:
		os.Remove(fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
	}

	return nil
}

func (si *ServiceInstaller) stopService() error {
	serviceName := si.osHandler.GetServiceName()

	switch si.osHandler.(type) {
	case *SystemdLinuxHandler:
		cmd := exec.Command("systemctl", "stop", serviceName)
		return cmd.Run()
	case *WindowsHandler:
		cmd := exec.Command("net", "stop", serviceName)
		return cmd.Run()
	case *MacOSHandler:
		cmd := exec.Command("launchctl", "stop", serviceName)
		return cmd.Run()
	default:
		// Try various methods
		if cmd := exec.Command("systemctl", "stop", serviceName); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("service", serviceName, "stop"); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("rc-service", serviceName, "stop"); cmd.Run() == nil {
			return nil
		}
		return fmt.Errorf("could not stop service")
	}
}

func (si *ServiceInstaller) disableService() error {
	serviceName := si.osHandler.GetServiceName()

	switch si.osHandler.(type) {
	case *SystemdLinuxHandler:
		cmd := exec.Command("systemctl", "disable", serviceName)
		return cmd.Run()
	case *MacOSHandler:
		cmd := exec.Command("launchctl", "unload",
			fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
		return cmd.Run()
	default:
		// Try various methods
		if cmd := exec.Command("systemctl", "disable", serviceName); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("update-rc.d", serviceName, "disable"); cmd.Run() == nil {
			return nil
		}
		if cmd := exec.Command("rc-update", "del", serviceName); cmd.Run() == nil {
			return nil
		}
		return fmt.Errorf("could not disable service")
	}
}
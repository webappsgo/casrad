// Package service provides server services
// See AI.md PART 32: TOR HIDDEN SERVICE
package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// TorStatus represents the current state of the Tor service
type TorStatus string

const (
	TorStatusDisabled     TorStatus = "disabled"
	TorStatusStarting     TorStatus = "starting"
	TorStatusConnected    TorStatus = "connected"
	TorStatusDisconnected TorStatus = "disconnected"
	TorStatusError        TorStatus = "error"
)

// TorConfig holds Tor configuration
type TorConfig struct {
	// Path to Tor binary (auto-detected if empty)
	Binary string `yaml:"binary" json:"binary"`
	// Data directory for Tor
	DataDir string `yaml:"data_dir" json:"data_dir"`
}

// TorInfo represents Tor status information
type TorInfo struct {
	Enabled      bool      `json:"enabled"`
	Status       TorStatus `json:"status"`
	OnionAddress string    `json:"onion_address,omitempty"`
	Binary       string    `json:"binary,omitempty"`
	DataDir      string    `json:"data_dir,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// VanityStatus represents vanity address generation status
type VanityStatus struct {
	Generating bool      `json:"generating"`
	Prefix     string    `json:"prefix,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	ElapsedSec int       `json:"elapsed_sec,omitempty"`
}

// TorManager handles all Tor lifecycle operations.
// Process integration (github.com/cretz/bine) is activated in PART 37.
type TorManager struct {
	mu           sync.Mutex
	config       TorConfig
	configDir    string
	dataDir      string
	logDir       string
	localPort    int
	ctx          context.Context
	cancel       context.CancelFunc
	onionAddress string
	status       TorStatus
	startedAt    time.Time
	errorMsg     string
	// Vanity generation
	vanityGenerating bool
	vanityPrefix     string
	vanityStarted    time.Time
	vanityCancel     context.CancelFunc
}

// NewTorManager creates a new Tor manager
func NewTorManager(config TorConfig, configDir, dataDir, logDir string, localPort int) *TorManager {
	return &TorManager{
		config:    config,
		configDir: configDir,
		dataDir:   dataDir,
		logDir:    logDir,
		localPort: localPort,
		status:    TorStatusDisabled,
	}
}

// FindTorBinary locates the Tor binary
// Returns empty string if not found (not an error - Tor is optional)
func FindTorBinary(configPath string) string {
	// Check config path first
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check PATH
	if path, err := exec.LookPath("tor"); err == nil {
		return path
	}

	// Check common locations
	var locations []string
	switch runtime.GOOS {
	case "linux":
		locations = []string{"/usr/bin/tor", "/usr/local/bin/tor"}
	case "darwin":
		locations = []string{"/usr/local/bin/tor", "/opt/homebrew/bin/tor"}
	case "windows":
		locations = []string{
			`C:\Program Files\Tor\tor.exe`,
			`C:\Program Files (x86)\Tor\tor.exe`,
		}
	case "freebsd", "openbsd", "netbsd":
		locations = []string{"/usr/local/bin/tor"}
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// Start starts the Tor hidden service
func (tm *TorManager) Start() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Find Tor binary
	binary := FindTorBinary(tm.config.Binary)
	if binary == "" {
		log.Println("INFO: Tor binary not found, hidden service disabled")
		tm.status = TorStatusDisabled
		return nil
	}

	// Ensure directories exist
	if err := tm.ensureDirs(); err != nil {
		tm.status = TorStatusError
		tm.errorMsg = err.Error()
		log.Printf("WARN: Failed to create Tor directories: %v", err)
		return err
	}

	tm.status = TorStatusStarting
	log.Println("INFO: Starting Tor hidden service...")

	// Tor process integration via github.com/cretz/bine is pending PART 37.
	// When active it will: start a dedicated Tor process with its own DataDir,
	// wait for bootstrap, create a hidden service via ADD_ONION, and store the
	// .onion address. Until then the subsystem is disabled.

	tm.ctx, tm.cancel = context.WithCancel(context.Background())
	tm.startedAt = time.Now()

	tm.status = TorStatusDisabled
	log.Println("INFO: Tor hidden service: disabled (subsystem not active)")

	return nil
}

// Stop stops the Tor hidden service
func (tm *TorManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.cancel != nil {
		tm.cancel()
		tm.cancel = nil
	}

	tm.status = TorStatusDisconnected
	tm.onionAddress = ""
	log.Println("INFO: Tor hidden service stopped")
	return nil
}

// Restart restarts the Tor hidden service
func (tm *TorManager) Restart() error {
	if err := tm.Stop(); err != nil {
		return err
	}
	return tm.Start()
}

// GetInfo returns current Tor status
func (tm *TorManager) GetInfo() TorInfo {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	return TorInfo{
		Enabled:      tm.status != TorStatusDisabled,
		Status:       tm.status,
		OnionAddress: tm.onionAddress,
		Binary:       FindTorBinary(tm.config.Binary),
		DataDir:      tm.dataDir,
		StartedAt:    tm.startedAt,
		ErrorMessage: tm.errorMsg,
	}
}

// GetOnionAddress returns the .onion address
func (tm *TorManager) GetOnionAddress() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.onionAddress
}

// IsEnabled returns whether Tor is enabled
func (tm *TorManager) IsEnabled() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.status != TorStatusDisabled
}

// SetEnabled enables or disables Tor
func (tm *TorManager) SetEnabled(enabled bool) error {
	if enabled {
		return tm.Start()
	}
	return tm.Stop()
}

// RegenerateAddress creates a new random .onion address
func (tm *TorManager) RegenerateAddress() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Delete existing keys
	keysDir := filepath.Join(tm.dataDir, "tor", "site")
	if err := os.RemoveAll(keysDir); err != nil {
		return "", fmt.Errorf("failed to remove old keys: %w", err)
	}

	tm.mu.Unlock()
	// Restart Tor to generate new keys
	if err := tm.Restart(); err != nil {
		tm.mu.Lock()
		return "", err
	}
	tm.mu.Lock()

	return tm.onionAddress, nil
}

// ApplyKeys stops Tor, replaces keys, and restarts
func (tm *TorManager) ApplyKeys(privateKey []byte) (string, error) {
	tm.mu.Lock()
	keysDir := filepath.Join(tm.dataDir, "tor", "site")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		tm.mu.Unlock()
		return "", fmt.Errorf("failed to create keys dir: %w", err)
	}

	keyPath := filepath.Join(keysDir, "hs_ed25519_secret_key")
	if err := os.WriteFile(keyPath, privateKey, 0600); err != nil {
		tm.mu.Unlock()
		return "", fmt.Errorf("failed to write key: %w", err)
	}
	tm.mu.Unlock()

	// Restart Tor with new keys
	if err := tm.Restart(); err != nil {
		return "", err
	}

	return tm.GetOnionAddress(), nil
}

// StartVanityGeneration starts background vanity address generation
func (tm *TorManager) StartVanityGeneration(prefix string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.vanityGenerating {
		return fmt.Errorf("vanity generation already in progress")
	}

	if len(prefix) > 6 {
		return fmt.Errorf("prefix too long (max 6 characters for built-in generation)")
	}

	tm.vanityGenerating = true
	tm.vanityPrefix = prefix
	tm.vanityStarted = time.Now()

	var ctx context.Context
	ctx, tm.vanityCancel = context.WithCancel(context.Background())

	go tm.generateVanity(ctx, prefix)

	return nil
}

// CancelVanityGeneration cancels ongoing vanity generation
func (tm *TorManager) CancelVanityGeneration() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.vanityCancel != nil {
		tm.vanityCancel()
		tm.vanityCancel = nil
	}
	tm.vanityGenerating = false
	tm.vanityPrefix = ""
}

// GetVanityStatus returns vanity generation status
func (tm *TorManager) GetVanityStatus() VanityStatus {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	elapsed := 0
	if tm.vanityGenerating {
		elapsed = int(time.Since(tm.vanityStarted).Seconds())
	}

	return VanityStatus{
		Generating: tm.vanityGenerating,
		Prefix:     tm.vanityPrefix,
		StartedAt:  tm.vanityStarted,
		ElapsedSec: elapsed,
	}
}

// generateVanity performs vanity address generation in background
func (tm *TorManager) generateVanity(ctx context.Context, prefix string) {
	defer func() {
		tm.mu.Lock()
		tm.vanityGenerating = false
		tm.vanityPrefix = ""
		tm.mu.Unlock()
	}()

	// Vanity address generation (ed25519 keypair derivation) is part of PART 37.
	// Currently blocks until context is cancelled or timeout is reached.

	select {
	case <-ctx.Done():
		log.Println("INFO: Vanity generation cancelled")
		return
	// Timeout
	case <-time.After(time.Hour):
		log.Println("WARN: Vanity generation timed out")
		return
	}
}

// ensureDirs creates Tor directories with correct permissions
func (tm *TorManager) ensureDirs() error {
	dirs := []string{
		filepath.Join(tm.configDir, "tor"),
		filepath.Join(tm.dataDir, "tor"),
		filepath.Join(tm.dataDir, "tor", "site"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create tor dir %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0700); err != nil {
			return fmt.Errorf("chmod tor dir %s: %w", dir, err)
		}
		// Chown (skip on Windows)
		if runtime.GOOS != "windows" {
			uid := os.Getuid()
			gid := os.Getgid()
			if err := os.Chown(dir, uid, gid); err != nil {
				return fmt.Errorf("chown tor dir %s: %w", dir, err)
			}
		}
	}

	return nil
}

// getTorConfig returns optimized torrc content for hidden-service-only mode
func getTorConfig() string {
	return `# Hidden service only - not a relay or exit
SocksPort 0
# No SOCKS proxy needed - we're server only

# Disable unused features
ExitRelay 0
ExitPolicy reject *:*
# Never act as exit node

# Don't relay traffic for others
ORPort 0
DirPort 0

# Reduce circuit building (we only need service circuits)
MaxCircuitDirtiness 600
# Keep circuits longer

# Reduce bandwidth for Tor overhead
BandwidthRate 1 MB
BandwidthBurst 2 MB

# Hidden service optimizations
HiddenServiceSingleHopMode 0
# Keep full anonymity (3 hops)

# Faster startup
FetchDirInfoEarly 1
FetchDirInfoExtraEarly 1

# Reduce memory usage
DisableDebuggerAttachment 1
`
}

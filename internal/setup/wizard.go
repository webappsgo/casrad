package setup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/casapps/casrad/internal/database"
	"github.com/casapps/casrad/internal/security"
	"github.com/casapps/casrad/internal/storage"
)

// SetupWizard handles the first-run setup process
type SetupWizard struct {
	db             *database.Engine
	storageManager *storage.Manager
	currentStep    int
	setupData      SetupData
	completed      bool
}

// SetupData contains all setup wizard configuration
type SetupData struct {
	// Step 1: Welcome
	Accepted bool `json:"accepted"`

	// Step 2: Admin Account
	AdminUsername string `json:"admin_username"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`

	// Step 3: Storage
	GlobalMusicPath     string `json:"global_music_path"`
	GlobalPodcastPath   string `json:"global_podcast_path"`
	GlobalAudiobookPath string `json:"global_audiobook_path"`
	GlobalPlaylistPath  string `json:"global_playlist_path"`
	UserBasePath        string `json:"user_base_path"`
	DefaultUserQuota    int64  `json:"default_user_quota"`

	// Step 4: Protocols
	EnableMPD      bool `json:"enable_mpd"`
	MPDPort        int  `json:"mpd_port"`
	EnableSubsonic bool `json:"enable_subsonic"`
	EnableAmpache  bool `json:"enable_ampache"`
	EnableWebDAV   bool `json:"enable_webdav"`
	EnableRTMP     bool `json:"enable_rtmp"`
	RTMPPort       int  `json:"rtmp_port"`
	EnableDLNA     bool `json:"enable_dlna"`

	// Step 5: Network
	HTTPPort     int    `json:"http_port"`
	HTTPSPort    int    `json:"https_port"`
	BindAddress  string `json:"bind_address"`
	BehindProxy  bool   `json:"behind_proxy"`
	PublicURL    string `json:"public_url"`

	// Step 6: SSL/TLS
	EnableSSL       bool   `json:"enable_ssl"`
	SSLEmail        string `json:"ssl_email"`
	SSLDomains      string `json:"ssl_domains"`
	SSLAutoRedirect bool   `json:"ssl_auto_redirect"`

	// Step 7: Complete
	StartService bool `json:"start_service"`
}

// SetupStatus represents the current setup state
type SetupStatus struct {
	Required  bool       `json:"required"`
	Completed bool       `json:"completed"`
	Step      int        `json:"step"`
	TotalSteps int       `json:"total_steps"`
	StepName  string     `json:"step_name"`
	Data      SetupData  `json:"data"`
	Errors    []string   `json:"errors,omitempty"`
}

// WizardStep represents a single step in the wizard
type WizardStep struct {
	Number      int         `json:"number"`
	Name        string      `json:"name"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Fields      []FormField `json:"fields"`
	CanSkip     bool        `json:"can_skip"`
}

// FormField represents a form input field
type FormField struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Type        string      `json:"type"` // text, password, email, number, checkbox, select, path
	Value       interface{} `json:"value"`
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
	Help        string      `json:"help,omitempty"`
	Placeholder string      `json:"placeholder,omitempty"`
	Options     []Option    `json:"options,omitempty"` // For select fields
	Validation  string      `json:"validation,omitempty"`
}

// Option represents a select field option
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

var wizardSteps = []WizardStep{
	{
		Number:      1,
		Name:        "welcome",
		Title:       "Welcome to CASRAD",
		Description: "Complete Audio Streaming, Radio, and Distribution Server",
		Fields: []FormField{
			{
				Name:     "accepted",
				Label:    "I'm ready to set up CASRAD",
				Type:     "checkbox",
				Required: true,
				Default:  false,
			},
		},
		CanSkip: false,
	},
	{
		Number:      2,
		Name:        "admin",
		Title:       "Create Admin Account",
		Description: "Set up the administrator account for managing CASRAD",
		Fields: []FormField{
			{
				Name:        "admin_username",
				Label:       "Admin Username",
				Type:        "text",
				Required:    true,
				Default:     "admin",
				Placeholder: "admin",
				Validation:  "^[a-zA-Z0-9_]{3,20}$",
			},
			{
				Name:        "admin_email",
				Label:       "Admin Email",
				Type:        "email",
				Required:    true,
				Placeholder: "admin@example.com",
				Validation:  "^[^@]+@[^@]+\\.[^@]+$",
			},
			{
				Name:        "admin_password",
				Label:       "Admin Password",
				Type:        "password",
				Required:    true,
				Help:        "Minimum 8 characters",
				Placeholder: "********",
				Validation:  "^.{8,}$",
			},
		},
		CanSkip: false,
	},
	{
		Number:      3,
		Name:        "storage",
		Title:       "Configure Storage",
		Description: "Set up storage paths for media and user data",
		Fields: []FormField{
			{
				Name:        "global_music_path",
				Label:       "Global Music Directory",
				Type:        "path",
				Default:     "/mnt/Music/Mp3",
				Help:        "Main music library path",
				Placeholder: "/mnt/Music/Mp3",
			},
			{
				Name:        "global_podcast_path",
				Label:       "Global Podcast Directory",
				Type:        "path",
				Default:     "/mnt/Podcasts",
				Help:        "Podcast storage path",
				Placeholder: "/mnt/Podcasts",
			},
			{
				Name:        "global_audiobook_path",
				Label:       "Global Audiobook Directory",
				Type:        "path",
				Default:     "/mnt/Audiobooks",
				Help:        "Audiobook storage path",
				Placeholder: "/mnt/Audiobooks",
			},
			{
				Name:        "global_playlist_path",
				Label:       "Global Playlist Directory",
				Type:        "path",
				Default:     "/mnt/Playlists",
				Help:        "Playlist storage path",
				Placeholder: "/mnt/Playlists",
			},
			{
				Name:        "user_base_path",
				Label:       "User Storage Base Path",
				Type:        "path",
				Default:     "/var/lib/casrad/users",
				Help:        "Base directory for user storage",
				Placeholder: "/var/lib/casrad/users",
			},
			{
				Name:        "default_user_quota",
				Label:       "Default User Quota (GB)",
				Type:        "number",
				Default:     50,
				Help:        "Default storage quota per user in GB",
				Placeholder: "50",
			},
		},
		CanSkip: true,
	},
	{
		Number:      4,
		Name:        "protocols",
		Title:       "Enable Protocols",
		Description: "Choose which protocols and APIs to enable",
		Fields: []FormField{
			{
				Name:    "enable_mpd",
				Label:   "Enable MPD Server",
				Type:    "checkbox",
				Default: true,
				Help:    "Music Player Daemon protocol for MPD clients",
			},
			{
				Name:        "mpd_port",
				Label:       "MPD Port",
				Type:        "number",
				Default:     6600,
				Placeholder: "6600",
			},
			{
				Name:    "enable_subsonic",
				Label:   "Enable Subsonic API",
				Type:    "checkbox",
				Default: true,
				Help:    "Subsonic API for mobile apps",
			},
			{
				Name:    "enable_ampache",
				Label:   "Enable Ampache API",
				Type:    "checkbox",
				Default: true,
				Help:    "Ampache API for web players",
			},
			{
				Name:    "enable_webdav",
				Label:   "Enable WebDAV",
				Type:    "checkbox",
				Default: true,
				Help:    "WebDAV for file access",
			},
			{
				Name:    "enable_rtmp",
				Label:   "Enable RTMP Server",
				Type:    "checkbox",
				Default: true,
				Help:    "RTMP for live streaming",
			},
			{
				Name:        "rtmp_port",
				Label:       "RTMP Port",
				Type:        "number",
				Default:     1935,
				Placeholder: "1935",
			},
			{
				Name:    "enable_dlna",
				Label:   "Enable DLNA/UPnP",
				Type:    "checkbox",
				Default: true,
				Help:    "DLNA/UPnP media server",
			},
		},
		CanSkip: true,
	},
	{
		Number:      5,
		Name:        "network",
		Title:       "Network Configuration",
		Description: "Configure network settings and ports",
		Fields: []FormField{
			{
				Name:        "http_port",
				Label:       "HTTP Port",
				Type:        "number",
				Default:     0,
				Help:        "0 = auto (64000-64999 for non-root, 80 for root)",
				Placeholder: "0",
			},
			{
				Name:        "https_port",
				Label:       "HTTPS Port",
				Type:        "number",
				Default:     0,
				Help:        "0 = auto (443 for root)",
				Placeholder: "0",
			},
			{
				Name:        "bind_address",
				Label:       "Bind Address",
				Type:        "text",
				Default:     "0.0.0.0",
				Help:        "IP address to bind to",
				Placeholder: "0.0.0.0",
			},
			{
				Name:    "behind_proxy",
				Label:   "Behind Reverse Proxy",
				Type:    "checkbox",
				Default: false,
				Help:    "Enable if running behind nginx/apache/etc",
			},
			{
				Name:        "public_url",
				Label:       "Public URL",
				Type:        "text",
				Help:        "Public URL for this server (optional)",
				Placeholder: "https://music.example.com",
			},
		},
		CanSkip: true,
	},
	{
		Number:      6,
		Name:        "ssl",
		Title:       "SSL/TLS Configuration",
		Description: "Configure HTTPS and SSL certificates",
		Fields: []FormField{
			{
				Name:    "enable_ssl",
				Label:   "Enable SSL/TLS",
				Type:    "checkbox",
				Default: false,
				Help:    "Enable HTTPS with Let's Encrypt",
			},
			{
				Name:        "ssl_email",
				Label:       "SSL Certificate Email",
				Type:        "email",
				Help:        "Email for Let's Encrypt notifications",
				Placeholder: "admin@example.com",
			},
			{
				Name:        "ssl_domains",
				Label:       "SSL Domains",
				Type:        "text",
				Help:        "Comma-separated list of domains",
				Placeholder: "music.example.com,stream.example.com",
			},
			{
				Name:    "ssl_auto_redirect",
				Label:   "Auto-redirect HTTP to HTTPS",
				Type:    "checkbox",
				Default: true,
				Help:    "Automatically redirect HTTP traffic to HTTPS",
			},
		},
		CanSkip: true,
	},
	{
		Number:      7,
		Name:        "complete",
		Title:       "Setup Complete",
		Description: "CASRAD is ready to start!",
		Fields: []FormField{
			{
				Name:    "start_service",
				Label:   "Start CASRAD service now",
				Type:    "checkbox",
				Default: true,
			},
		},
		CanSkip: false,
	},
}

func NewSetupWizard(db *database.Engine, storageManager *storage.Manager) *SetupWizard {
	w := &SetupWizard{
		db:             db,
		storageManager: storageManager,
		currentStep:    1,
		setupData:      SetupData{},
	}

	// Load existing setup state if any
	w.loadState()

	// Set defaults
	w.setDefaults()

	return w
}

func (w *SetupWizard) loadState() {
	var completed bool
	var step int
	var data string

	err := w.db.QueryRow(`
		SELECT setup_completed, wizard_step, wizard_data
		FROM setup_state WHERE id = 1
	`).Scan(&completed, &step, &data)

	if err == nil {
		w.completed = completed
		w.currentStep = step
		if data != "" {
			json.Unmarshal([]byte(data), &w.setupData)
		}
	}
}

func (w *SetupWizard) saveState() error {
	data, _ := json.Marshal(w.setupData)

	_, err := w.db.Exec(`
		INSERT OR REPLACE INTO setup_state (id, setup_completed, wizard_step, wizard_data, created_at)
		VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
	`, w.completed, w.currentStep, string(data))

	return err
}

func (w *SetupWizard) setDefaults() {
	// Set all default values
	w.setupData.GlobalMusicPath = "/mnt/Music/Mp3"
	w.setupData.GlobalPodcastPath = "/mnt/Podcasts"
	w.setupData.GlobalAudiobookPath = "/mnt/Audiobooks"
	w.setupData.GlobalPlaylistPath = "/mnt/Playlists"
	w.setupData.UserBasePath = "/var/lib/casrad/users"
	w.setupData.DefaultUserQuota = 50 * 1024 * 1024 * 1024 // 50GB

	w.setupData.EnableMPD = true
	w.setupData.MPDPort = 6600
	w.setupData.EnableSubsonic = true
	w.setupData.EnableAmpache = true
	w.setupData.EnableWebDAV = true
	w.setupData.EnableRTMP = true
	w.setupData.RTMPPort = 1935
	w.setupData.EnableDLNA = true

	w.setupData.HTTPPort = 0
	w.setupData.HTTPSPort = 0
	w.setupData.BindAddress = "0.0.0.0"
	w.setupData.BehindProxy = false

	w.setupData.StartService = true
}

// IsRequired checks if setup wizard needs to be run
func (w *SetupWizard) IsRequired() bool {
	return !w.completed
}

// GetStatus returns the current setup status
func (w *SetupWizard) GetStatus() SetupStatus {
	status := SetupStatus{
		Required:   !w.completed,
		Completed:  w.completed,
		Step:       w.currentStep,
		TotalSteps: len(wizardSteps),
		Data:       w.setupData,
	}

	if w.currentStep > 0 && w.currentStep <= len(wizardSteps) {
		status.StepName = wizardSteps[w.currentStep-1].Name
	}

	return status
}

// GetCurrentStep returns the current wizard step
func (w *SetupWizard) GetCurrentStep() WizardStep {
	if w.currentStep > 0 && w.currentStep <= len(wizardSteps) {
		return wizardSteps[w.currentStep-1]
	}
	return wizardSteps[0]
}

// GetAllSteps returns all wizard steps
func (w *SetupWizard) GetAllSteps() []WizardStep {
	return wizardSteps
}

// ValidateStep validates the current step data
func (w *SetupWizard) ValidateStep(stepNumber int, data map[string]interface{}) []string {
	errors := []string{}

	if stepNumber < 1 || stepNumber > len(wizardSteps) {
		return []string{"Invalid step number"}
	}

	step := wizardSteps[stepNumber-1]

	for _, field := range step.Fields {
		value, exists := data[field.Name]

		// Check required fields
		if field.Required && (!exists || value == nil || value == "") {
			errors = append(errors, fmt.Sprintf("%s is required", field.Label))
			continue
		}

		// Skip validation if not required and empty
		if !field.Required && (!exists || value == nil || value == "") {
			continue
		}

		// Type-specific validation
		switch field.Type {
		case "email":
			if str, ok := value.(string); ok {
				if !isValidEmail(str) {
					errors = append(errors, fmt.Sprintf("%s must be a valid email address", field.Label))
				}
			}

		case "number":
			if _, err := strconv.Atoi(fmt.Sprintf("%v", value)); err != nil {
				errors = append(errors, fmt.Sprintf("%s must be a valid number", field.Label))
			}

		case "path":
			if str, ok := value.(string); ok {
				if !filepath.IsAbs(str) {
					errors = append(errors, fmt.Sprintf("%s must be an absolute path", field.Label))
				}
			}
		}

		// Custom validation regex
		if field.Validation != "" {
			if str, ok := value.(string); ok {
				if matched, _ := regexp.MatchString(field.Validation, str); !matched {
					errors = append(errors, fmt.Sprintf("%s is invalid", field.Label))
				}
			}
		}
	}

	// Step-specific validation
	switch stepNumber {
	case 2: // Admin account
		if pass, ok := data["admin_password"].(string); ok && len(pass) < 8 {
			errors = append(errors, "Password must be at least 8 characters")
		}

	case 3: // Storage
		// Check if paths exist or can be created
		paths := []string{
			data["global_music_path"].(string),
			data["global_podcast_path"].(string),
			data["global_audiobook_path"].(string),
			data["global_playlist_path"].(string),
			data["user_base_path"].(string),
		}

		for _, path := range paths {
			if path != "" && !pathExists(path) {
				// Try to create it
				if err := os.MkdirAll(path, 0755); err != nil {
					errors = append(errors, fmt.Sprintf("Cannot create directory: %s", path))
				}
			}
		}
	}

	return errors
}

// ProcessStep processes the current step with provided data
func (w *SetupWizard) ProcessStep(stepNumber int, data map[string]interface{}) error {
	// Validate first
	if errors := w.ValidateStep(stepNumber, data); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Update setup data based on step
	switch stepNumber {
	case 1: // Welcome
		w.setupData.Accepted = true

	case 2: // Admin account
		w.setupData.AdminUsername = data["admin_username"].(string)
		w.setupData.AdminEmail = data["admin_email"].(string)
		w.setupData.AdminPassword = data["admin_password"].(string)

	case 3: // Storage
		if v, ok := data["global_music_path"].(string); ok {
			w.setupData.GlobalMusicPath = v
		}
		if v, ok := data["global_podcast_path"].(string); ok {
			w.setupData.GlobalPodcastPath = v
		}
		if v, ok := data["global_audiobook_path"].(string); ok {
			w.setupData.GlobalAudiobookPath = v
		}
		if v, ok := data["global_playlist_path"].(string); ok {
			w.setupData.GlobalPlaylistPath = v
		}
		if v, ok := data["user_base_path"].(string); ok {
			w.setupData.UserBasePath = v
		}
		if v, ok := data["default_user_quota"]; ok {
			if n, err := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64); err == nil {
				w.setupData.DefaultUserQuota = n * 1024 * 1024 * 1024 // Convert GB to bytes
			}
		}

	case 4: // Protocols
		w.setupData.EnableMPD = getBool(data, "enable_mpd", true)
		w.setupData.MPDPort = getInt(data, "mpd_port", 6600)
		w.setupData.EnableSubsonic = getBool(data, "enable_subsonic", true)
		w.setupData.EnableAmpache = getBool(data, "enable_ampache", true)
		w.setupData.EnableWebDAV = getBool(data, "enable_webdav", true)
		w.setupData.EnableRTMP = getBool(data, "enable_rtmp", true)
		w.setupData.RTMPPort = getInt(data, "rtmp_port", 1935)
		w.setupData.EnableDLNA = getBool(data, "enable_dlna", true)

	case 5: // Network
		w.setupData.HTTPPort = getInt(data, "http_port", 0)
		w.setupData.HTTPSPort = getInt(data, "https_port", 0)
		w.setupData.BindAddress = getString(data, "bind_address", "0.0.0.0")
		w.setupData.BehindProxy = getBool(data, "behind_proxy", false)
		w.setupData.PublicURL = getString(data, "public_url", "")

	case 6: // SSL
		w.setupData.EnableSSL = getBool(data, "enable_ssl", false)
		w.setupData.SSLEmail = getString(data, "ssl_email", "")
		w.setupData.SSLDomains = getString(data, "ssl_domains", "")
		w.setupData.SSLAutoRedirect = getBool(data, "ssl_auto_redirect", true)

	case 7: // Complete
		w.setupData.StartService = getBool(data, "start_service", true)
		// Apply all settings
		if err := w.applySettings(); err != nil {
			return fmt.Errorf("failed to apply settings: %w", err)
		}
		w.completed = true
	}

	// Move to next step
	if stepNumber < len(wizardSteps) {
		w.currentStep = stepNumber + 1
	}

	// Save state
	return w.saveState()
}

// SkipStep skips the current step if allowed
func (w *SetupWizard) SkipStep(stepNumber int) error {
	if stepNumber < 1 || stepNumber > len(wizardSteps) {
		return fmt.Errorf("invalid step number")
	}

	step := wizardSteps[stepNumber-1]
	if !step.CanSkip {
		return fmt.Errorf("step %d cannot be skipped", stepNumber)
	}

	// Move to next step
	if stepNumber < len(wizardSteps) {
		w.currentStep = stepNumber + 1
	}

	return w.saveState()
}

// applySettings applies all wizard settings to the database
func (w *SetupWizard) applySettings() error {
	// Create admin user
	passwordHash, err := security.HashPassword(w.setupData.AdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = w.db.Exec(`
		INSERT INTO users (username, email, password_hash, role, is_active, email_verified, created_at)
		VALUES (?, ?, ?, 'admin', 1, 1, CURRENT_TIMESTAMP)
	`, w.setupData.AdminUsername, w.setupData.AdminEmail, passwordHash)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Apply storage settings
	settings := map[string]interface{}{
		"storage.global_music_path":     w.setupData.GlobalMusicPath,
		"storage.global_podcast_path":   w.setupData.GlobalPodcastPath,
		"storage.global_audiobook_path": w.setupData.GlobalAudiobookPath,
		"storage.global_playlist_path":  w.setupData.GlobalPlaylistPath,
		"storage.user_base_path":        w.setupData.UserBasePath,
		"storage.default_user_quota":    fmt.Sprintf("%d", w.setupData.DefaultUserQuota),

		// Protocol settings
		"mpd.enabled":      fmt.Sprintf("%t", w.setupData.EnableMPD),
		"mpd.port":         fmt.Sprintf("%d", w.setupData.MPDPort),
		"subsonic.enabled": fmt.Sprintf("%t", w.setupData.EnableSubsonic),
		"ampache.enabled":  fmt.Sprintf("%t", w.setupData.EnableAmpache),
		"webdav.enabled":   fmt.Sprintf("%t", w.setupData.EnableWebDAV),
		"rtmp.enabled":     fmt.Sprintf("%t", w.setupData.EnableRTMP),
		"rtmp.port":        fmt.Sprintf("%d", w.setupData.RTMPPort),
		"dlna.enabled":     fmt.Sprintf("%t", w.setupData.EnableDLNA),

		// Network settings
		"network.port":         fmt.Sprintf("%d", w.setupData.HTTPPort),
		"network.https_port":   fmt.Sprintf("%d", w.setupData.HTTPSPort),
		"network.bind_address": w.setupData.BindAddress,
		"network.behind_proxy": fmt.Sprintf("%t", w.setupData.BehindProxy),

		// SSL settings
		"security.require_https": fmt.Sprintf("%t", w.setupData.EnableSSL),
	}

	// Update settings in database
	for key, value := range settings {
		_, err := w.db.Exec(`
			UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP
			WHERE key = ?
		`, value, key)

		if err != nil {
			return fmt.Errorf("failed to update setting %s: %w", key, err)
		}
	}

	// Update global directories
	_, err = w.db.Exec(`
		UPDATE global_directories SET path = ? WHERE type = 'music'
	`, w.setupData.GlobalMusicPath)

	_, err = w.db.Exec(`
		UPDATE global_directories SET path = ? WHERE type = 'podcast'
	`, w.setupData.GlobalPodcastPath)

	_, err = w.db.Exec(`
		UPDATE global_directories SET path = ? WHERE type = 'audiobook'
	`, w.setupData.GlobalAudiobookPath)

	_, err = w.db.Exec(`
		UPDATE global_directories SET path = ? WHERE type = 'playlist'
	`, w.setupData.GlobalPlaylistPath)

	// Mark setup as complete
	_, err = w.db.Exec(`
		UPDATE setup_state SET
			setup_completed = 1,
			completed_at = CURRENT_TIMESTAMP,
			admin_account_id = (SELECT id FROM users WHERE username = ?)
		WHERE id = 1
	`, w.setupData.AdminUsername)

	return err
}

// HTTP Handlers for the setup wizard API
func (w *SetupWizard) HandleStatus(rw http.ResponseWriter, r *http.Request) {
	status := w.GetStatus()
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(status)
}

func (w *SetupWizard) HandleGetStep(rw http.ResponseWriter, r *http.Request) {
	step := w.GetCurrentStep()
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(step)
}

func (w *SetupWizard) HandleProcessStep(rw http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(rw, "Invalid request", http.StatusBadRequest)
		return
	}

	stepNumber := w.currentStep
	if step, ok := data["step"].(float64); ok {
		stepNumber = int(step)
	}

	// Validate and process
	errors := w.ValidateStep(stepNumber, data)
	if len(errors) > 0 {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(map[string]interface{}{
			"success": false,
			"errors":  errors,
		})
		return
	}

	if err := w.ProcessStep(stepNumber, data); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated status
	status := w.GetStatus()
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
	})
}

// Helper functions
func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getBool(data map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return defaultVal
}

func getInt(data map[string]interface{}, key string, defaultVal int) int {
	if val, ok := data[key]; ok {
		if n, err := strconv.Atoi(fmt.Sprintf("%v", val)); err == nil {
			return n
		}
	}
	return defaultVal
}

func getString(data map[string]interface{}, key string, defaultVal string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return defaultVal
}
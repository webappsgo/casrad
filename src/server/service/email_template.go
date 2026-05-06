// Package service - Email template service
// See AI.md PART 18 for email template specification
package service

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	Name    string
	Subject string
	Body    string
}

// TemplateVars contains all available template variables
type TemplateVars struct {
	// Global variables
	AppName           string
	AppURL            string
	FQDN              string
	OnionURL          string
	AdminEmail        string
	RecipientEmail    string
	RecipientUsername string
	Timestamp         string
	Year              string

	// Context-specific variables
	ResetLink   string
	VerifyLink  string
	LoginURL    string
	ProfileURL  string
	AdminURL    string
	Expires     string
	IP          string
	Location    string
	Device      string
	Time        string
	Event       string
	Details     string
	Method      string
	Filename    string
	Size        string
	Error       string
	TaskName    string
	NextRun     string
	ExpiresIn   string
	ExpiryDate  string
	ValidUntil  string

	// Breach-related variables
	BreachID               string
	BreachDate             string
	BreachType             string
	BreachSummary          string
	AffectedData           string
	RecommendedActions     string
	ContactEmail           string
	ContactPhone           string
	DPOContact             string
	RegulatoryNotice       string
	NotificationDeadline   string
	Severity               string
	DetectionMethod        string
	Trigger                string
	SourceIP               string
	AffectedScope          string
	AffectedUsers          string
	AutoActions            string
	ComplianceRequirements string
	NotifyDeadline         string
}

// EmailTemplateService manages email templates
type EmailTemplateService struct {
	embeddedFS  embed.FS
	customDir   string
	templates   map[string]*EmailTemplate
	appName     string
	appURL      string
	fqdn        string
	adminEmail  string
}

// NewEmailTemplateService creates a new template service
func NewEmailTemplateService(embeddedFS embed.FS, customDir string) *EmailTemplateService {
	return &EmailTemplateService{
		embeddedFS: embeddedFS,
		customDir:  customDir,
		templates:  make(map[string]*EmailTemplate),
	}
}

// SetAppInfo sets application info for templates
func (s *EmailTemplateService) SetAppInfo(appName, appURL, fqdn, adminEmail string) {
	s.appName = appName
	s.appURL = appURL
	s.fqdn = fqdn
	s.adminEmail = adminEmail
}

// LoadTemplate loads a template, preferring custom over embedded
func (s *EmailTemplateService) LoadTemplate(name string) (*EmailTemplate, error) {
	// Check cache first
	if t, ok := s.templates[name]; ok {
		return t, nil
	}

	// Try custom template first
	customPath := filepath.Join(s.customDir, name+".txt")
	if content, err := os.ReadFile(customPath); err == nil {
		t := parseTemplate(name, string(content))
		s.templates[name] = t
		return t, nil
	}

	// Fall back to embedded template
	embeddedPath := "template/email/" + name + ".txt"
	if content, err := s.embeddedFS.ReadFile(embeddedPath); err == nil {
		t := parseTemplate(name, string(content))
		s.templates[name] = t
		return t, nil
	}

	return nil, fmt.Errorf("template not found: %s", name)
}

// RenderTemplate renders a template with the given variables
func (s *EmailTemplateService) RenderTemplate(name string, vars *TemplateVars) (subject, body string, err error) {
	t, err := s.LoadTemplate(name)
	if err != nil {
		return "", "", err
	}

	// Set global variables
	if vars.AppName == "" {
		vars.AppName = s.appName
	}
	if vars.AppURL == "" {
		vars.AppURL = s.appURL
	}
	if vars.FQDN == "" {
		vars.FQDN = s.fqdn
	}
	if vars.AdminEmail == "" {
		vars.AdminEmail = s.adminEmail
	}
	if vars.Timestamp == "" {
		vars.Timestamp = time.Now().Format(time.RFC1123)
	}
	if vars.Year == "" {
		vars.Year = time.Now().Format("2006")
	}

	subject = replaceVars(t.Subject, vars)
	body = replaceVars(t.Body, vars)

	return subject, body, nil
}

// InvalidateCache clears the template cache
func (s *EmailTemplateService) InvalidateCache() {
	s.templates = make(map[string]*EmailTemplate)
}

// SaveCustomTemplate saves a custom template
func (s *EmailTemplateService) SaveCustomTemplate(name, subject, body string) error {
	if err := os.MkdirAll(s.customDir, 0750); err != nil {
		return err
	}

	content := fmt.Sprintf("Subject: %s\n---\n%s", subject, body)
	path := filepath.Join(s.customDir, name+".txt")

	if err := os.WriteFile(path, []byte(content), 0640); err != nil {
		return err
	}

	// Invalidate cache for this template
	delete(s.templates, name)
	return nil
}

// ResetToDefault deletes custom template, restoring embedded default
func (s *EmailTemplateService) ResetToDefault(name string) error {
	path := filepath.Join(s.customDir, name+".txt")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Invalidate cache
	delete(s.templates, name)
	return nil
}

// IsCustom returns true if a custom template exists
func (s *EmailTemplateService) IsCustom(name string) bool {
	path := filepath.Join(s.customDir, name+".txt")
	_, err := os.Stat(path)
	return err == nil
}

// ListTemplates returns all available template names
func (s *EmailTemplateService) ListTemplates() []string {
	return []string{
		"welcome",
		"welcome_admin",
		"password_reset",
		"email_verify",
		"login_alert",
		"security_alert",
		"mfa_reminder",
		"2fa_enabled",
		"2fa_disabled",
		"password_changed",
		"backup_complete",
		"backup_failed",
		"ssl_expiring",
		"ssl_renewed",
		"scheduler_error",
		"breach_notification",
		"breach_admin_alert",
		"test",
	}
}

// parseTemplate parses a template file into subject and body
func parseTemplate(name, content string) *EmailTemplate {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) < 2 {
		return &EmailTemplate{Name: name, Subject: "", Body: content}
	}

	// Extract subject from first line
	subject := strings.TrimPrefix(lines[0], "Subject: ")
	subject = strings.TrimSpace(subject)

	// Rest is body (skip separator line if present)
	body := lines[1]
	if strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") {
		body = strings.SplitN(body, "\n", 2)[1]
	}
	body = strings.TrimPrefix(body, "\n")

	return &EmailTemplate{
		Name:    name,
		Subject: subject,
		Body:    body,
	}
}

// replaceVars replaces {variable} placeholders with values
func replaceVars(text string, vars *TemplateVars) string {
	replacements := map[string]string{
		"{app_name}":                vars.AppName,
		"{APP_NAME}":                strings.ToUpper(vars.AppName),
		"{app_url}":                 vars.AppURL,
		"{fqdn}":                    vars.FQDN,
		"{onion_url}":               vars.OnionURL,
		"{admin_email}":             vars.AdminEmail,
		"{recipient_email}":         vars.RecipientEmail,
		"{recipient_username}":      vars.RecipientUsername,
		"{timestamp}":               vars.Timestamp,
		"{year}":                    vars.Year,
		"{reset_link}":              vars.ResetLink,
		"{verify_link}":             vars.VerifyLink,
		"{login_url}":               vars.LoginURL,
		"{profile_url}":             vars.ProfileURL,
		"{admin_url}":               vars.AdminURL,
		"{expires}":                 vars.Expires,
		"{ip}":                      vars.IP,
		"{location}":                vars.Location,
		"{device}":                  vars.Device,
		"{time}":                    vars.Time,
		"{event}":                   vars.Event,
		"{details}":                 vars.Details,
		"{method}":                  vars.Method,
		"{filename}":                vars.Filename,
		"{size}":                    vars.Size,
		"{error}":                   vars.Error,
		"{task_name}":               vars.TaskName,
		"{next_run}":                vars.NextRun,
		"{expires_in}":              vars.ExpiresIn,
		"{expiry_date}":             vars.ExpiryDate,
		"{valid_until}":             vars.ValidUntil,
		"{breach_id}":               vars.BreachID,
		"{breach_date}":             vars.BreachDate,
		"{breach_type}":             vars.BreachType,
		"{breach_summary}":          vars.BreachSummary,
		"{affected_data}":           vars.AffectedData,
		"{recommended_actions}":     vars.RecommendedActions,
		"{contact_email}":           vars.ContactEmail,
		"{contact_phone}":           vars.ContactPhone,
		"{dpo_contact}":             vars.DPOContact,
		"{regulatory_notice}":       vars.RegulatoryNotice,
		"{notification_deadline}":   vars.NotificationDeadline,
		"{severity}":                vars.Severity,
		"{detection_method}":        vars.DetectionMethod,
		"{trigger}":                 vars.Trigger,
		"{source_ip}":               vars.SourceIP,
		"{affected_scope}":          vars.AffectedScope,
		"{affected_users}":          vars.AffectedUsers,
		"{auto_actions}":            vars.AutoActions,
		"{compliance_requirements}": vars.ComplianceRequirements,
		"{notify_deadline}":         vars.NotifyDeadline,
	}

	for placeholder, value := range replacements {
		text = strings.ReplaceAll(text, placeholder, value)
	}

	return text
}

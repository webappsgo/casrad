// Package service — Tests for EmailTemplateService.
// Covers: parseTemplate (unexported), replaceVars (unexported),
// NewEmailTemplateService, SetAppInfo, LoadTemplate (custom file path),
// RenderTemplate, InvalidateCache, SaveCustomTemplate, ResetToDefault,
// IsCustom, ListTemplates.
package service

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestEmailSvc creates a service with an empty embedded FS and a temp custom dir.
func newTestEmailSvc(t *testing.T) (*EmailTemplateService, string) {
	t.Helper()
	customDir := t.TempDir()
	svc := NewEmailTemplateService(embed.FS{}, customDir)
	svc.SetAppInfo("TestApp", "https://example.com", "example.com", "admin@example.com")
	return svc, customDir
}

// writeCustomTemplate writes a raw template file to the custom directory.
func writeCustomTemplate(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name+".txt")
	if err := os.WriteFile(path, []byte(content), 0640); err != nil {
		t.Fatalf("write custom template %q: %v", name, err)
	}
}

// --- parseTemplate (unexported) ---

func TestParseTemplateWithSubjectAndBody(t *testing.T) {
	t.Parallel()
	content := "Subject: Hello World\n---\nThis is the body."
	tmpl := parseTemplate("test", content)
	if tmpl.Name != "test" {
		t.Errorf("Name = %q, want test", tmpl.Name)
	}
	if tmpl.Subject != "Hello World" {
		t.Errorf("Subject = %q, want Hello World", tmpl.Subject)
	}
	if !strings.Contains(tmpl.Body, "This is the body") {
		t.Errorf("Body = %q, want contains 'This is the body'", tmpl.Body)
	}
}

func TestParseTemplateNoSeparator(t *testing.T) {
	t.Parallel()
	content := "Subject: OneLine\nBody text here."
	tmpl := parseTemplate("nosep", content)
	if tmpl.Subject != "OneLine" {
		t.Errorf("Subject = %q, want OneLine", tmpl.Subject)
	}
}

func TestParseTemplateSingleLine(t *testing.T) {
	t.Parallel()
	tmpl := parseTemplate("single", "just a line with no newline")
	if tmpl.Body != "just a line with no newline" {
		t.Errorf("Body = %q, want full content as body", tmpl.Body)
	}
}

func TestParseTemplateEmpty(t *testing.T) {
	t.Parallel()
	tmpl := parseTemplate("empty", "")
	if tmpl == nil {
		t.Fatal("parseTemplate should not return nil")
	}
}

// --- replaceVars (unexported) ---

func TestReplaceVarsSubstitution(t *testing.T) {
	t.Parallel()
	text := "Hello {recipient_username}, reset at {reset_link}"
	vars := &TemplateVars{
		RecipientUsername: "alice",
		ResetLink:         "https://example.com/reset/abc",
	}
	got := replaceVars(text, vars)
	if !strings.Contains(got, "alice") {
		t.Errorf("replaceVars missing username replacement: %q", got)
	}
	if !strings.Contains(got, "https://example.com/reset/abc") {
		t.Errorf("replaceVars missing reset_link replacement: %q", got)
	}
}

func TestReplaceVarsAppNameUppercase(t *testing.T) {
	t.Parallel()
	text := "{app_name} / {APP_NAME}"
	vars := &TemplateVars{AppName: "casrad"}
	got := replaceVars(text, vars)
	if !strings.Contains(got, "casrad") {
		t.Errorf("replaceVars missing lowercase app_name: %q", got)
	}
	if !strings.Contains(got, "CASRAD") {
		t.Errorf("replaceVars missing uppercase APP_NAME: %q", got)
	}
}

func TestReplaceVarsNoMatch(t *testing.T) {
	t.Parallel()
	text := "static text with no placeholders"
	vars := &TemplateVars{}
	got := replaceVars(text, vars)
	if got != text {
		t.Errorf("replaceVars no match: got %q, want %q", got, text)
	}
}

// --- SetAppInfo ---

func TestSetAppInfo(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	svc.SetAppInfo("NewApp", "https://newapp.io", "newapp.io", "new@newapp.io")
	if svc.appName != "NewApp" {
		t.Errorf("appName = %q, want NewApp", svc.appName)
	}
	if svc.appURL != "https://newapp.io" {
		t.Errorf("appURL = %q, want https://newapp.io", svc.appURL)
	}
	if svc.fqdn != "newapp.io" {
		t.Errorf("fqdn = %q, want newapp.io", svc.fqdn)
	}
	if svc.adminEmail != "new@newapp.io" {
		t.Errorf("adminEmail = %q, want new@newapp.io", svc.adminEmail)
	}
}

// --- LoadTemplate ---

func TestLoadTemplateCustomFile(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "welcome", "Subject: Welcome!\n---\nHello {recipient_username}!")

	tmpl, err := svc.LoadTemplate("welcome")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	if tmpl.Subject != "Welcome!" {
		t.Errorf("Subject = %q, want Welcome!", tmpl.Subject)
	}
}

func TestLoadTemplateCached(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "cached_tmpl", "Subject: Cached\n---\nBody")

	tmpl1, err := svc.LoadTemplate("cached_tmpl")
	if err != nil {
		t.Fatalf("first LoadTemplate: %v", err)
	}
	tmpl2, err := svc.LoadTemplate("cached_tmpl")
	if err != nil {
		t.Fatalf("second LoadTemplate: %v", err)
	}
	if tmpl1 != tmpl2 {
		t.Error("second LoadTemplate should return cached instance")
	}
}

func TestLoadTemplateNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	_, err := svc.LoadTemplate("nonexistent_template")
	if err == nil {
		t.Error("LoadTemplate(missing) should return error")
	}
}

// --- RenderTemplate ---

func TestRenderTemplateSubstitutesVars(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "render_test", "Subject: Welcome {recipient_username}\n---\nHi {recipient_username}!")

	subject, body, err := svc.RenderTemplate("render_test", &TemplateVars{
		RecipientUsername: "bob",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	if !strings.Contains(subject, "bob") {
		t.Errorf("subject = %q, want contains 'bob'", subject)
	}
	if !strings.Contains(body, "bob") {
		t.Errorf("body = %q, want contains 'bob'", body)
	}
}

func TestRenderTemplateSetsGlobalVars(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "global_test", "Subject: From {app_name}\n---\nVisit {app_url}")

	_, body, err := svc.RenderTemplate("global_test", &TemplateVars{})
	if err != nil {
		t.Fatalf("RenderTemplate globals: %v", err)
	}
	if !strings.Contains(body, "https://example.com") {
		t.Errorf("body = %q, want contains app_url", body)
	}
}

func TestRenderTemplateMissing(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	_, _, err := svc.RenderTemplate("no_such_template", &TemplateVars{})
	if err == nil {
		t.Error("RenderTemplate(missing) should return error")
	}
}

// --- InvalidateCache ---

func TestInvalidateCache(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "invalidate_test", "Subject: Old\n---\nOld body")

	tmpl1, _ := svc.LoadTemplate("invalidate_test")
	svc.InvalidateCache()

	// Overwrite the file to a new content
	writeCustomTemplate(t, customDir, "invalidate_test", "Subject: New\n---\nNew body")

	tmpl2, err := svc.LoadTemplate("invalidate_test")
	if err != nil {
		t.Fatalf("LoadTemplate after invalidate: %v", err)
	}
	// After cache invalidation, the new file content should be loaded
	if tmpl1 == tmpl2 {
		t.Error("after InvalidateCache, LoadTemplate should reload from disk")
	}
	if tmpl2.Subject != "New" {
		t.Errorf("Subject after reload = %q, want New", tmpl2.Subject)
	}
}

// --- SaveCustomTemplate ---

func TestSaveCustomTemplate(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)

	if err := svc.SaveCustomTemplate("mytmpl", "My Subject", "My body text"); err != nil {
		t.Fatalf("SaveCustomTemplate: %v", err)
	}

	// Verify file exists
	path := filepath.Join(customDir, "mytmpl.txt")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("template file not created: %v", err)
	}
}

func TestSaveCustomTemplateInvalidatesCache(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "save_test", "Subject: Old\n---\nOld")
	svc.LoadTemplate("save_test")

	if err := svc.SaveCustomTemplate("save_test", "New Subject", "New body"); err != nil {
		t.Fatalf("SaveCustomTemplate: %v", err)
	}

	tmpl, _ := svc.LoadTemplate("save_test")
	if tmpl != nil && tmpl.Subject == "Old" {
		t.Error("SaveCustomTemplate should invalidate cached template")
	}
}

// --- ResetToDefault ---

func TestResetToDefault(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "reset_test", "Subject: Custom\n---\nCustom body")

	if err := svc.ResetToDefault("reset_test"); err != nil {
		t.Fatalf("ResetToDefault: %v", err)
	}

	path := filepath.Join(customDir, "reset_test.txt")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("ResetToDefault should delete the custom template file")
	}
}

func TestResetToDefaultNonexistent(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	// Should not error when file doesn't exist
	if err := svc.ResetToDefault("no_such_template"); err != nil {
		t.Errorf("ResetToDefault(nonexistent): %v", err)
	}
}

// --- IsCustom ---

func TestIsCustomTrue(t *testing.T) {
	t.Parallel()
	svc, customDir := newTestEmailSvc(t)
	writeCustomTemplate(t, customDir, "custom_check", "Subject: X\n---\nY")

	if !svc.IsCustom("custom_check") {
		t.Error("IsCustom should return true when custom file exists")
	}
}

func TestIsCustomFalse(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	if svc.IsCustom("no_such_template") {
		t.Error("IsCustom should return false when no custom file exists")
	}
}

// --- ListTemplates ---

func TestListTemplatesCount(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	templates := svc.ListTemplates()
	if len(templates) == 0 {
		t.Error("ListTemplates should return non-empty list")
	}
}

func TestListTemplatesContainsRequired(t *testing.T) {
	t.Parallel()
	svc, _ := newTestEmailSvc(t)
	templates := svc.ListTemplates()

	required := []string{"welcome", "password_reset", "email_verify", "login_alert"}
	set := make(map[string]bool, len(templates))
	for _, name := range templates {
		set[name] = true
	}
	for _, name := range required {
		if !set[name] {
			t.Errorf("ListTemplates missing required template %q", name)
		}
	}
}

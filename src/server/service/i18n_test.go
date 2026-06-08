// Package service — Tests for I18nService.
// Covers: DefaultI18nConfig, NewI18nService, IsAvailable, GetDirection, Translate,
// GetLanguage (query param, cookie, Accept-Language, default), SetLangCookie,
// SetLanguageCookie, GetAvailableLanguages, GetTemplateData, LoadTranslations.
package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildI18nSvc creates an I18nService directly without using embed.FS.
// Translations are injected into the internal map for deterministic testing.
func buildI18nSvc(enabled bool, defaultLang string, langs []string) *I18nService {
	cfg := I18nConfig{
		Enabled:            enabled,
		DefaultLanguage:    defaultLang,
		FallbackLanguage:   defaultLang,
		AvailableLanguages: langs,
		CookieName:         "lang",
		CookieMaxAge:       31536000,
	}
	return &I18nService{
		config:       cfg,
		translations: make(map[string]*Translation),
	}
}

func TestDefaultI18nConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultI18nConfig()
	if !cfg.Enabled {
		t.Error("DefaultI18nConfig: Enabled should be true")
	}
	if cfg.DefaultLanguage != "en" {
		t.Errorf("DefaultI18nConfig: DefaultLanguage = %q, want en", cfg.DefaultLanguage)
	}
	if cfg.FallbackLanguage != "en" {
		t.Errorf("DefaultI18nConfig: FallbackLanguage = %q, want en", cfg.FallbackLanguage)
	}
	if cfg.CookieName != "lang" {
		t.Errorf("DefaultI18nConfig: CookieName = %q, want lang", cfg.CookieName)
	}
	required := map[string]bool{"en": false, "es": false, "zh": false, "fr": false, "ar": false, "de": false, "ja": false}
	for _, lang := range cfg.AvailableLanguages {
		required[lang] = true
	}
	for lang, found := range required {
		if !found {
			t.Errorf("required language %q missing from DefaultI18nConfig.AvailableLanguages", lang)
		}
	}
}

func TestIsAvailable(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es", "fr"})
	if !svc.IsAvailable("en") {
		t.Error("en should be available")
	}
	if !svc.IsAvailable("es") {
		t.Error("es should be available")
	}
	if svc.IsAvailable("de") {
		t.Error("de should not be available")
	}
	if svc.IsAvailable("") {
		t.Error("empty string should not be available")
	}
}

func TestGetDirectionRTL(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", nil)
	for _, lang := range []string{"ar", "he", "fa", "ur"} {
		if got := svc.GetDirection(lang); got != "rtl" {
			t.Errorf("GetDirection(%q) = %q, want rtl", lang, got)
		}
	}
}

func TestGetDirectionLTR(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", nil)
	for _, lang := range []string{"en", "es", "fr", "de", "ja", "zh", ""} {
		if got := svc.GetDirection(lang); got != "ltr" {
			t.Errorf("GetDirection(%q) = %q, want ltr", lang, got)
		}
	}
}

func TestRTLLanguagesMap(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"ar", "he", "fa", "ur"} {
		if !RTLLanguages[lang] {
			t.Errorf("RTLLanguages[%q] should be true", lang)
		}
	}
	for _, lang := range []string{"en", "es", "zh", "fr", "de"} {
		if RTLLanguages[lang] {
			t.Errorf("RTLLanguages[%q] should be false", lang)
		}
	}
}

func TestTranslateReturnsFallbackKey(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	got := svc.Translate("en", "common.save")
	if got != "common.save" {
		t.Errorf("Translate with no translations = %q, want key as fallback", got)
	}
}

func TestTranslateExistingKey(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	svc.translations["en"] = &Translation{
		Data: map[string]interface{}{
			"common": map[string]interface{}{
				"save":   "Save",
				"cancel": "Cancel",
			},
		},
	}
	if got := svc.Translate("en", "common.save"); got != "Save" {
		t.Errorf("Translate(en, common.save) = %q, want Save", got)
	}
	if got := svc.Translate("en", "common.cancel"); got != "Cancel" {
		t.Errorf("Translate(en, common.cancel) = %q, want Cancel", got)
	}
}

func TestTranslateFallsBackToFallbackLang(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es"})
	svc.translations["en"] = &Translation{
		Data: map[string]interface{}{
			"greeting": "Hello",
		},
	}
	if got := svc.Translate("es", "greeting"); got != "Hello" {
		t.Errorf("Translate fallback: got %q, want Hello", got)
	}
}

func TestTranslateMissingKeyReturnsKey(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	svc.translations["en"] = &Translation{
		Data: map[string]interface{}{"greeting": "Hello"},
	}
	got := svc.Translate("en", "nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("Translate(missing key) = %q, want key itself", got)
	}
}

func TestTranslateNestedKey(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	svc.translations["en"] = &Translation{
		Data: map[string]interface{}{
			"auth": map[string]interface{}{
				"error": map[string]interface{}{
					"invalid_credentials": "Invalid credentials",
				},
			},
		},
	}
	got := svc.Translate("en", "auth.error.invalid_credentials")
	if got != "Invalid credentials" {
		t.Errorf("Translate nested key = %q, want Invalid credentials", got)
	}
}

func TestGetLanguageQueryParam(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es", "fr"})
	r := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
	if got := svc.GetLanguage(r); got != "es" {
		t.Errorf("GetLanguage with ?lang=es = %q, want es", got)
	}
}

func TestGetLanguageQueryParamUnknown(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es"})
	r := httptest.NewRequest(http.MethodGet, "/?lang=xx", nil)
	if got := svc.GetLanguage(r); got != "en" {
		t.Errorf("unknown ?lang=xx should fall back to default en, got %q", got)
	}
}

func TestGetLanguageCookie(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "fr"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
	if got := svc.GetLanguage(r); got != "fr" {
		t.Errorf("GetLanguage from cookie = %q, want fr", got)
	}
}

func TestGetLanguageCookieUnknown(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "lang", Value: "xx"})
	if got := svc.GetLanguage(r); got != "en" {
		t.Errorf("unknown cookie lang should fall to default, got %q", got)
	}
}

func TestGetLanguageAcceptLanguageHeader(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "de", "fr"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	if got := svc.GetLanguage(r); got != "de" {
		t.Errorf("GetLanguage from Accept-Language = %q, want de", got)
	}
}

func TestGetLanguageAcceptLanguageExactMatch(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "fr"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "fr")
	if got := svc.GetLanguage(r); got != "fr" {
		t.Errorf("GetLanguage exact Accept-Language = %q, want fr", got)
	}
}

func TestGetLanguageDefault(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := svc.GetLanguage(r); got != "en" {
		t.Errorf("GetLanguage default = %q, want en", got)
	}
}

func TestGetLanguageDisabled(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(false, "en", []string{"en", "es"})
	r := httptest.NewRequest(http.MethodGet, "/?lang=es", nil)
	if got := svc.GetLanguage(r); got != "en" {
		t.Errorf("disabled i18n should return default, got %q", got)
	}
}

func TestGetAvailableLanguages(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es"})
	svc.translations["en"] = &Translation{Meta: LanguageMeta{Language: "en", Name: "English", Direction: "ltr"}}
	svc.translations["es"] = &Translation{Meta: LanguageMeta{Language: "es", Name: "Spanish", Direction: "ltr"}}
	langs := svc.GetAvailableLanguages()
	if len(langs) != 2 {
		t.Errorf("GetAvailableLanguages count = %d, want 2", len(langs))
	}
}

func TestGetAvailableLanguagesEmptyWhenNoTranslations(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	langs := svc.GetAvailableLanguages()
	if len(langs) != 0 {
		t.Errorf("GetAvailableLanguages with no translations = %d, want 0", len(langs))
	}
}

func TestGetTemplateData(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "ar"})
	svc.translations["en"] = &Translation{Meta: LanguageMeta{Language: "en"}}
	svc.translations["ar"] = &Translation{Meta: LanguageMeta{Language: "ar"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	data := svc.GetTemplateData(r)

	if data.Lang != "en" {
		t.Errorf("TemplateData.Lang = %q, want en", data.Lang)
	}
	if data.Dir != "ltr" {
		t.Errorf("TemplateData.Dir = %q, want ltr", data.Dir)
	}
}

func TestGetTemplateDataRTL(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "ar", []string{"ar"})
	svc.translations["ar"] = &Translation{Meta: LanguageMeta{Language: "ar", Direction: "rtl"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	data := svc.GetTemplateData(r)

	if data.Dir != "rtl" {
		t.Errorf("TemplateData.Dir for ar = %q, want rtl", data.Dir)
	}
}

func TestSetLangCookieWritesCookie(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	w := httptest.NewRecorder()
	svc.SetLangCookie(w, "fr")
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "lang" && c.Value == "fr" {
			found = true
		}
	}
	if !found {
		t.Error("SetLangCookie should write a lang=fr cookie")
	}
}

func TestSetLanguageCookieIgnoresUnavailable(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en"})
	w := httptest.NewRecorder()
	svc.SetLanguageCookie(w, "xx")
	if len(w.Result().Cookies()) != 0 {
		t.Error("SetLanguageCookie should not write a cookie for an unavailable language")
	}
}

func TestSetLanguageCookieWritesWhenAvailable(t *testing.T) {
	t.Parallel()
	svc := buildI18nSvc(true, "en", []string{"en", "es"})
	w := httptest.NewRecorder()
	svc.SetLanguageCookie(w, "es")
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "lang" && c.Value == "es" {
			found = true
		}
	}
	if !found {
		t.Error("SetLanguageCookie should write lang=es cookie when available")
	}
}

func TestLoadTranslationsWithEmptyFS(t *testing.T) {
	t.Parallel()
	cfg := DefaultI18nConfig()
	svc := &I18nService{
		config:       cfg,
		translations: make(map[string]*Translation),
		// localesFS is zero-value embed.FS — ReadFile always returns error
	}
	err := svc.LoadTranslations()
	if err != nil {
		t.Errorf("LoadTranslations with empty FS should not error, got: %v", err)
	}
	if len(svc.translations) != 0 {
		t.Errorf("translations count = %d, want 0 with empty FS", len(svc.translations))
	}
}

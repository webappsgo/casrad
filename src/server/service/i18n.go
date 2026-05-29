// Package service provides server services
// See AI.md PART 31: I18N & A11Y
package service

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// I18nConfig holds internationalization configuration
type I18nConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	DefaultLanguage    string   `yaml:"default_language" json:"default_language"`
	FallbackLanguage   string   `yaml:"fallback_language" json:"fallback_language"`
	AvailableLanguages []string `yaml:"available_languages" json:"available_languages"`
	CookieName         string   `yaml:"cookie_name" json:"cookie_name"`
	CookieMaxAge       int      `yaml:"cookie_max_age" json:"cookie_max_age"` // seconds
}

// DefaultI18nConfig returns default i18n configuration
func DefaultI18nConfig() I18nConfig {
	return I18nConfig{
		Enabled:         true,
		DefaultLanguage: "en",
		FallbackLanguage: "en",
		// All 7 required languages per AI.md PART 31
		AvailableLanguages: []string{"en", "es", "zh", "fr", "ar", "de", "ja"},
		CookieName:         "lang",
		CookieMaxAge:       31536000, // 1 year
	}
}

// LanguageMeta holds metadata for a language
type LanguageMeta struct {
	Language   string `json:"language"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	Direction  string `json:"direction"` // ltr or rtl
	Version    string `json:"version"`
}

// Translation holds a complete translation file
type Translation struct {
	Meta LanguageMeta           `json:"meta"`
	Data map[string]interface{} `json:"data"` // Nested translation keys
}

// I18nService provides internationalization services
type I18nService struct {
	config       I18nConfig
	translations map[string]*Translation
	mu           sync.RWMutex
	localesFS    embed.FS
}

// RTLLanguages lists languages that use right-to-left text
var RTLLanguages = map[string]bool{
	"ar": true, // Arabic
	"he": true, // Hebrew
	"fa": true, // Persian
	"ur": true, // Urdu
}

// NewI18nService creates a new i18n service
func NewI18nService(config I18nConfig, localesFS embed.FS) *I18nService {
	return &I18nService{
		config:       config,
		translations: make(map[string]*Translation),
		localesFS:    localesFS,
	}
}

// LoadTranslations loads all available translations from embedded filesystem
func (s *I18nService) LoadTranslations() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, lang := range s.config.AvailableLanguages {
		// Embedded FS via //go:embed *.json uses bare filenames (no directory prefix)
		filename := lang + ".json"
		data, err := s.localesFS.ReadFile(filename)
		if err != nil {
			continue // Skip missing translations
		}

		var rawData map[string]interface{}
		if err := json.Unmarshal(data, &rawData); err != nil {
			continue
		}

		// Extract meta
		var meta LanguageMeta
		if metaData, ok := rawData["meta"].(map[string]interface{}); ok {
			if v, ok := metaData["language"].(string); ok {
				meta.Language = v
			}
			if v, ok := metaData["name"].(string); ok {
				meta.Name = v
			}
			if v, ok := metaData["native_name"].(string); ok {
				meta.NativeName = v
			}
			if v, ok := metaData["direction"].(string); ok {
				meta.Direction = v
			}
			if v, ok := metaData["version"].(string); ok {
				meta.Version = v
			}
		}

		// Remove meta from data
		delete(rawData, "meta")

		s.translations[lang] = &Translation{
			Meta: meta,
			Data: rawData,
		}
	}

	return nil
}

// GetLanguage determines the best language for a request.
// Priority per AI.md PART 31: ?lang= query param → lang cookie → Accept-Language header → default "en"
func (s *I18nService) GetLanguage(r *http.Request) string {
	if !s.config.Enabled {
		return s.config.DefaultLanguage
	}

	// 1. ?lang= query parameter (highest priority — also sets cookie via SetLangCookie)
	if lang := r.URL.Query().Get("lang"); lang != "" && s.IsAvailable(lang) {
		return lang
	}

	// 2. lang cookie
	if cookie, err := r.Cookie(s.config.CookieName); err == nil {
		if s.IsAvailable(cookie.Value) {
			return cookie.Value
		}
	}

	// 3. Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		if lang := s.parseAcceptLanguage(acceptLang); lang != "" {
			return lang
		}
	}

	// 4. Default
	return s.config.DefaultLanguage
}

// SetLangCookie writes a long-lived language preference cookie to the response.
// Call this when a ?lang= query param is present to persist the choice.
func (s *I18nService) SetLangCookie(w http.ResponseWriter, lang string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName,
		Value:    lang,
		Path:     "/",
		MaxAge:   s.config.CookieMaxAge,
		HttpOnly: false, // JS needs to read it for client-side rendering
		SameSite: http.SameSiteLaxMode,
	})
}

// parseAcceptLanguage parses Accept-Language header and returns best match
func (s *I18nService) parseAcceptLanguage(header string) string {
	// Simple parsing - more sophisticated would use q values
	parts := strings.Split(header, ",")
	for _, part := range parts {
		// Remove quality value if present
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		// Try exact match
		if s.IsAvailable(lang) {
			return lang
		}
		// Try base language (e.g., "en" from "en-US")
		if idx := strings.Index(lang, "-"); idx > 0 {
			baseLang := lang[:idx]
			if s.IsAvailable(baseLang) {
				return baseLang
			}
		}
	}
	return ""
}

// IsAvailable checks if a language is available
func (s *I18nService) IsAvailable(lang string) bool {
	for _, l := range s.config.AvailableLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// Translate returns a translation for a key
func (s *I18nService) Translate(lang, key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try requested language
	if translation := s.getTranslation(lang, key); translation != "" {
		return translation
	}

	// Try fallback language
	if lang != s.config.FallbackLanguage {
		if translation := s.getTranslation(s.config.FallbackLanguage, key); translation != "" {
			return translation
		}
	}

	// Return key as fallback
	return key
}

// getTranslation gets a translation by navigating nested keys
func (s *I18nService) getTranslation(lang, key string) string {
	trans, ok := s.translations[lang]
	if !ok {
		return ""
	}

	// Navigate nested keys (e.g., "common.save")
	parts := strings.Split(key, ".")
	var current interface{} = trans.Data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}
	return ""
}

// GetDirection returns text direction for a language
func (s *I18nService) GetDirection(lang string) string {
	if RTLLanguages[lang] {
		return "rtl"
	}
	return "ltr"
}

// GetAvailableLanguages returns list of available languages
func (s *I18nService) GetAvailableLanguages() []LanguageMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var langs []LanguageMeta
	for _, lang := range s.config.AvailableLanguages {
		if trans, ok := s.translations[lang]; ok {
			langs = append(langs, trans.Meta)
		}
	}
	return langs
}

// SetLanguageCookie sets the language preference cookie
func (s *I18nService) SetLanguageCookie(w http.ResponseWriter, lang string) {
	if !s.IsAvailable(lang) {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.CookieName,
		Value:    lang,
		MaxAge:   s.config.CookieMaxAge,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// TemplateData returns i18n data for template rendering
type I18nTemplateData struct {
	Lang      string // Current language code
	Dir       string // Text direction (ltr/rtl)
	Languages []LanguageMeta
}

// GetTemplateData returns data for template rendering
func (s *I18nService) GetTemplateData(r *http.Request) I18nTemplateData {
	lang := s.GetLanguage(r)
	return I18nTemplateData{
		Lang:      lang,
		Dir:       s.GetDirection(lang),
		Languages: s.GetAvailableLanguages(),
	}
}

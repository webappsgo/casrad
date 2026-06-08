// Package service — Tests for GeoIP service.
// Covers: NewGeoIPService (nil config defaults, explicit config), IsEnabled,
// Configure, Lookup (valid IP, invalid IP, disabled), IsCountryDenied,
// CheckIP, LastUpdate, DatabasesLoaded, error variable values.
// Note: UpdateDatabases / Init require network and are not tested here.
package service

import (
	"errors"
	"testing"
)

// --- error variables ---

func TestGeoIPErrorValues(t *testing.T) {
	t.Parallel()
	if ErrGeoIPNotEnabled == nil {
		t.Error("ErrGeoIPNotEnabled should be non-nil")
	}
	if ErrGeoIPNotLoaded == nil {
		t.Error("ErrGeoIPNotLoaded should be non-nil")
	}
	if ErrGeoIPCountryDenied == nil {
		t.Error("ErrGeoIPCountryDenied should be non-nil")
	}
	if errors.Is(ErrGeoIPNotEnabled, ErrGeoIPNotLoaded) {
		t.Error("GeoIP errors should be distinct")
	}
}

// --- NewGeoIPService ---

func TestNewGeoIPServiceNilConfigDefaults(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	if s == nil {
		t.Fatal("NewGeoIPService(nil) returned nil")
	}
	if !s.config.Enabled {
		t.Error("default config should have Enabled=true")
	}
	if !s.config.EnableASN {
		t.Error("default config should have EnableASN=true")
	}
	if !s.config.EnableCountry {
		t.Error("default config should have EnableCountry=true")
	}
	if !s.config.EnableCity {
		t.Error("default config should have EnableCity=true")
	}
}

func TestNewGeoIPServiceExplicitConfig(t *testing.T) {
	t.Parallel()
	cfg := &GeoIPConfig{
		Enabled:       false,
		EnableASN:     false,
		EnableCountry: true,
	}
	s := NewGeoIPService(cfg)
	if s == nil {
		t.Fatal("NewGeoIPService returned nil")
	}
	if s.config.Enabled {
		t.Error("config.Enabled should be false")
	}
	if s.config.EnableASN {
		t.Error("config.EnableASN should be false")
	}
	if !s.config.EnableCountry {
		t.Error("config.EnableCountry should be true")
	}
}

func TestNewGeoIPServiceInitializesDBLoaded(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	loaded := s.DatabasesLoaded()
	if loaded == nil {
		t.Error("DatabasesLoaded() should return non-nil map")
	}
}

// --- IsEnabled ---

func TestIsEnabledTrueWhenEnabled(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	if !s.IsEnabled() {
		t.Error("IsEnabled() should return true when config.Enabled=true")
	}
}

func TestIsEnabledFalseWhenDisabled(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: false})
	if s.IsEnabled() {
		t.Error("IsEnabled() should return false when config.Enabled=false")
	}
}

func TestIsEnabledFalseWhenNilConfig(t *testing.T) {
	t.Parallel()
	s := &GeoIPService{}
	if s.IsEnabled() {
		t.Error("IsEnabled() with nil config should return false")
	}
}

// --- Configure ---

func TestConfigureUpdatesEnabled(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	s.Configure(&GeoIPConfig{Enabled: false})
	if s.IsEnabled() {
		t.Error("Configure should update enabled state to false")
	}
}

func TestConfigureUpdatesConfig(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	newCfg := &GeoIPConfig{Enabled: true, Dir: "/tmp/geoip-test"}
	s.Configure(newCfg)
	if s.config.Dir != "/tmp/geoip-test" {
		t.Errorf("Configure() Dir = %q, want /tmp/geoip-test", s.config.Dir)
	}
}

// --- Lookup ---

func TestLookupDisabledReturnsError(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: false})
	_, err := s.Lookup("8.8.8.8")
	if !errors.Is(err, ErrGeoIPNotEnabled) {
		t.Errorf("Lookup on disabled service err = %v, want ErrGeoIPNotEnabled", err)
	}
}

func TestLookupInvalidIPReturnsError(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	_, err := s.Lookup("not-an-ip")
	if err == nil {
		t.Error("Lookup with invalid IP should return error")
	}
}

func TestLookupValidIPReturnsResult(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	result, err := s.Lookup("8.8.8.8")
	if err != nil {
		t.Errorf("Lookup(8.8.8.8) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Lookup(8.8.8.8) returned nil result")
	}
	if result.IP != "8.8.8.8" {
		t.Errorf("Lookup result.IP = %q, want 8.8.8.8", result.IP)
	}
}

func TestLookupIPv6ReturnsResult(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	result, err := s.Lookup("2001:4860:4860::8888")
	if err != nil {
		t.Errorf("Lookup(IPv6) unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Lookup(IPv6) returned nil result")
	}
}

// --- IsCountryDenied ---

func TestIsCountryDeniedFalseWhenDisabled(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: false, DenyCountries: []string{"US"}})
	if s.IsCountryDenied("US") {
		t.Error("IsCountryDenied should return false when GeoIP is disabled")
	}
}

func TestIsCountryDeniedFalseWhenNoDenyList(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	if s.IsCountryDenied("US") {
		t.Error("IsCountryDenied should return false with empty deny list")
	}
}

func TestIsCountryDeniedTrueForDeniedCountry(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true, DenyCountries: []string{"CN", "RU"}})
	if !s.IsCountryDenied("CN") {
		t.Error("IsCountryDenied(CN) should be true")
	}
	if !s.IsCountryDenied("RU") {
		t.Error("IsCountryDenied(RU) should be true")
	}
}

func TestIsCountryDeniedFalseForAllowedCountry(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true, DenyCountries: []string{"CN"}})
	if s.IsCountryDenied("US") {
		t.Error("IsCountryDenied(US) should be false when only CN is denied")
	}
}

// --- CheckIP ---

func TestCheckIPDisabledAlwaysNil(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: false})
	if err := s.CheckIP("1.2.3.4"); err != nil {
		t.Errorf("CheckIP on disabled service = %v, want nil", err)
	}
}

func TestCheckIPValidIPNoDenyListReturnsNil(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	if err := s.CheckIP("8.8.8.8"); err != nil {
		t.Errorf("CheckIP(valid IP, no deny list) = %v, want nil", err)
	}
}

func TestCheckIPInvalidIPReturnsNil(t *testing.T) {
	t.Parallel()
	// Invalid IPs don't block — lookup errors are ignored per design
	s := NewGeoIPService(&GeoIPConfig{Enabled: true})
	if err := s.CheckIP("not-an-ip"); err != nil {
		t.Errorf("CheckIP(invalid IP) = %v, want nil (lookup errors are non-blocking)", err)
	}
}

// --- LastUpdate ---

func TestLastUpdateZeroOnNewService(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	ts := s.LastUpdate()
	if !ts.IsZero() {
		t.Errorf("LastUpdate() on new service = %v, want zero time", ts)
	}
}

// --- DatabasesLoaded ---

func TestDatabasesLoadedEmptyOnNewService(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	loaded := s.DatabasesLoaded()
	if len(loaded) != 0 {
		t.Errorf("DatabasesLoaded() on new service = %v, want empty map", loaded)
	}
}

func TestDatabasesLoadedReturnsCopy(t *testing.T) {
	t.Parallel()
	s := NewGeoIPService(nil)
	loaded1 := s.DatabasesLoaded()
	loaded2 := s.DatabasesLoaded()
	// Should return independent copies
	loaded1["test"] = true
	if loaded2["test"] {
		t.Error("DatabasesLoaded() should return a copy, not the internal map")
	}
}

// --- GeoIPDatabaseURLs ---

func TestGeoIPDatabaseURLsNotEmpty(t *testing.T) {
	t.Parallel()
	for key, url := range GeoIPDatabaseURLs {
		if url == "" {
			t.Errorf("GeoIPDatabaseURLs[%q] is empty", key)
		}
	}
}

func TestGeoIPDatabaseURLsHasExpectedKeys(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"asn", "country", "city", "whois"} {
		if _, ok := GeoIPDatabaseURLs[key]; !ok {
			t.Errorf("GeoIPDatabaseURLs missing key %q", key)
		}
	}
}

// --- GeoIPResult struct ---

func TestGeoIPResultFieldsAccessible(t *testing.T) {
	t.Parallel()
	r := GeoIPResult{
		IP:          "1.2.3.4",
		CountryCode: "US",
		City:        "San Francisco",
	}
	if r.IP != "1.2.3.4" {
		t.Errorf("GeoIPResult.IP = %q, want 1.2.3.4", r.IP)
	}
	if r.CountryCode != "US" {
		t.Errorf("GeoIPResult.CountryCode = %q, want US", r.CountryCode)
	}
}

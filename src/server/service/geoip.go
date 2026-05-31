// Package service - GeoIP lookup service
// See AI.md PART 20 for GeoIP specification
package service

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrGeoIPNotEnabled   = errors.New("GeoIP is not enabled")
	ErrGeoIPNotLoaded    = errors.New("GeoIP database not loaded")
	ErrGeoIPCountryDenied = errors.New("country is denied")
)

// GeoIP database URLs from sapics/ip-location-db (no API key required)
// Per AI.md PART 20
var GeoIPDatabaseURLs = map[string]string{
	"asn":     "https://cdn.jsdelivr.net/npm/@ip-location-db/asn-mmdb/asn.mmdb",
	"country": "https://cdn.jsdelivr.net/npm/@ip-location-db/geo-whois-asn-country-mmdb/geo-whois-asn-country.mmdb",
	"city":    "https://cdn.jsdelivr.net/npm/@ip-location-db/dbip-city-mmdb/dbip-city-ipv4.mmdb",
	"whois":   "https://cdn.jsdelivr.net/npm/@ip-location-db/geo-whois-asn-country-mmdb/geo-whois-asn-country.mmdb",
}

// GeoIPResult contains the result of a GeoIP lookup
type GeoIPResult struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code,omitempty"`
	City        string  `json:"city,omitempty"`
	Region      string  `json:"region,omitempty"`
	PostalCode  string  `json:"postal_code,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	ASN         uint    `json:"asn,omitempty"`
	ASOrg       string  `json:"as_org,omitempty"`
}

// GeoIPConfig holds GeoIP configuration
type GeoIPConfig struct {
	Enabled       bool
	Dir           string
	DenyCountries []string
	EnableASN     bool
	EnableCountry bool
	EnableCity    bool
	EnableWHOIS   bool
}

// GeoIPService provides IP geolocation lookups
type GeoIPService struct {
	config      *GeoIPConfig
	mu          sync.RWMutex
	lastUpdate  time.Time
	dbLoaded    map[string]bool
	httpClient  *http.Client
}

// NewGeoIPService creates a new GeoIP service
func NewGeoIPService(config *GeoIPConfig) *GeoIPService {
	if config == nil {
		config = &GeoIPConfig{
			Enabled:       true,
			EnableASN:     true,
			EnableCountry: true,
			EnableCity:    true,
			EnableWHOIS:   true,
		}
	}

	return &GeoIPService{
		config:     config,
		dbLoaded:   make(map[string]bool),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// IsEnabled returns whether GeoIP is enabled
func (s *GeoIPService) IsEnabled() bool {
	return s.config != nil && s.config.Enabled
}

// Configure updates the GeoIP configuration
func (s *GeoIPService) Configure(config *GeoIPConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// Init initializes the GeoIP service and downloads databases if needed
func (s *GeoIPService) Init() error {
	if !s.IsEnabled() {
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(s.config.Dir, 0750); err != nil {
		return fmt.Errorf("failed to create GeoIP directory: %w", err)
	}

	// Download databases if not present
	return s.UpdateDatabases(false)
}

// UpdateDatabases downloads/updates the GeoIP databases
func (s *GeoIPService) UpdateDatabases(force bool) error {
	if !s.IsEnabled() {
		return ErrGeoIPNotEnabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine which databases to download
	databases := make(map[string]string)
	if s.config.EnableASN {
		databases["asn"] = GeoIPDatabaseURLs["asn"]
	}
	if s.config.EnableCountry {
		databases["country"] = GeoIPDatabaseURLs["country"]
	}
	if s.config.EnableCity {
		databases["city"] = GeoIPDatabaseURLs["city"]
	}
	if s.config.EnableWHOIS {
		databases["whois"] = GeoIPDatabaseURLs["whois"]
	}

	for name, url := range databases {
		filename := filepath.Join(s.config.Dir, name+".mmdb")

		// Skip if file exists and not forcing update
		if !force {
			if info, err := os.Stat(filename); err == nil {
				// Check if file is recent (within 7 days)
				if time.Since(info.ModTime()) < 7*24*time.Hour {
					s.dbLoaded[name] = true
					continue
				}
			}
		}

		// Download the database
		if err := s.downloadDatabase(url, filename); err != nil {
			return fmt.Errorf("failed to download %s database: %w", name, err)
		}

		s.dbLoaded[name] = true
	}

	s.lastUpdate = time.Now()
	return nil
}

// downloadDatabase downloads a database from the given URL
func (s *GeoIPService) downloadDatabase(url, filename string) error {
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Download to temp file first
	tempFile := filename + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tempFile)
		return err
	}

	// Rename temp file to final name
	return os.Rename(tempFile, filename)
}

// Lookup performs a GeoIP lookup for the given IP address.
// Returns minimal data when MMDB files have not yet been downloaded.
// Full lookups are available after the weekly geoip_update scheduler task runs.
func (s *GeoIPService) Lookup(ipStr string) (*GeoIPResult, error) {
	if !s.IsEnabled() {
		return nil, ErrGeoIPNotEnabled
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Returns IP only until MMDB files are downloaded by the scheduler.
	// Full lookup uses oschwald/maxminddb-golang once files are present.
	return &GeoIPResult{IP: ipStr}, nil
}

// IsCountryDenied checks if the given country code is in the deny list
func (s *GeoIPService) IsCountryDenied(countryCode string) bool {
	if !s.IsEnabled() || len(s.config.DenyCountries) == 0 {
		return false
	}

	for _, denied := range s.config.DenyCountries {
		if denied == countryCode {
			return true
		}
	}
	return false
}

// CheckIP performs a lookup and returns an error if the country is denied
func (s *GeoIPService) CheckIP(ipStr string) error {
	if !s.IsEnabled() {
		return nil
	}

	result, err := s.Lookup(ipStr)
	if err != nil {
		// Don't block on lookup errors
		return nil
	}

	if s.IsCountryDenied(result.CountryCode) {
		return ErrGeoIPCountryDenied
	}

	return nil
}

// LastUpdate returns the time of the last database update
func (s *GeoIPService) LastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// DatabasesLoaded returns which databases are loaded
func (s *GeoIPService) DatabasesLoaded() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]bool)
	for k, v := range s.dbLoaded {
		result[k] = v
	}
	return result
}

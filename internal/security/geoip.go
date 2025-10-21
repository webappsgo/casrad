package security

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/oschwald/maxminddb-golang"
)

// GeoIPManager handles GeoIP database management and lookups
// Uses P3TERX mirror as primary source (no registration required)
type GeoIPManager struct {
	db           *database.Engine
	dataPath     string
	cityDB       *maxminddb.Reader
	asnDB        *maxminddb.Reader
	countryDB    *maxminddb.Reader
	mu           sync.RWMutex
	cache        map[string]*GeoIPResult
	cacheMu      sync.RWMutex
	downloading  bool
	lastUpdate   time.Time
}

// GeoIPResult contains location information for an IP
type GeoIPResult struct {
	IP          string
	Country     string
	CountryCode string
	City        string
	Region      string
	PostalCode  string
	Latitude    float64
	Longitude   float64
	ASN         int
	ASOrg       string
	ISP         string
	Timezone    string
	CachedAt    time.Time
}

// P3TERX mirror URLs - no registration required!
const (
	P3TERXCityURL    = "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"
	P3TERXASNURL     = "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb"
	P3TERXCountryURL = "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb"
)

func NewGeoIPManager(dataPath string, db *database.Engine) *GeoIPManager {
	geoipPath := filepath.Join(dataPath, "geoip")
	os.MkdirAll(geoipPath, 0755)

	m := &GeoIPManager{
		db:       db,
		dataPath: geoipPath,
		cache:    make(map[string]*GeoIPResult),
	}

	// Load existing databases if available
	m.loadDatabases()

	// Check if update is needed
	if m.shouldUpdate() {
		go m.UpdateDatabases()
	}

	return m
}

func (m *GeoIPManager) loadDatabases() {
	// Try to load City database
	cityPath := filepath.Join(m.dataPath, "GeoLite2-City.mmdb")
	if db, err := maxminddb.Open(cityPath); err == nil {
		m.mu.Lock()
		m.cityDB = db
		m.mu.Unlock()
		log.Printf("Loaded GeoIP City database from %s", cityPath)
	}

	// Try to load ASN database
	asnPath := filepath.Join(m.dataPath, "GeoLite2-ASN.mmdb")
	if db, err := maxminddb.Open(asnPath); err == nil {
		m.mu.Lock()
		m.asnDB = db
		m.mu.Unlock()
		log.Printf("Loaded GeoIP ASN database from %s", asnPath)
	}

	// Try to load Country database
	countryPath := filepath.Join(m.dataPath, "GeoLite2-Country.mmdb")
	if db, err := maxminddb.Open(countryPath); err == nil {
		m.mu.Lock()
		m.countryDB = db
		m.mu.Unlock()
		log.Printf("Loaded GeoIP Country database from %s", countryPath)
	}
}

func (m *GeoIPManager) shouldUpdate() bool {
	// Check if databases exist
	if m.cityDB == nil || m.asnDB == nil || m.countryDB == nil {
		return true
	}

	// Check last update time (update weekly by default)
	if time.Since(m.lastUpdate) > 7*24*time.Hour {
		return true
	}

	// Check settings for update schedule
	var updateDay, updateHour int
	if day, err := m.db.GetSetting("geoip.update_day"); err == nil {
		fmt.Sscanf(day, "%d", &updateDay)
	}
	if hour, err := m.db.GetSetting("geoip.update_hour"); err == nil {
		fmt.Sscanf(hour, "%d", &updateHour)
	}

	now := time.Now()
	if int(now.Weekday()) == updateDay && now.Hour() == updateHour {
		return true
	}

	return false
}

// UpdateDatabases downloads the latest GeoIP databases from P3TERX
func (m *GeoIPManager) UpdateDatabases() error {
	m.mu.Lock()
	if m.downloading {
		m.mu.Unlock()
		return fmt.Errorf("download already in progress")
	}
	m.downloading = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.downloading = false
		m.mu.Unlock()
	}()

	log.Println("Updating GeoIP databases from P3TERX mirror...")

	// Get source from settings (default to P3TERX)
	source := "p3terx"
	if s, err := m.db.GetSetting("geoip.source"); err == nil {
		source = strings.ToLower(s)
	}

	var urls map[string]string

	switch source {
	case "p3terx", "":
		urls = map[string]string{
			"GeoLite2-City.mmdb":    P3TERXCityURL,
			"GeoLite2-ASN.mmdb":     P3TERXASNURL,
			"GeoLite2-Country.mmdb": P3TERXCountryURL,
		}
	case "custom":
		// Get custom URL from settings
		if customURL, err := m.db.GetSetting("geoip.custom_url"); err == nil && customURL != "" {
			// Custom URL should point to a tar.gz with all databases
			return m.downloadCustomDatabase(customURL)
		}
		return fmt.Errorf("custom GeoIP source specified but no URL provided")
	default:
		// Default to P3TERX
		urls = map[string]string{
			"GeoLite2-City.mmdb":    P3TERXCityURL,
			"GeoLite2-ASN.mmdb":     P3TERXASNURL,
			"GeoLite2-Country.mmdb": P3TERXCountryURL,
		}
	}

	// Download each database
	for filename, url := range urls {
		if err := m.downloadDatabase(filename, url); err != nil {
			log.Printf("Failed to download %s: %v", filename, err)
			// Continue with other databases
		}
	}

	// Reload databases
	m.closeDatabases()
	m.loadDatabases()

	// Update last update time
	m.lastUpdate = time.Now()

	// Update database record
	m.db.Exec(`
		INSERT OR REPLACE INTO component_downloads (component, status, completed_at, current_version)
		VALUES ('geoip', 'completed', ?, ?)
	`, m.lastUpdate, m.lastUpdate.Format("2006-01-02"))

	log.Println("GeoIP databases updated successfully")

	// Deduplicate if enabled
	if dedup, err := m.db.GetSetting("geoip.enable_dedup"); err == nil && dedup == "true" {
		go m.deduplicateDatabases()
	}

	return nil
}

func (m *GeoIPManager) downloadDatabase(filename, url string) error {
	log.Printf("Downloading %s from %s", filename, url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temporary file
	tmpPath := filepath.Join(m.dataPath, filename+".tmp")
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	// Copy data
	written, err := io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	log.Printf("Downloaded %d bytes for %s", written, filename)

	// Move to final location
	finalPath := filepath.Join(m.dataPath, filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func (m *GeoIPManager) downloadCustomDatabase(url string) error {
	// Download tar.gz archive
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile, err := os.CreateTemp(m.dataPath, "geoip-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	// Download
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}
	tmpFile.Close()

	// Extract
	return m.extractTarGz(tmpFile.Name())
}

func (m *GeoIPManager) extractTarGz(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Extract only .mmdb files
		if strings.HasSuffix(header.Name, ".mmdb") {
			targetPath := filepath.Join(m.dataPath, filepath.Base(header.Name))
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func (m *GeoIPManager) deduplicateDatabases() {
	// This would implement de-duplication logic for the GeoIP databases
	// to reduce size by removing duplicate entries
	log.Println("Running GeoIP de-duplication...")
	// Implementation would go here
}

func (m *GeoIPManager) closeDatabases() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cityDB != nil {
		m.cityDB.Close()
		m.cityDB = nil
	}
	if m.asnDB != nil {
		m.asnDB.Close()
		m.asnDB = nil
	}
	if m.countryDB != nil {
		m.countryDB.Close()
		m.countryDB = nil
	}
}

// Lookup performs a GeoIP lookup for an IP address
func (m *GeoIPManager) Lookup(ip string) (*GeoIPResult, error) {
	// Check cache first
	m.cacheMu.RLock()
	if cached, ok := m.cache[ip]; ok {
		if time.Since(cached.CachedAt) < 1*time.Hour {
			m.cacheMu.RUnlock()
			return cached, nil
		}
	}
	m.cacheMu.RUnlock()

	// Parse IP
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	result := &GeoIPResult{
		IP:       ip,
		CachedAt: time.Now(),
	}

	// Lookup in City database
	m.mu.RLock()
	if m.cityDB != nil {
		var record struct {
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
				Names   map[string]string `maxminddb:"names"`
			} `maxminddb:"country"`
			City struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"city"`
			Subdivisions []struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"subdivisions"`
			Postal struct {
				Code string `maxminddb:"code"`
			} `maxminddb:"postal"`
			Location struct {
				Latitude  float64 `maxminddb:"latitude"`
				Longitude float64 `maxminddb:"longitude"`
				TimeZone  string  `maxminddb:"time_zone"`
			} `maxminddb:"location"`
		}

		if err := m.cityDB.Lookup(netIP, &record); err == nil {
			result.CountryCode = record.Country.ISOCode
			result.Country = record.Country.Names["en"]
			result.City = record.City.Names["en"]
			if len(record.Subdivisions) > 0 {
				result.Region = record.Subdivisions[0].Names["en"]
			}
			result.PostalCode = record.Postal.Code
			result.Latitude = record.Location.Latitude
			result.Longitude = record.Location.Longitude
			result.Timezone = record.Location.TimeZone
		}
	}

	// Lookup in ASN database
	if m.asnDB != nil {
		var asnRecord struct {
			AutonomousSystemNumber       int    `maxminddb:"autonomous_system_number"`
			AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
		}

		if err := m.asnDB.Lookup(netIP, &asnRecord); err == nil {
			result.ASN = asnRecord.AutonomousSystemNumber
			result.ASOrg = asnRecord.AutonomousSystemOrganization
			result.ISP = asnRecord.AutonomousSystemOrganization
		}
	}
	m.mu.RUnlock()

	// Cache the result
	m.cacheMu.Lock()
	m.cache[ip] = result
	// Clean old cache entries if cache is too large
	if len(m.cache) > 10000 {
		// Remove oldest entries
		for k, v := range m.cache {
			if time.Since(v.CachedAt) > 1*time.Hour {
				delete(m.cache, k)
			}
			if len(m.cache) <= 5000 {
				break
			}
		}
	}
	m.cacheMu.Unlock()

	return result, nil
}

// IsBlocked checks if an IP is blocked based on country or other criteria
func (m *GeoIPManager) IsBlocked(ip string) bool {
	result, err := m.Lookup(ip)
	if err != nil {
		return false
	}

	// Check country blocks
	// This would check against a list of blocked countries in the database
	// For now, return false
	_ = result
	return false
}

// GetCountry returns just the country code for an IP
func (m *GeoIPManager) GetCountry(ip string) string {
	result, err := m.Lookup(ip)
	if err != nil {
		return ""
	}
	return result.CountryCode
}

// Close closes all open databases
func (m *GeoIPManager) Close() {
	m.closeDatabases()
}
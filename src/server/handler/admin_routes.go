// Package handler - Admin route validation
// See AI.md PART 17 for admin route hierarchy specification
package handler

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidAdminRootPaths contains the only valid direct children of /{adminpath}/
var ValidAdminRootPaths = map[string]bool{
	"":              true, // Dashboard (/{adminpath}/)
	"profile":       true, // Admin's own profile
	"preferences":   true, // Admin's own preferences
	"notifications": true, // Admin's own notifications
	"server":        true, // Server management (has sub-routes)
}

// ReservedAdminPaths that cannot be used as admin path
var ReservedAdminPaths = []string{
	"api", "static", "assets", "health", "healthz", "version",
	"metrics", ".well-known", "robots.txt", "favicon.ico",
	"auth", "login", "logout", "register",
}

// adminPathRegex validates admin path format
var adminPathRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}[a-z0-9]$|^[a-z0-9]{1,2}$`)

// ValidateAdminRoute checks if an admin route follows the hierarchy
// See AI.md PART 17 - Route Hierarchy Rules
func ValidateAdminRoute(path string) error {
	// Extract first segment after /{adminpath}/
	path = strings.Trim(path, "/")
	if path == "" {
		return nil // Root path is OK
	}

	parts := strings.SplitN(path, "/", 2)
	firstSegment := parts[0]

	if !ValidAdminRootPaths[firstSegment] {
		return fmt.Errorf("invalid admin route: /%s/* - must use /server/* for server management", firstSegment)
	}
	return nil
}

// ValidateAdminPath checks if a path can be used as the admin path
// See AI.md PART 17 - Configurable Admin Path validation
func ValidateAdminPath(newPath string) error {
	// Normalize
	newPath = strings.ToLower(strings.TrimSpace(newPath))
	newPath = strings.Trim(newPath, "/")

	// Check length (2-32 characters)
	if len(newPath) < 2 || len(newPath) > 32 {
		return fmt.Errorf("admin path must be 2-32 characters, got %d", len(newPath))
	}

	// Check format (lowercase alphanumeric and hyphens, no leading/trailing hyphens)
	if !adminPathRegex.MatchString(newPath) {
		return fmt.Errorf("admin path must be lowercase alphanumeric with hyphens, no leading/trailing hyphens")
	}

	// Check reserved paths
	for _, reserved := range ReservedAdminPaths {
		if newPath == reserved {
			return fmt.Errorf("'%s' is a reserved path", newPath)
		}
	}

	return nil
}

// AdminRoutePaths returns all valid admin route patterns
// Used for router registration
func AdminRoutePaths(adminPath string) map[string]string {
	return map[string]string{
		"dashboard":        "/" + adminPath,
		"profile":          "/" + adminPath + "/profile",
		"preferences":      "/" + adminPath + "/preferences",
		"notifications":    "/" + adminPath + "/notifications",
		"server_settings":  "/" + adminPath + "/server/settings",
		"server_ssl":       "/" + adminPath + "/server/ssl",
		"server_email":     "/" + adminPath + "/server/email",
		"server_scheduler": "/" + adminPath + "/server/scheduler",
		"server_logs":      "/" + adminPath + "/server/logs",
		"server_audit":     "/" + adminPath + "/server/logs/audit",
		"server_backup":    "/" + adminPath + "/server/backup",
		"server_updates":   "/" + adminPath + "/server/updates",
		"server_info":      "/" + adminPath + "/server/info",
		"server_metrics":   "/" + adminPath + "/server/metrics",
		"server_users":     "/" + adminPath + "/server/users",
		"server_orgs":      "/" + adminPath + "/server/orgs",
		"server_cluster":   "/" + adminPath + "/server/cluster",
		"server_agents":    "/" + adminPath + "/server/agents",
		"network_tor":      "/" + adminPath + "/server/network/tor",
		"network_geoip":    "/" + adminPath + "/server/network/geoip",
		"security_auth":    "/" + adminPath + "/server/security/auth",
		"security_tokens":  "/" + adminPath + "/server/security/tokens",
		"security_firewall": "/" + adminPath + "/server/security/firewall",
	}
}

// AdminAPIRoutePaths returns all valid admin API route patterns
func AdminAPIRoutePaths(adminPath, apiVersion string) map[string]string {
	base := "/api/" + apiVersion + "/" + adminPath
	return map[string]string{
		"api_profile":         base + "/profile",
		"api_preferences":     base + "/preferences",
		"api_server_settings": base + "/server/settings",
		"api_server_users":    base + "/server/users",
		"api_server_logs":     base + "/server/logs",
		"api_server_backup":   base + "/server/backup",
		"api_server_metrics":  base + "/server/metrics",
	}
}

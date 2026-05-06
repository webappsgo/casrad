// Package handler - Content negotiation per AI.md PART 14
package handler

import (
	"net/http"
	"strings"
)

// isCliTool detects CLI tools that prefer plain text
// Per AI.md PART 14: CLI Tool Detection
func isCliTool(r *http.Request) bool {
	ua := r.Header.Get("User-Agent")

	// Our own CLI client
	if strings.HasPrefix(ua, "casrad-cli/") {
		return true
	}

	// Common CLI tools per PART 14
	cliTools := []string{
		"curl/", "wget/", "httpie/", "HTTPie/",
		"Wget/", "libcurl/", "python-requests/",
		"Go-http-client/", "axios/", "node-fetch/",
	}
	for _, tool := range cliTools {
		if strings.Contains(ua, tool) {
			return true
		}
	}

	// No User-Agent or empty = likely CLI
	if ua == "" {
		return true
	}

	return false
}

// getAPIResponseFormat determines format for /api/** routes
// Per AI.md PART 14: Backend API Content Negotiation
func getAPIResponseFormat(r *http.Request) string {
	// 1. Check .txt extension
	if strings.HasSuffix(r.URL.Path, ".txt") {
		return "text"
	}

	// 2. Check Accept header
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/plain") {
		return "text"
	}

	// 3. Check if CLI tool
	if isCliTool(r) {
		return "text"
	}

	// 4. Default to JSON
	return "json"
}

// detectResponseFormat determines response format based on request
// Per AI.md PART 14: Content Negotiation Implementation
func detectResponseFormat(r *http.Request) string {
	// 1. Check for .txt extension
	if strings.HasSuffix(r.URL.Path, ".txt") {
		return "text/plain"
	}

	// 2. Check Accept header
	accept := r.Header.Get("Accept")

	switch {
	case strings.Contains(accept, "application/json"):
		return "application/json"
	case strings.Contains(accept, "text/plain"):
		return "text/plain"
	case strings.Contains(accept, "text/html"):
		return "text/html"
	default:
		// 3. Default based on endpoint
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return "application/json"
		}
		return "text/html"
	}
}

// getFrontendResponseFormat determines format for frontend routes
// Per AI.md PART 14: Smart Content Negotiation for frontend routes
func getFrontendResponseFormat(r *http.Request) string {
	// Priority 1: Accept: text/html header
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		return "html"
	}

	// Priority 2: Accept: text/plain header
	if strings.Contains(accept, "text/plain") {
		return "text"
	}

	// Priority 3: Browser detection via User-Agent
	ua := r.Header.Get("User-Agent")
	browserIndicators := []string{
		"Mozilla/", "Chrome/", "Safari/", "Firefox/", "Edge/",
		"Opera/", "MSIE", "Trident/",
	}
	for _, indicator := range browserIndicators {
		if strings.Contains(ua, indicator) {
			return "html"
		}
	}

	// Priority 4: CLI/curl (no browser UA)
	if isCliTool(r) {
		return "text"
	}

	// Priority 5: Default to HTML
	return "html"
}

// stripTxtExtension removes .txt extension from path if present
// Used for routing when .txt is used for plain text output
func stripTxtExtension(path string) string {
	if strings.HasSuffix(path, ".txt") {
		return strings.TrimSuffix(path, ".txt")
	}
	return path
}

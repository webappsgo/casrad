// Package handler - Reserved names for routing
// See AI.md PART 16 for reserved names specification
package handler

// ReservedNames contains names that cannot be used as usernames or org slugs
// These are blocked from registration to prevent route conflicts
var ReservedNames = []string{
	// System routes
	"api", "admin", "static", "assets", "healthz", "metrics",
	"login", "logout", "register", "signup", "signin", "auth",
	"oauth", "callback", "webhook", "webhooks",

	// Common paths
	"users", "orgs", "organizations", "teams", "groups",
	"settings", "profile", "account", "dashboard",
	"search", "explore", "discover", "trending",
	"help", "support", "docs", "documentation",
	"about", "contact", "terms", "privacy", "legal",

	// Technical
	"graphql", "rest", "rpc", "ws", "websocket",
	"cdn", "media", "uploads", "files", "images",
	".well-known", "robots.txt", "sitemap.xml", "favicon.ico",

	// CASRAD-specific (PART 37)
	"stream", "streams", "radio", "broadcast", "broadcasts",
	"playlist", "playlists", "track", "tracks", "album", "albums",
	"artist", "artists", "podcast", "podcasts", "audiobook", "audiobooks",
	"library", "queue", "nowplaying", "player",
}

// IsReservedName checks if a name is reserved
func IsReservedName(name string) bool {
	for _, reserved := range ReservedNames {
		if name == reserved {
			return true
		}
	}
	return false
}

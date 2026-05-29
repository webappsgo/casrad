// Package locales embeds all translation JSON files per AI.md PART 7 and PART 31.
// The embed.FS is exported so the server and CLI can load translations at runtime
// without reading from disk.
package locales

import "embed"

// FS contains all locale JSON files (en, es, zh, fr, ar, de, ja).
//
//go:embed *.json
var FS embed.FS

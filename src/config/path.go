// Package config - Path normalization and validation utilities
// See AI.md PART 5 for path security specification
package config

import (
	"errors"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrPathTraversal = errors.New("path traversal attempt detected")
	ErrInvalidPath   = errors.New("invalid path characters")
	ErrPathTooLong   = errors.New("path exceeds maximum length")

	// Valid path segment: lowercase alphanumeric, hyphens, underscores
	validPathSegment = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

// NormalizePath cleans a path for safe use
// - Strips leading/trailing slashes
// - Collapses multiple slashes (// → /)
// - Removes path traversal (.., .)
// - Returns empty string for invalid input
func NormalizePath(input string) string {
	if input == "" {
		return ""
	}

	// Use path.Clean to handle .., ., and //
	cleaned := path.Clean(input)

	// Strip leading/trailing slashes
	cleaned = strings.Trim(cleaned, "/")

	// Reject if still contains .. after cleaning (shouldn't happen, but be safe)
	if strings.Contains(cleaned, "..") {
		return ""
	}

	return cleaned
}

// ValidatePathSegment checks a single path segment (e.g., "admin" in "/admin/dashboard")
func ValidatePathSegment(segment string) error {
	if segment == "" {
		return ErrInvalidPath
	}
	if len(segment) > 64 {
		return ErrPathTooLong
	}
	if !validPathSegment.MatchString(segment) {
		return ErrInvalidPath
	}
	if segment == "." || segment == ".." {
		return ErrPathTraversal
	}
	return nil
}

// ValidatePath checks an entire path
func ValidatePath(p string) error {
	if len(p) > 2048 {
		return ErrPathTooLong
	}

	// Check for traversal attempts before normalization
	if strings.Contains(p, "..") {
		return ErrPathTraversal
	}

	// Check each segment
	segments := strings.Split(strings.Trim(p, "/"), "/")
	for _, seg := range segments {
		if seg == "" {
			// Skip empty (from //)
			continue
		}
		if err := ValidatePathSegment(seg); err != nil {
			return err
		}
	}

	return nil
}

// SafePath normalizes and validates - returns error if invalid
func SafePath(input string) (string, error) {
	if err := ValidatePath(input); err != nil {
		return "", err
	}
	return NormalizePath(input), nil
}

// SafeFilePath ensures path stays within base directory
func SafeFilePath(baseDir, userPath string) (string, error) {
	// Normalize user input
	safe, err := SafePath(userPath)
	if err != nil {
		return "", err
	}

	// Construct full path
	fullPath := filepath.Join(baseDir, safe)

	// Resolve to absolute
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	// Verify path is still within base
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", ErrPathTraversal
	}

	return absPath, nil
}

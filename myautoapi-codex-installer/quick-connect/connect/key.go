package connect

import (
	"regexp"
	"strings"
)

// MyAutoAPI keys are "sk-" followed by the characters the site allows in
// generated and custom keys.
var keyPattern = regexp.MustCompile(`^sk-[A-Za-z0-9_-]+$`)

// NormalizeKey trims surrounding whitespace and reports whether the result is
// a well-formed MyAutoAPI key.
func NormalizeKey(raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	return key, keyPattern.MatchString(key)
}

// MaskKey keeps the first 12 and last 8 characters, matching the website.
// Short keys keep less so that the middle is never fully shown.
func MaskKey(key string) string {
	switch {
	case len(key) >= 24:
		return key[:12] + "****" + key[len(key)-8:]
	case len(key) >= 8:
		return key[:3] + "****" + key[len(key)-4:]
	default:
		return "sk-****"
	}
}

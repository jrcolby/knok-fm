package urldetector

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL creates a canonical form of a URL for storage and deduplication.
// It handles:
// - Adding https:// protocol if missing
// - Lowercasing the domain
// - Removing www. prefix
// - Removing tracking parameters (utm_*, si, fbclid, ref, source)
// - Validating the URL structure
func NormalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Step 1: Add protocol if missing
	if !strings.HasPrefix(strings.ToLower(rawURL), "http://") &&
		!strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		// Check if it looks like a domain (has at least one dot)
		if strings.Contains(rawURL, ".") {
			rawURL = "https://" + rawURL
		} else {
			return "", fmt.Errorf("invalid URL: no domain found")
		}
	}

	// Step 2: Parse URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Step 3: Validate URL has a host
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL: no host found")
	}

	// Step 4: Normalize domain (lowercase, remove www.)
	u.Host = strings.ToLower(u.Host)
	u.Host = strings.TrimPrefix(u.Host, "www.")

	// Step 5: Remove tracking parameters
	q := u.Query()
	trackingParams := []string{
		// Google Analytics
		"utm_source",
		"utm_medium",
		"utm_campaign",
		"utm_content",
		"utm_term",
		// Platform-specific tracking
		"si",      // Spotify/YouTube share ID
		"fbclid",  // Facebook click ID
		"gclid",   // Google click ID
		"ref",     // Generic referrer
		"source",  // Generic source
		// Additional tracking params
		"msclkid", // Microsoft click ID
		"igshid",  // Instagram share ID
	}

	for _, param := range trackingParams {
		q.Del(param)
	}
	u.RawQuery = q.Encode()

	// Step 6: Rebuild canonical URL
	return u.String(), nil
}

// cleanTrailingPunctuation removes trailing punctuation from a URL intelligently.
// It preserves closing parentheses if they're balanced (for Wikipedia-style URLs).
func cleanTrailingPunctuation(urlStr string) string {
	// Check for balanced parentheses (Wikipedia-style URLs)
	if strings.Contains(urlStr, "(") && strings.HasSuffix(urlStr, ")") {
		openCount := strings.Count(urlStr, "(")
		closeCount := strings.Count(urlStr, ")")
		// If parentheses are balanced or more open than close, keep the closing paren
		if openCount >= closeCount {
			return urlStr
		}
	}

	// Remove common sentence-ending punctuation
	return strings.TrimRight(urlStr, ".,!?;:\"'")
}

// GetCanonicalURL creates a canonical URL from an already-parsed URL object.
// This is useful when you already have a *url.URL and want to normalize it.
func GetCanonicalURL(u *url.URL) string {
	// Clone the URL to avoid modifying the original
	canonical := *u

	// Normalize host (lowercase, remove www.)
	canonical.Host = strings.ToLower(canonical.Host)
	canonical.Host = strings.TrimPrefix(canonical.Host, "www.")

	// Remove tracking parameters
	q := canonical.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
		"si", "fbclid", "gclid", "ref", "source", "msclkid", "igshid",
	}

	for _, param := range trackingParams {
		q.Del(param)
	}
	canonical.RawQuery = q.Encode()

	return canonical.String()
}

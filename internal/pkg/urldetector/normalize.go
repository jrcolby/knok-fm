package urldetector

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// NormalizeURL creates a canonical form of a URL for storage and deduplication.
// It handles:
// - Adding https:// protocol if missing
// - Lowercasing the domain (keeps www. as posted)
// - Removing tracking parameters (utm_*, si, fbclid, ref, source)
// - Validating the URL structure
func NormalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Step 1: Add protocol if missing (required for url.Parse to work correctly)
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

	// Step 4: Normalize domain (lowercase only - keep www. as posted)
	u.Host = strings.ToLower(u.Host)

	// Step 5: Remove tracking parameters
	// NOTE: u.Query() only parses properly-formatted query strings (after the first ?)
	// It will NOT parse encoded query params like %3Fsi%3D - this is intentional!
	// We want to preserve the URL as the user posted it, just remove actual tracking params
	q := u.Query()
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

// trackingParams is the consolidated list of tracking parameters to strip.
var trackingParams = []string{
	// Google Analytics
	"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
	// Platform-specific tracking
	"si",      // Spotify/YouTube share ID
	"fbclid",  // Facebook click ID
	"gclid",   // Google click ID
	"ref",     // Generic referrer
	"source",  // Generic source
	"msclkid", // Microsoft click ID
	"igshid",  // Instagram share ID
	// Additional tracking params (Spotify, YouTube, etc.)
	"pt",         // Spotify tracking
	"pi",         // Spotify tracking
	"nd",         // Spotify tracking
	"feature",    // YouTube feature tracking
	"pp",         // YouTube playlist tracking
	"ab_channel", // YouTube A/B test channel
}

// mobileDomains maps mobile domains to their canonical equivalents.
var mobileDomains = map[string]string{
	"m.youtube.com":    "youtube.com",
	"m.soundcloud.com": "soundcloud.com",
}

// youtubeVideoIDRegex matches YouTube video IDs (11 alphanumeric + - + _).
var youtubeVideoIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// spotifyIDRegex matches Spotify IDs (22 alphanumeric).
var spotifyIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]{22}$`)

// spotifyTypes are valid Spotify resource types.
var spotifyTypes = map[string]bool{
	"track":    true,
	"album":    true,
	"playlist": true,
	"artist":   true,
	"episode":  true,
	"show":     true,
}

// CanonicalizeURL produces a canonical URL for dedup comparison.
// It calls NormalizeURL first, then applies:
// - Strip www.
// - Normalize mobile domains (m.youtube.com → youtube.com)
// - Remove fragments
// - Sort query params
// - Strip expanded tracking params
// - Platform-specific ID extraction (YouTube, Spotify)
func CanonicalizeURL(rawURL string) (string, error) {
	// Start with existing normalization
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to parse normalized URL: %w", err)
	}

	// Strip www.
	host := u.Host
	host = strings.TrimPrefix(host, "www.")

	// Normalize mobile domains
	if canonical, ok := mobileDomains[host]; ok {
		host = canonical
	}
	u.Host = host

	// Remove fragments
	u.Fragment = ""
	u.RawFragment = ""

	// Strip tracking params and sort remaining
	q := u.Query()
	for _, param := range trackingParams {
		q.Del(param)
	}

	// Sort query params for deterministic form
	sortedQuery := sortQueryValues(q)
	u.RawQuery = sortedQuery

	// Platform-specific ID extraction
	canonical := tryExtractPlatformCanonical(u)
	if canonical != "" {
		return canonical, nil
	}

	return u.String(), nil
}

// sortQueryValues produces a sorted, encoded query string.
func sortQueryValues(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vs := q[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// tryExtractPlatformCanonical returns a canonical URL for YouTube or Spotify,
// or empty string if not applicable.
func tryExtractPlatformCanonical(u *url.URL) string {
	host := u.Host

	// YouTube: extract 11-char video ID → https://youtube.com/watch?v=ID
	if host == "youtube.com" || host == "music.youtube.com" {
		videoID := extractYouTubeVideoID(u)
		if videoID != "" {
			return "https://youtube.com/watch?v=" + videoID
		}
	}
	if host == "youtu.be" {
		// youtu.be/VIDEO_ID
		id := strings.TrimPrefix(u.Path, "/")
		if youtubeVideoIDRegex.MatchString(id) {
			return "https://youtube.com/watch?v=" + id
		}
	}

	// Spotify: extract type + 22-char ID → https://open.spotify.com/{type}/{id}
	if host == "open.spotify.com" || host == "spotify.com" {
		spotifyType, spotifyID := extractSpotifyID(u)
		if spotifyType != "" && spotifyID != "" {
			return "https://open.spotify.com/" + spotifyType + "/" + spotifyID
		}
	}

	return ""
}

// extractYouTubeVideoID extracts the video ID from a YouTube URL.
func extractYouTubeVideoID(u *url.URL) string {
	// /watch?v=VIDEO_ID
	if v := u.Query().Get("v"); youtubeVideoIDRegex.MatchString(v) {
		return v
	}
	// /shorts/VIDEO_ID or /embed/VIDEO_ID or /v/VIDEO_ID
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) == 2 {
		prefix := parts[0]
		id := parts[1]
		if (prefix == "shorts" || prefix == "embed" || prefix == "v" || prefix == "live") && youtubeVideoIDRegex.MatchString(id) {
			return id
		}
	}
	return ""
}

// extractSpotifyID extracts the resource type and ID from a Spotify URL.
// Expects paths like /track/22charID or /intl-xx/track/22charID
func extractSpotifyID(u *url.URL) (string, string) {
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	// Skip optional intl prefix like "intl-us"
	if len(parts) >= 3 && strings.HasPrefix(parts[0], "intl-") {
		parts = parts[1:]
	}

	if len(parts) >= 2 {
		resourceType := parts[0]
		id := parts[1]
		if spotifyTypes[resourceType] && spotifyIDRegex.MatchString(id) {
			return resourceType, id
		}
	}
	return "", ""
}

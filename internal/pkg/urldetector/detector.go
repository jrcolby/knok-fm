package urldetector

import (
	"knock-fm/internal/domain"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// URLInfo contains information about a detected URL
type URLInfo struct {
	URL      string
	Platform string
}

// PlatformLoader defines the interface for loading platform configurations
type PlatformLoader interface {
	GetAllByPriority() ([]*domain.Platform, error)
	IsLoaded() bool
}

// Detector provides centralized URL detection using platform loader
type Detector struct {
	loader   PlatformLoader
	logger   *slog.Logger
	patterns []compiledPattern
	mu       sync.RWMutex
}

type compiledPattern struct {
	regex    *regexp.Regexp
	platform string
}

// New creates a new URL detector using the platform loader
func New(loader PlatformLoader, logger *slog.Logger) *Detector {
	detector := &Detector{
		loader: loader,
		logger: logger,
	}
	detector.buildPatterns()
	return detector
}

// buildPatterns generates optimized regex patterns from platform loader
func (d *Detector) buildPatterns() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if loader has loaded platforms
	if !d.loader.IsLoaded() {
		d.logger.Warn("Platform loader not ready, patterns not built yet")
		d.patterns = make([]compiledPattern, 0)
		return
	}

	// Get platforms sorted by priority (highest first)
	platforms, err := d.loader.GetAllByPriority()
	if err != nil {
		d.logger.Error("Failed to get platforms from loader", "error", err)
		d.patterns = make([]compiledPattern, 0)
		return
	}

	d.patterns = make([]compiledPattern, 0)

	// Build patterns for each platform (respecting priority order)
	for _, platform := range platforms {
		for _, urlPattern := range platform.URLPatterns {
			// Build comprehensive regex pattern that handles:
			// - Optional protocols (http/https)
			// - Optional www subdomain
			// - Exact domain matching
			// - Word boundaries to avoid false positives
			regexPattern := d.buildRegexPattern(urlPattern)

			if compiled, err := regexp.Compile(regexPattern); err == nil {
				d.patterns = append(d.patterns, compiledPattern{
					regex:    compiled,
					platform: platform.ID,
				})
			}
		}
	}

	d.logger.Info("Built URL detection patterns",
		"platform_count", len(platforms),
		"pattern_count", len(d.patterns),
	)
}

// buildRegexPattern creates an optimized regex from a simple URL pattern
func (d *Detector) buildRegexPattern(urlPattern string) string {
	// Handle special cases for different patterns
	switch {
	case strings.HasPrefix(urlPattern, "www."):
		// For www.example.com patterns, make www optional
		domain := strings.TrimPrefix(urlPattern, "www.")
		return `(?i)(?:https?://)?(?:www\.)?` + regexp.QuoteMeta(domain) + `\b`
	
	case strings.Contains(urlPattern, "open."):
		// For Spotify-like patterns (open.spotify.com)
		return `(?i)(?:https?://)?(?:open\.)?` + regexp.QuoteMeta(strings.TrimPrefix(urlPattern, "open.")) + `\b`
	
	case strings.Contains(urlPattern, "music."):
		// For Apple Music-like patterns (music.apple.com)
		return `(?i)(?:https?://)?` + regexp.QuoteMeta(urlPattern) + `\b`
	
	case strings.Contains(urlPattern, "bandcamp.com"):
		// For Bandcamp subdomains (artist.bandcamp.com)
		return `(?i)(?:https?://)?[\w-]+\.` + regexp.QuoteMeta(urlPattern) + `\b`
	
	default:
		// Standard pattern with optional protocol and www
		return `(?i)(?:https?://)?(?:www\.)?` + regexp.QuoteMeta(urlPattern) + `\b`
	}
}

// DetectURLs finds all supported music URLs in content and returns them with platform info.
// Uses a multi-stage approach to handle various URL formats:
// - Stage 1: Markdown links [text](url)
// - Stage 2: Discord suppressed embeds <https://...>
// - Stage 3: Standard URLs (http/https with full pattern matching)
// - Stage 4: Plain domains (no protocol, known platforms only)
// - Stage 5: Multi-line URLs (Discord wrapping)
// All mobile/short link patterns are now handled by platform URLPatterns from the database.
func (d *Detector) DetectURLs(content string) []URLInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var urls []URLInfo
	seen := make(map[string]bool)

	// Stage 1: Extract Markdown Links [text](url)
	markdownRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	for _, match := range markdownRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 2 {
			d.addIfSupported(match[2], &urls, seen)
		}
	}

	// Remove markdown from content to avoid duplicate detection
	cleanContent := markdownRegex.ReplaceAllString(content, " ")

	// Stage 2: Extract Discord Suppressed Embeds <https://...>
	suppressedRegex := regexp.MustCompile(`<(https?://[^>]+)>`)
	cleanContent = suppressedRegex.ReplaceAllStringFunc(cleanContent, func(s string) string {
		url := s[1 : len(s)-1] // Remove angle brackets
		d.addIfSupported(url, &urls, seen)
		return " " // Replace with space to avoid re-detection
	})

	// Stage 3: Extract Standard URLs (comprehensive pattern)
	// Matches http(s):// URLs with domain, path, query, fragment
	urlRegex := regexp.MustCompile(
		`(?i)https?://[\w\-]+(?:\.[\w\-]+)+(?:/[^\s<>\[\]()]*)?(?:\?[^\s<>\[\]()]*)?(?:#[^\s<>\[\]()]*)?`,
	)

	for _, match := range urlRegex.FindAllString(cleanContent, -1) {
		cleaned := cleanTrailingPunctuation(match)
		d.addIfSupported(cleaned, &urls, seen)
	}

	// Stage 4: Extract Plain Domains (no protocol)
	// Common case: "spotify.com/track/..." or "www.youtube.com/watch?v=..."
	// Only match known music platform domains to avoid false positives
	domainRegex := regexp.MustCompile(
		`(?i)(?:^|\s)((?:www\.)?(?:spotify|youtube|youtu|soundcloud|bandcamp|mixcloud|tidal|deezer|apple)\.(?:com|be|fm|live)(?:/[^\s<>\[\]()]*)?)`,
	)

	for _, match := range domainRegex.FindAllStringSubmatch(cleanContent, -1) {
		if len(match) > 1 {
			cleaned := cleanTrailingPunctuation(match[1])
			d.addIfSupported(cleaned, &urls, seen)
		}
	}

	// Stage 5: Handle Multi-line URLs (Discord wrapping)
	// Discord sometimes wraps long URLs across lines
	if strings.Contains(content, "\n") {
		unwrappedContent := strings.ReplaceAll(content, "\n", "")
		// Re-run URL detection on unwrapped content (just the generic pattern)
		for _, match := range urlRegex.FindAllString(unwrappedContent, -1) {
			cleaned := cleanTrailingPunctuation(match)
			d.addIfSupported(cleaned, &urls, seen)
		}
	}

	return urls
}

// addIfSupported normalizes a URL, detects its platform, and adds it to the results if valid.
// This helper prevents duplicates and ensures all URLs are properly normalized.
func (d *Detector) addIfSupported(rawURL string, urls *[]URLInfo, seen map[string]bool) {
	// URL-decode first to handle Discord's double-encoded URLs
	// Discord sometimes sends URLs like: https://youtube.com/watch?v=ABC%3Fsi%3DXYZ
	// which should be: https://youtube.com/watch?v=ABC&si=XYZ
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		// If decoding fails, use the original URL
		decodedURL = rawURL
	}

	// Fix malformed query strings: after the first ?, any subsequent ? should be &
	// This handles Discord's encoding where &si=123 becomes %3Fsi%3D123
	decodedURL = fixMalformedQueryString(decodedURL)

	// Log the URL decoding if it changed the URL
	if decodedURL != rawURL {
		d.logger.Info("URL decoded and fixed",
			"raw_url", rawURL,
			"fixed_url", decodedURL)
	}

	// Normalize the URL for canonical form and deduplication
	normalizedURL, err := NormalizeURL(decodedURL)
	if err != nil {
		// Invalid URL, skip it
		return
	}

	// Check for duplicates using normalized form
	if seen[normalizedURL] {
		return
	}

	// Detect platform using normalized URL
	platform := d.detectPlatformFromURL(normalizedURL)

	// Add to results (including unknown platforms)
	seen[normalizedURL] = true
	*urls = append(*urls, URLInfo{
		URL:      normalizedURL,
		Platform: platform,
	})
}

// fixMalformedQueryString fixes URLs where subsequent ? should be &
// Example: "https://youtube.com/watch?v=ABC?si=XYZ" -> "https://youtube.com/watch?v=ABC&si=XYZ"
func fixMalformedQueryString(rawURL string) string {
	// Find the first ? (start of query string)
	firstQ := strings.Index(rawURL, "?")
	if firstQ == -1 {
		// No query string, return as-is
		return rawURL
	}

	// Split into base and query parts
	base := rawURL[:firstQ+1] // Include the first ?
	queryPart := rawURL[firstQ+1:]

	// Replace any remaining ? with &
	queryPart = strings.ReplaceAll(queryPart, "?", "&")

	return base + queryPart
}

// detectPlatformFromURL detects platform using database-loaded patterns
// Note: This is called from addIfSupported which already holds a read lock,
// so we don't acquire another lock here to avoid deadlock
func (d *Detector) detectPlatformFromURL(url string) string {
	// Use the compiled patterns from the database (respects priority)
	for _, pattern := range d.patterns {
		if pattern.regex.MatchString(url) {
			return pattern.platform
		}
	}

	// No match found - return unknown
	return domain.PlatformUnknown
}

// IsSupported checks if a URL matches any supported platform pattern
func (d *Detector) IsSupported(url string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, pattern := range d.patterns {
		if pattern.regex.MatchString(url) {
			return true
		}
	}
	return false
}

// GetSupportedPlatforms returns a list of all supported platform IDs from the loader
func (d *Detector) GetSupportedPlatforms() []string {
	platforms, err := d.loader.GetAllByPriority()
	if err != nil {
		d.logger.Warn("Failed to get platforms from loader", "error", err)
		return []string{}
	}

	platformIDs := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		platformIDs = append(platformIDs, platform.ID)
	}

	return platformIDs
}

// Refresh rebuilds patterns from the current domain configuration
// Useful if platform config is updated at runtime
func (d *Detector) Refresh() {
	d.buildPatterns()
}
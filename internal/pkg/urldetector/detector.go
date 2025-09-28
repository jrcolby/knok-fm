package urldetector

import (
	"knock-fm/internal/domain"
	"regexp"
	"strings"
	"sync"
)

// URLInfo contains information about a detected URL
type URLInfo struct {
	URL      string
	Platform string
}

// Detector provides centralized URL detection using domain platform config
type Detector struct {
	patterns []compiledPattern
	mu       sync.RWMutex
}

type compiledPattern struct {
	regex    *regexp.Regexp
	platform string
}

// New creates a new URL detector using the centralized platform configuration
func New() *Detector {
	detector := &Detector{}
	detector.buildPatterns()
	return detector
}

// buildPatterns generates optimized regex patterns from domain platform config
func (d *Detector) buildPatterns() {
	d.mu.Lock()
	defer d.mu.Unlock()

	config := domain.GetPlatformConfig()
	d.patterns = make([]compiledPattern, 0)

	for platformID, platform := range config.Platforms {
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
					platform: platformID,
				})
			}
		}
	}
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

// DetectURLs finds all supported music URLs in content and returns them with platform info
func (d *Detector) DetectURLs(content string) []URLInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var urls []URLInfo
	seen := make(map[string]bool)

	// Split content into words and check each for URL patterns
	words := strings.Fields(content)
	for _, word := range words {
		for _, pattern := range d.patterns {
			if pattern.regex.MatchString(word) {
				// Clean up the URL (remove any trailing punctuation)
				url := strings.TrimRight(word, ".,!?;:")

				// Avoid duplicates
				if !seen[url] {
					seen[url] = true
					urls = append(urls, URLInfo{
						URL:      url,
						Platform: d.detectPlatformFromURL(url),
					})
				}
				break
			}
		}
	}

	return urls
}

// detectPlatformFromURL uses the domain's centralized platform detection
func (d *Detector) detectPlatformFromURL(url string) string {
	return domain.DetectPlatformFromURL(url)
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

// GetSupportedPlatforms returns a list of all supported platform IDs
func (d *Detector) GetSupportedPlatforms() []string {
	return domain.GetValidPlatforms()
}

// Refresh rebuilds patterns from the current domain configuration
// Useful if platform config is updated at runtime
func (d *Detector) Refresh() {
	d.buildPatterns()
}
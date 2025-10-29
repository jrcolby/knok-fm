package worker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed oembed_providers.json
var oembedProvidersJSON []byte

// OEmbedProvider represents an oEmbed provider with its endpoint and URL patterns
type OEmbedProvider struct {
	Name     string
	Endpoint string
	Schemes  []*regexp.Regexp // Compiled regex patterns for URL matching
}

// OEmbedRegistry manages oEmbed providers and matches URLs to providers
type OEmbedRegistry struct {
	providers []*OEmbedProvider
}

// rawProvider matches the JSON structure from oembed.com/providers.json
type rawProvider struct {
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Endpoints    []struct {
		Schemes []string `json:"schemes"`
		URL     string   `json:"url"`
	} `json:"endpoints"`
}

// NewOEmbedRegistry creates and initializes a new oEmbed registry
func NewOEmbedRegistry() (*OEmbedRegistry, error) {
	var rawProviders []rawProvider
	if err := json.Unmarshal(oembedProvidersJSON, &rawProviders); err != nil {
		return nil, fmt.Errorf("failed to parse oEmbed providers: %w", err)
	}

	registry := &OEmbedRegistry{
		providers: make([]*OEmbedProvider, 0, len(rawProviders)),
	}

	// Parse and compile patterns for each provider
	for _, raw := range rawProviders {
		if len(raw.Endpoints) == 0 {
			continue
		}

		// Use first endpoint (most providers have only one)
		endpoint := raw.Endpoints[0]
		if endpoint.URL == "" {
			continue
		}

		provider := &OEmbedProvider{
			Name:     raw.ProviderName,
			Endpoint: endpoint.URL,
			Schemes:  make([]*regexp.Regexp, 0, len(endpoint.Schemes)),
		}

		// KLUDGE: Add custom schemes for YouTube to handle double-encoded share URLs
		// Discord users share URLs like https://youtube.com/watch?v=ABC%3Fsi%3DXYZ
		// which don't match YouTube's official oEmbed schemes (which require subdomains)
		schemes := endpoint.Schemes
		if raw.ProviderName == "YouTube" {
			schemes = append(schemes, "https://youtube.com/watch*")
		}

		// Compile URL patterns into regexes
		for _, scheme := range schemes {
			pattern := schemeToRegex(scheme)
			regex, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid patterns
				continue
			}
			provider.Schemes = append(provider.Schemes, regex)
		}

		// Only add provider if it has at least one valid scheme
		if len(provider.Schemes) > 0 {
			registry.providers = append(registry.providers, provider)
		}
	}

	return registry, nil
}

// Match finds an oEmbed provider for the given URL
// Returns nil if no provider matches
func (r *OEmbedRegistry) Match(url string) *OEmbedProvider {
	for _, provider := range r.providers {
		for _, pattern := range provider.Schemes {
			if pattern.MatchString(url) {
				return provider
			}
		}
	}
	return nil
}

// schemeToRegex converts an oEmbed URL scheme pattern to a regex pattern
// Examples:
//   - "https://*.youtube.com/watch*" → "^https://[^/]*\\.youtube\\.com/watch.*$"
//   - "https://open.spotify.com/*" → "^https://open\\.spotify\\.com/.*$"
func schemeToRegex(scheme string) string {
	// Escape special regex characters except * and ?
	pattern := regexp.QuoteMeta(scheme)

	// Replace escaped wildcards with regex equivalents
	// \* (escaped by QuoteMeta) → .* (match any characters)
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")

	// \? (escaped by QuoteMeta) → . (match single character)
	pattern = strings.ReplaceAll(pattern, "\\?", ".")

	// Anchor the pattern to match the entire URL
	pattern = "^" + pattern + "$"

	return pattern
}

// GetProviderCount returns the total number of registered providers
func (r *OEmbedRegistry) GetProviderCount() int {
	return len(r.providers)
}

// GetProvider returns a provider by name (case-sensitive)
// Useful for testing or debugging
func (r *OEmbedRegistry) GetProvider(name string) *OEmbedProvider {
	for _, provider := range r.providers {
		if provider.Name == name {
			return provider
		}
	}
	return nil
}

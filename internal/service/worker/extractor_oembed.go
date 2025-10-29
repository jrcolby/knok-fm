package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OEmbedExtractor extracts metadata using oEmbed APIs
type OEmbedExtractor struct {
	registry   *OEmbedRegistry
	logger     *slog.Logger
	httpClient *http.Client
}

// oEmbedResponse represents the standard oEmbed JSON response
// See: https://oembed.com/#section2.3
type oEmbedResponse struct {
	Type            string `json:"type"`             // "video", "photo", "link", "rich"
	Version         string `json:"version"`          // oEmbed version (should be "1.0")
	Title           string `json:"title"`            // Title of the resource
	AuthorName      string `json:"author_name"`      // Author/creator name
	AuthorURL       string `json:"author_url"`       // Author/creator URL
	ProviderName    string `json:"provider_name"`    // Provider name (e.g., "YouTube")
	ProviderURL     string `json:"provider_url"`     // Provider URL
	ThumbnailURL    string `json:"thumbnail_url"`    // Thumbnail image URL
	ThumbnailWidth  int    `json:"thumbnail_width"`  // Thumbnail width
	ThumbnailHeight int    `json:"thumbnail_height"` // Thumbnail height
	HTML            string `json:"html"`             // Embed HTML (for video/rich types)
	Width           int    `json:"width"`            // Resource width
	Height          int    `json:"height"`           // Resource height
	Description     string `json:"description"`      // Description (not in spec, but some providers include it)
}

// NewOEmbedExtractor creates a new oEmbed metadata extractor
func NewOEmbedExtractor(registry *OEmbedRegistry, logger *slog.Logger) *OEmbedExtractor {
	return &OEmbedExtractor{
		registry: registry,
		logger:   logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// normalizeForOEmbed normalizes URLs to match oEmbed provider patterns
// Currently just a passthrough, but kept for future normalization needs
func normalizeForOEmbed(rawURL string) string {
	// Could add URL normalization logic here if needed in the future
	return rawURL
}

// TryExtract attempts to extract metadata using oEmbed
// Returns nil metadata and nil error if no oEmbed provider is found (not an error, just skip)
// Returns error only if provider exists but extraction failed
func (e *OEmbedExtractor) TryExtract(ctx context.Context, resourceURL string) (map[string]string, error) {
	// Normalize URL for oEmbed pattern matching
	normalizedURL := normalizeForOEmbed(resourceURL)

	// Check if we have an oEmbed provider for this URL
	provider := e.registry.Match(normalizedURL)
	if provider == nil {
		// No provider found - this is not an error, just means oEmbed doesn't support this URL
		return nil, nil
	}

	e.logger.Info("oEmbed provider found",
		"provider", provider.Name,
		"url", resourceURL,
		"normalized_url", normalizedURL,
		"endpoint", provider.Endpoint)

	// Build oEmbed API URL using normalized URL
	oembedURL, err := e.buildOEmbedURL(provider.Endpoint, normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build oEmbed URL: %w", err)
	}

	// Fetch oEmbed data
	oembedData, err := e.fetchOEmbed(ctx, oembedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch oEmbed data from %s: %w", provider.Name, err)
	}

	// Convert oEmbed response to our metadata format
	metadata := e.oembedToMetadata(oembedData, resourceURL)

	e.logger.Info("oEmbed extraction successful",
		"provider", provider.Name,
		"url", resourceURL,
		"title", metadata["title"],
		"has_image", metadata["image"] != "")

	return metadata, nil
}

// buildOEmbedURL constructs the oEmbed API URL with proper parameters
func (e *OEmbedExtractor) buildOEmbedURL(endpoint, resourceURL string) (string, error) {
	// Some endpoints have placeholders like {format}
	// Replace {format} with "json"
	endpoint = strings.ReplaceAll(endpoint, "{format}", "json")

	// Parse endpoint URL
	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Add query parameters
	query := baseURL.Query()
	query.Set("url", resourceURL)
	query.Set("format", "json")
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}

// fetchOEmbed makes the HTTP request to the oEmbed endpoint
func (e *OEmbedExtractor) fetchOEmbed(ctx context.Context, oembedURL string) (*oEmbedResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", oembedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a realistic User-Agent (some providers check this)
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("HTTP error: %d %s (body: %s)", resp.StatusCode, resp.Status, string(body))
	}

	// Parse JSON response
	var oembedResp oEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&oembedResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &oembedResp, nil
}

// oembedToMetadata converts an oEmbed response to our standard metadata format
func (e *OEmbedExtractor) oembedToMetadata(oembed *oEmbedResponse, originalURL string) map[string]string {
	metadata := make(map[string]string)

	// Title - primary field
	if oembed.Title != "" {
		metadata["title"] = oembed.Title
	}

	// Description - use author_name or provider_name as fallback
	if oembed.Description != "" {
		metadata["description"] = oembed.Description
	} else if oembed.AuthorName != "" {
		metadata["description"] = fmt.Sprintf("By %s", oembed.AuthorName)
	} else {
		// Fallback to URL if no description
		metadata["description"] = originalURL
	}

	// Image - use thumbnail
	if oembed.ThumbnailURL != "" {
		metadata["image"] = oembed.ThumbnailURL
	}

	// Site name - use provider name or author name
	if oembed.ProviderName != "" {
		metadata["site_name"] = oembed.ProviderName
	} else if oembed.AuthorName != "" {
		metadata["site_name"] = oembed.AuthorName
	}

	// Additional metadata for debugging/logging
	if oembed.Type != "" {
		metadata["oembed_type"] = oembed.Type
	}

	return metadata
}

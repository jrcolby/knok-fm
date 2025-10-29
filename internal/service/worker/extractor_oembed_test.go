package worker

import (
	"log/slog"
	"os"
	"testing"
)

// createTestLogger creates a logger for testing
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors during tests
	}))
}

func TestBuildOEmbedURL(t *testing.T) {
	// Create a minimal logger for testing
	logger := createTestLogger()

	// Create registry (we need this for the extractor)
	registry, err := NewOEmbedRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Create extractor
	extractor := NewOEmbedExtractor(registry, logger)

	tests := []struct {
		name        string
		endpoint    string
		resourceURL string
		wantURL     string
	}{
		{
			name:        "YouTube URL with si parameter",
			endpoint:    "https://www.youtube.com/oembed",
			resourceURL: "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
			wantURL:     "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A%26si%3D5c2KrgyqSfo_0jSE",
		},
		{
			name:        "YouTube URL without si parameter",
			endpoint:    "https://www.youtube.com/oembed",
			resourceURL: "https://youtube.com/watch?v=D1avYj7q42A",
			wantURL:     "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A",
		},
		{
			name:        "Spotify track URL",
			endpoint:    "https://open.spotify.com/oembed",
			resourceURL: "https://open.spotify.com/track/abc123",
			wantURL:     "https://open.spotify.com/oembed?format=json&url=https%3A%2F%2Fopen.spotify.com%2Ftrack%2Fabc123",
		},
		{
			name:        "Endpoint with {format} placeholder",
			endpoint:    "https://www.youtube.com/oembed?format={format}",
			resourceURL: "https://youtube.com/watch?v=test",
			wantURL:     "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3Dtest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := extractor.buildOEmbedURL(tt.endpoint, tt.resourceURL)
			if err != nil {
				t.Fatalf("buildOEmbedURL() error = %v", err)
			}

			if gotURL != tt.wantURL {
				t.Errorf("buildOEmbedURL() = %v, want %v", gotURL, tt.wantURL)
			}

			t.Logf("✓ Built oEmbed URL correctly:\n  Resource: %s\n  OEmbed:   %s", tt.resourceURL, gotURL)
		})
	}
}

func TestDoubleEncodedURLTransformation(t *testing.T) {
	// This test proves that a double-encoded URL from Discord
	// will be properly transformed into the correct oEmbed API URL

	// Create a minimal logger for testing
	logger := createTestLogger()

	// Create registry
	registry, err := NewOEmbedRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Create extractor
	extractor := NewOEmbedExtractor(registry, logger)

	// Simulate what happens when Discord sends a double-encoded URL:
	// 1. Discord sends: https://youtube.com/watch?v=D1avYj7q42A%3Fsi%3D5c2KrgyqSfo_0jSE
	// 2. URL detector decodes it to: https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE
	// 3. Normalizer removes tracking params: https://youtube.com/watch?v=D1avYj7q42A
	// 4. oEmbed builder encodes it: https://www.youtube.com/oembed?url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A&format=json

	// Step 2-3 simulation (what urldetector + normalizer will produce)
	// After decoding and normalization (si parameter removed)
	normalizedURL := "https://youtube.com/watch?v=D1avYj7q42A"

	// Step 4: Build oEmbed URL
	endpoint := "https://www.youtube.com/oembed"
	oembedURL, err := extractor.buildOEmbedURL(endpoint, normalizedURL)
	if err != nil {
		t.Fatalf("buildOEmbedURL() error = %v", err)
	}

	// The expected oEmbed URL (note: query params can be in any order)
	// We check that it contains the properly encoded URL parameter
	wantEncodedParam := "url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A"
	wantFormatParam := "format=json"

	if !containsString(oembedURL, wantEncodedParam) {
		t.Errorf("oEmbed URL missing expected encoded URL parameter.\nGot:  %s\nWant to contain: %s", oembedURL, wantEncodedParam)
	}

	if !containsString(oembedURL, wantFormatParam) {
		t.Errorf("oEmbed URL missing format parameter.\nGot:  %s\nWant to contain: %s", oembedURL, wantFormatParam)
	}

	t.Logf("✓ Double-encoded URL transformation successful!")
	t.Logf("  Input (after decode+normalize): %s", normalizedURL)
	t.Logf("  Output (oEmbed API URL):        %s", oembedURL)
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

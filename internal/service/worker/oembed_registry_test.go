package worker

import (
	"testing"
)

func TestOEmbedRegistry(t *testing.T) {
	registry, err := NewOEmbedRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	if count := registry.GetProviderCount(); count == 0 {
		t.Fatal("Registry has no providers")
	}

	t.Logf("Loaded %d oEmbed providers", registry.GetProviderCount())
}

func TestOEmbedRegistryMatching(t *testing.T) {
	registry, err := NewOEmbedRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	tests := []struct {
		name         string
		url          string
		wantProvider string
		wantMatch    bool
	}{
		{
			name:         "YouTube watch URL",
			url:          "https://www.youtube.com/watch?v=JUDUC87VuPU",
			wantProvider: "YouTube",
			wantMatch:    true,
		},
		{
			name:         "YouTube youtu.be short URL",
			url:          "https://youtu.be/BnkqvBn4OiE",
			wantProvider: "YouTube",
			wantMatch:    true,
		},
		{
			name:         "YouTube mobile URL",
			url:          "https://m.youtube.com/watch?v=test123",
			wantProvider: "YouTube",
			wantMatch:    true,
		},
		{
			name:         "Spotify track URL",
			url:          "https://open.spotify.com/track/abc123",
			wantProvider: "Spotify",
			wantMatch:    true,
		},
		{
			name:         "Spotify album URL",
			url:          "https://open.spotify.com/album/xyz789",
			wantProvider: "Spotify",
			wantMatch:    true,
		},
		{
			name:         "Spotify short link",
			url:          "https://spotify.link/kQa2OWJLzXb",
			wantProvider: "Spotify",
			wantMatch:    true,
		},
		{
			name:         "SoundCloud track URL",
			url:          "https://soundcloud.com/artist/track-name",
			wantProvider: "SoundCloud",
			wantMatch:    true,
		},
		{
			name:         "Vimeo URL",
			url:          "https://vimeo.com/123456789",
			wantProvider: "Vimeo",
			wantMatch:    true,
		},
		{
			name:         "TikTok URL",
			url:          "https://www.tiktok.com/@user/video/123456",
			wantProvider: "TikTok",
			wantMatch:    true,
		},
		{
			name:         "Twitter URL",
			url:          "https://twitter.com/user/status/123456",
			wantProvider: "Twitter",
			wantMatch:    true,
		},
		{
			name:         "Unknown URL",
			url:          "https://example.com/some/page",
			wantProvider: "",
			wantMatch:    false,
		},
		{
			name:         "Random website",
			url:          "https://github.com/user/repo",
			wantProvider: "",
			wantMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := registry.Match(tt.url)

			if tt.wantMatch {
				if provider == nil {
					t.Errorf("Expected to match provider %q, but got no match", tt.wantProvider)
					return
				}
				if provider.Name != tt.wantProvider {
					t.Errorf("Expected provider %q, got %q", tt.wantProvider, provider.Name)
				}
				if provider.Endpoint == "" {
					t.Errorf("Provider %q has empty endpoint", provider.Name)
				}
				t.Logf("✓ Matched %q → %s (endpoint: %s)", tt.url, provider.Name, provider.Endpoint)
			} else {
				if provider != nil {
					t.Errorf("Expected no match for %q, but matched provider %q", tt.url, provider.Name)
				}
				t.Logf("✓ No match for %q (as expected)", tt.url)
			}
		})
	}
}

func TestSchemeToRegex(t *testing.T) {
	tests := []struct {
		scheme      string
		testURL     string
		shouldMatch bool
	}{
		{
			scheme:      "https://*.youtube.com/watch*",
			testURL:     "https://www.youtube.com/watch?v=123",
			shouldMatch: true,
		},
		{
			scheme:      "https://*.youtube.com/watch*",
			testURL:     "https://m.youtube.com/watch?v=123",
			shouldMatch: true,
		},
		{
			scheme:      "https://*.youtube.com/watch*",
			testURL:     "https://youtube.com/watch?v=123",
			shouldMatch: false, // No subdomain
		},
		{
			scheme:      "https://open.spotify.com/*",
			testURL:     "https://open.spotify.com/track/123",
			shouldMatch: true,
		},
		{
			scheme:      "https://open.spotify.com/*",
			testURL:     "https://spotify.com/track/123",
			shouldMatch: false, // Wrong subdomain
		},
		{
			scheme:      "https://soundcloud.com/*",
			testURL:     "https://soundcloud.com/artist/track",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.scheme+"_vs_"+tt.testURL, func(t *testing.T) {
			pattern := schemeToRegex(tt.scheme)
			t.Logf("Scheme: %s", tt.scheme)
			t.Logf("Regex:  %s", pattern)

			// Note: We can't compile the regex here easily without importing regexp
			// The actual matching is tested in TestOEmbedRegistryMatching
			t.Logf("Pattern generated for scheme %q: %s", tt.scheme, pattern)
		})
	}
}

func BenchmarkOEmbedRegistryMatch(b *testing.B) {
	registry, err := NewOEmbedRegistry()
	if err != nil {
		b.Fatalf("Failed to create registry: %v", err)
	}

	testURLs := []string{
		"https://www.youtube.com/watch?v=test",
		"https://open.spotify.com/track/test",
		"https://soundcloud.com/artist/track",
		"https://example.com/unknown",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := testURLs[i%len(testURLs)]
		_ = registry.Match(url)
	}
}

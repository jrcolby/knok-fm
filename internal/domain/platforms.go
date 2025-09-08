package domain

import "regexp"

// Platform represents a music streaming platform
type Platform struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	URLPatterns []string `json:"url_patterns"`
}

// PlatformConfig holds all platform configurations
type PlatformConfig struct {
	Platforms map[string]Platform `json:"platforms"`
}

// Platform constants - single source of truth
const (
	PlatformYouTube    = "youtube"
	PlatformSoundCloud = "soundcloud"
	PlatformMixcloud   = "mixcloud"
	PlatformBandcamp   = "bandcamp"
	PlatformSpotify    = "spotify"
	PlatformAppleMusic = "apple_music"
	PlatformNTS        = "nts"
	PlatformDublab     = "dublab"
	PlatformNoods      = "noods"
)

// GetPlatformConfig returns the centralized platform configuration
func GetPlatformConfig() PlatformConfig {
	return PlatformConfig{
		Platforms: map[string]Platform{
			PlatformYouTube: {
				ID:          PlatformYouTube,
				Name:        "YouTube",
				URLPatterns: []string{"youtube.com", "youtu.be"},
			},
			PlatformSoundCloud: {
				ID:          PlatformSoundCloud,
				Name:        "SoundCloud",
				URLPatterns: []string{"soundcloud.com"},
			},
			PlatformMixcloud: {
				ID:          PlatformMixcloud,
				Name:        "Mixcloud",
				URLPatterns: []string{"mixcloud.com"},
			},
			PlatformBandcamp: {
				ID:          PlatformBandcamp,
				Name:        "Bandcamp",
				URLPatterns: []string{"bandcamp.com"},
			},
			PlatformSpotify: {
				ID:          PlatformSpotify,
				Name:        "Spotify",
				URLPatterns: []string{"spotify.com"},
			},
			PlatformAppleMusic: {
				ID:          PlatformAppleMusic,
				Name:        "Apple Music",
				URLPatterns: []string{"music.apple.com"},
			},
			PlatformNTS: {
				ID:          PlatformNTS,
				Name:        "NTS Radio",
				URLPatterns: []string{"nts.live"},
			},
			PlatformDublab: {
				ID:          PlatformDublab,
				Name:        "Dublab",
				URLPatterns: []string{"dublab.com"},
			},
			PlatformNoods: {
				ID:          PlatformNoods,
				Name:        "Noods Radio",
				URLPatterns: []string{"noodsradio.com"},
			},
		},
	}
}

// DetectPlatformFromURL detects the platform from a URL using centralized config
func DetectPlatformFromURL(url string) string {
	config := GetPlatformConfig()

	for platformID, platform := range config.Platforms {
		for _, pattern := range platform.URLPatterns {
			matched, _ := regexp.MatchString(`(?i)`+regexp.QuoteMeta(pattern), url)
			if matched {
				return platformID
			}
		}
	}

	return "unknown"
}

// GetValidPlatforms returns a list of all valid platform IDs
func GetValidPlatforms() []string {
	config := GetPlatformConfig()
	platforms := make([]string, 0, len(config.Platforms))

	for platformID := range config.Platforms {
		platforms = append(platforms, platformID)
	}

	return platforms
}

// IsValidPlatform checks if a platform ID is valid
func IsValidPlatform(platformID string) bool {
	config := GetPlatformConfig()
	_, exists := config.Platforms[platformID]
	return exists
}

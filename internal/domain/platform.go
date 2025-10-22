package domain

import (
	"regexp"
	"time"
)

// Platform represents a music streaming platform
type Platform struct {
	ID                 string     `json:"id" db:"id"`
	Name               string     `json:"name" db:"name"`
	URLPatterns        []string   `json:"url_patterns" db:"url_patterns"`
	Priority           int        `json:"priority" db:"priority"`
	Enabled            bool       `json:"enabled" db:"enabled"`
	ExtractionPatterns []string   `json:"extraction_patterns,omitempty" db:"extraction_patterns"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty" db:"updated_at"`
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
	PlatformRinseFM    = "rinse_fm"
	PlatformTidal      = "tidal"
	PlatformDeezer     = "deezer"
	PlatformUnknown    = "unknown" // For unrecognized music platforms
)

// GetDefaultPlatformConfig returns the default platform configuration used for seeding and fallback
func GetDefaultPlatformConfig() PlatformConfig {
	return PlatformConfig{
		Platforms: map[string]Platform{
			PlatformYouTube: {
				ID:   PlatformYouTube,
				Name: "YouTube",
				URLPatterns: []string{
					"youtube.com",
					"youtu.be",
					"m.youtube.com",     // Mobile
					"music.youtube.com", // YouTube Music
				},
			},
			PlatformSoundCloud: {
				ID:   PlatformSoundCloud,
				Name: "SoundCloud",
				URLPatterns: []string{
					"soundcloud.com",
					"on.soundcloud.com", // Short link
					"m.soundcloud.com",  // Mobile
				},
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
				ID:   PlatformSpotify,
				Name: "Spotify",
				URLPatterns: []string{
					"spotify.com",
					"open.spotify.com",   // Web player
					"play.spotify.com",   // Legacy
					"link.tospotify.com", // Mobile share
				},
			},
			PlatformAppleMusic: {
				ID:   PlatformAppleMusic,
				Name: "Apple Music",
				URLPatterns: []string{
					"music.apple.com",
					"itunes.apple.com", // Legacy iTunes links
				},
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
			PlatformRinseFM: {
				ID:          PlatformRinseFM,
				Name:        "Rinse FM",
				URLPatterns: []string{"rinse.fm", "www.rinse.fm"},
			},
			PlatformTidal: {
				ID:   PlatformTidal,
				Name: "Tidal",
				URLPatterns: []string{
					"tidal.com",
					"listen.tidal.com",
				},
			},
			PlatformDeezer: {
				ID:   PlatformDeezer,
				Name: "Deezer",
				URLPatterns: []string{
					"deezer.com",
					"deezer.page.link", // Short link
				},
			},
		},
	}
}

// DetectPlatformFromURL detects the platform from a URL using default config
// Note: This is a legacy function. New code should use the PlatformLoader service instead.
func DetectPlatformFromURL(url string) string {
	config := GetDefaultPlatformConfig()

	for platformID, platform := range config.Platforms {
		for _, pattern := range platform.URLPatterns {
			matched, _ := regexp.MatchString(`(?i)`+regexp.QuoteMeta(pattern), url)
			if matched {
				return platformID
			}
		}
	}

	// Return PlatformUnknown constant instead of empty string
	return PlatformUnknown
}

// GetValidPlatforms returns a list of all valid platform IDs from default config
func GetValidPlatforms() []string {
	config := GetDefaultPlatformConfig()
	platforms := make([]string, 0, len(config.Platforms))

	for platformID := range config.Platforms {
		platforms = append(platforms, platformID)
	}

	return platforms
}

// IsValidPlatform checks if a platform ID is valid in default config
func IsValidPlatform(platformID string) bool {
	config := GetDefaultPlatformConfig()
	_, exists := config.Platforms[platformID]
	return exists
}

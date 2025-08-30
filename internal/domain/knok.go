package domain

import (
	"time"

	"github.com/google/uuid"
)

// Knok represents a music track, mix, or DJ set shared in a Discord server
type Knok struct {
	ID       uuid.UUID `json:"id" db:"id"`
	ServerID string    `json:"server_id" db:"server_id"`
	URL      string    `json:"url" db:"url"`
	Platform string  `json:"platform" db:"platform"`
	Title    *string `json:"title" db:"title"`

	// Discord-specific fields
	DiscordMessageID string  `json:"discord_message_id" db:"discord_message_id"`
	DiscordChannelID string  `json:"discord_channel_id" db:"discord_channel_id"`
	MessageContent   *string `json:"message_content" db:"message_content"`

	// Metadata and processing
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	ExtractionStatus string                 `json:"extraction_status" db:"extraction_status"`

	// Timestamps
	PostedAt  time.Time  `json:"posted_at" db:"posted_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}

// Platform constants
const (
	PlatformYouTube    = "youtube"
	PlatformSoundCloud = "soundcloud"
	PlatformMixcloud   = "mixcloud"
	PlatformBandcamp   = "bandcamp"
	PlatformSpotify    = "spotify"
	PlatformAppleMusic = "apple_music"
)

// Extraction status constants
const (
	ExtractionStatusPending    = "pending"
	ExtractionStatusProcessing = "processing"
	ExtractionStatusComplete   = "complete"
	ExtractionStatusFailed     = "failed"
)

// IsValidPlatform checks if the platform is supported
func (k *Knok) IsValidPlatform() bool {
	validPlatforms := map[string]bool{
		PlatformYouTube:    true,
		PlatformSoundCloud: true,
		PlatformMixcloud:   true,
		PlatformBandcamp:   true,
		PlatformSpotify:    true,
		PlatformAppleMusic: true,
	}
	return validPlatforms[k.Platform]
}


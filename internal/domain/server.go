package domain

import "time"

// Server represents a Discord server configuration
type Server struct {
	ID                  string                 `json:"id" db:"id"`
	Name                string                 `json:"name" db:"name"`
	ConfiguredChannelID *string                `json:"configured_channel_id" db:"configured_channel_id"`

	// Settings is stored as JSONB in the database and contains server-specific configuration
	// Schema documented in ServerSettings struct below
	Settings            map[string]interface{} `json:"settings" db:"settings"`

	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           *time.Time             `json:"updated_at" db:"updated_at"`
}

// ServerSettings represents configurable server options stored in the Settings JSONB field
//
// Example Settings JSONB:
// {
//   "unknown_platform_mode": "permissive",  // or "strict"
//   "auto_extraction": true,
//   "allowed_channels": ["123456789"],
//   "banned_users": ["987654321"],
//   "require_metadata": false,
//   "notification_channel": "111222333",
//   "max_knoks_per_user": 100
// }
type ServerSettings struct {
	// UnknownPlatformMode controls how the server handles URLs from unrecognized platforms
	// Values: "permissive" (accept all URLs) or "strict" (reject unknown platforms)
	// If not set, falls back to global config.DefaultUnknownPlatformMode
	UnknownPlatformMode *string  `json:"unknown_platform_mode"`

	AutoExtraction      bool     `json:"auto_extraction"`
	AllowedChannels     []string `json:"allowed_channels"`
	BannedUsers         []string `json:"banned_users"`
	RequireMetadata     bool     `json:"require_metadata"`
	NotificationChannel *string  `json:"notification_channel"`
	MaxKnoksPerUser     *int     `json:"max_knoks_per_user"`
}

// HasConfiguredChannel returns true if a channel is configured for knok tracking
func (s *Server) HasConfiguredChannel() bool {
	return s.ConfiguredChannelID != nil && *s.ConfiguredChannelID != ""
}

// IsChannelAllowed checks if a channel ID is allowed for knok tracking
func (s *Server) IsChannelAllowed(channelID string) bool {
	// If no specific channel is configured, allow all
	if !s.HasConfiguredChannel() {
		return true
	}

	// If a specific channel is configured, only allow that one
	return *s.ConfiguredChannelID == channelID
}

package domain

import "time"

// Server represents a Discord server configuration
type Server struct {
	ID                  string                 `json:"id" db:"id"`
	Name                string                 `json:"name" db:"name"`
	ConfiguredChannelID *string                `json:"configured_channel_id" db:"configured_channel_id"`
	Settings            map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           *time.Time             `json:"updated_at" db:"updated_at"`
}

// ServerSettings represents configurable server options
type ServerSettings struct {
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

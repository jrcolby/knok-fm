package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Command definitions
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "stats",
		Description: "Show server music statistics",
		Type:        discordgo.ChatApplicationCommand,
	},
}

// registerCommands registers slash commands with Discord
func (s *BotService) registerCommands() error {
	s.logger.Info("Registering slash commands...")

	// Register commands globally (takes up to 1 hour to propagate)
	_, err := s.session.ApplicationCommandBulkOverwrite(s.session.State.User.ID, "", commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	s.logger.Info("Slash commands registered successfully")
	return nil
}

// onInteractionCreate handles slash command interactions
func (s *BotService) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	command := interaction.ApplicationCommandData()
	s.logger.Debug("Received slash command",
		"command", command.Name,
		"user_id", interaction.User.ID,
		"guild_id", interaction.GuildID,
	)

	var response *discordgo.InteractionResponse

	switch command.Name {
	case "recent":
		response = s.handleRecentCommand(interaction)
	case "stats":
		response = s.handleStatsCommand(interaction)
	case "search":
		response = s.handleSearchCommand(interaction)
	default:
		response = &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Unknown command",
			},
		}
	}

	// Send response
	if err := session.InteractionRespond(interaction.Interaction, response); err != nil {
		s.logger.Error("Failed to respond to interaction", "error", err)
	}
}

// handleRecentCommand handles the /recent command
func (s *BotService) handleRecentCommand(interaction *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	// Get count option (default to 5)
	count := 5
	if len(interaction.ApplicationCommandData().Options) > 0 {
		if option := interaction.ApplicationCommandData().Options[0]; option.Name == "count" {
			if countVal, ok := option.Value.(float64); ok {
				count = int(countVal)
			}
		}
	}

	// TODO: Implement recent knoks fetching from database
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "🎵 Recent Music Knoks",
					Color: 0x00ff00,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Status",
							Value: "Coming soon... The bot will show the latest knoks shared in this server.",
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("Requested %d knoks", count),
					},
				},
			},
		},
	}

	return response
}

// handleStatsCommand handles the /stats command
func (s *BotService) handleStatsCommand(interaction *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	// TODO: Implement stats fetching from database
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "📊 Server Music Statistics",
					Color: 0x0099ff,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Total Knoks",
							Value: "Coming soon...",
						},
						{
							Name:  "Most Active Platform",
							Value: "Coming soon...",
						},
						{
							Name:  "Recent Activity",
							Value: "Coming soon...",
						},
					},
				},
			},
		},
	}

	return response
}

// handleSearchCommand handles the /search command
func (s *BotService) handleSearchCommand(interaction *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	// Get search query
	var query string
	var limit int = 5

	for _, option := range interaction.ApplicationCommandData().Options {
		switch option.Name {
		case "query":
			if queryVal, ok := option.Value.(string); ok {
				query = queryVal
			}
		case "limit":
			if limitVal, ok := option.Value.(float64); ok {
				limit = int(limitVal)
			}
		}
	}

	if query == "" {
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Please provide a search query",
			},
		}
	}

	// TODO: Implement search functionality
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "🔍 Search Results",
					Color:       0xff9900,
					Description: fmt.Sprintf("Searching for: **%s**", query),
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Status",
							Value: "Coming soon... The bot will search through all knoks in this server.",
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("Limit: %d results", limit),
					},
				},
			},
		},
	}

	return response
}

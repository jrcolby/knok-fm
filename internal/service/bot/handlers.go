package bot

import (
	"context"
	"fmt"
	"knock-fm/internal/domain"
	"knock-fm/internal/pkg/urldetector"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)


// onMessageCreate handles new Discord messages
func (s *BotService) onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// DEBUG: Add unique handler invocation ID to trace duplicates
	handlerID := fmt.Sprintf("%p-%d", s, message.Timestamp.UnixNano())

	// ALWAYS log this at INFO level to catch both invocations
	s.logger.Info("🚨 HANDLER_INVOCATION: onMessageCreate called",
		"handler_id", handlerID,
		"message_id", message.ID,
		"bot_service_ptr", fmt.Sprintf("%p", s),
		"session_ptr", fmt.Sprintf("%p", session),
		"timestamp", message.Timestamp.UnixNano(),
	)

	// Ignore bot messages
	if message.Author.Bot {
		s.logger.Debug("HANDLER_EXIT: Ignoring bot message", "handler_id", handlerID)
		return
	}

	// Check global guild (server) restrictions from environment
	if len(s.config.DiscordAllowedGuilds) > 0 {
		allowed := false
		for _, guildID := range s.config.DiscordAllowedGuilds {
			if guildID == message.GuildID {
				allowed = true
				break
			}
		}
		if !allowed {
			s.logger.Debug("HANDLER_EXIT: Guild not in allowed list",
				"handler_id", handlerID,
				"guild_id", message.GuildID,
				"allowed_guilds", s.config.DiscordAllowedGuilds,
			)
			return
		}
	}

	// Check global channel restrictions from environment
	if len(s.config.DiscordAllowedChannels) > 0 {
		allowed := false
		for _, channelID := range s.config.DiscordAllowedChannels {
			if channelID == message.ChannelID {
				allowed = true
				break
			}
		}
		if !allowed {
			s.logger.Debug("HANDLER_EXIT: Channel not in global allowed list",
				"handler_id", handlerID,
				"channel_id", message.ChannelID,
				"allowed_channels", s.config.DiscordAllowedChannels,
			)
			return
		}
	}

	// Check if server has channel restrictions (per-server database settings)
	if s.serverRepo != nil {
		server, err := s.serverRepo.GetByID(context.Background(), message.GuildID)
		if err == nil && server != nil && server.Settings != nil {
			if allowedChannels, ok := server.Settings["allowed_channels"].([]interface{}); ok && len(allowedChannels) > 0 {
				// Check if message channel is in allowed list
				isAllowed := false
				for _, ch := range allowedChannels {
					if channelID, ok := ch.(string); ok && channelID == message.ChannelID {
						isAllowed = true
						break
					}
				}
				if !isAllowed {
					s.logger.Debug("Message from non-allowed channel, ignoring",
						"handler_id", handlerID,
						"channel_id", message.ChannelID,
						"allowed_channels", allowedChannels,
					)
					return
				}
				s.logger.Debug("Message from allowed channel, processing",
					"handler_id", handlerID,
					"channel_id", message.ChannelID,
				)
			}
		}
	}

	// Check if message contains any supported URLs
	s.logger.Info("🔍 RAW MESSAGE CONTENT from Discord",
		"handler_id", handlerID,
		"message_content", message.Content,
		"content_length", len(message.Content))

	urls := s.extractURLs(message.Content)
	if len(urls) == 0 {
		s.logger.Debug("HANDLER_EXIT: No URLs found",
			"handler_id", handlerID)
		return
	}

	s.logger.Debug("EXTRACTED_URLS: Found URLs",
		"handler_id", handlerID,
		"url_count", len(urls),
		"urls", urls,
	)

	s.logger.Info("Detected music URLs in message from within onMessageCreate",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"guild_id", message.GuildID,
		"urls", urls,
	)

	// Process each detected URL
	knoksCreated := 0
	for i, urlInfo := range urls {
		s.logger.Debug("PROCESSING_URL: Starting URL processing",
			"handler_id", handlerID,
			"url_index", i,
			"url", urlInfo.URL,
			"platform", urlInfo.Platform,
		)

		if err := s.processDetectedURL(message, urlInfo); err != nil {
			s.logger.Error("Failed to process URL",
				"error", err,
				"url", urlInfo.URL,
				"message_id", message.ID,
			)
		} else {
			knoksCreated++
		}
	}

	if knoksCreated > 0 {
		s.logger.Info("Successfully processed music URLs",
			"message_id", message.ID,
			"guild_id", message.GuildID,
			"knoks_created", knoksCreated,
		)

		s.logger.Debug("HANDLER_EXIT: Processing completed successfully",
			"handler_id", handlerID,
			"knoks_created", knoksCreated,
		)

		// Add emoji reaction to give user feedback
		if err := session.MessageReactionAdd(message.ChannelID, message.ID, "🎵"); err != nil {
			s.logger.Warn("Failed to add emoji reaction",
				"error", err,
				"message_id", message.ID,
			)
		}
	}
}

// processDetectedURL creates knok records and queues metadata extraction jobs
func (s *BotService) processDetectedURL(message *discordgo.MessageCreate, urlInfo urldetector.URLInfo) error {
	ctx := context.Background()

	// DEBUG: Track processDetectedURL invocations
	processID := fmt.Sprintf("PROCESS_%d_%s", time.Now().UnixNano(), urlInfo.URL[len(urlInfo.URL)-8:])
	s.logger.Info("🔍 PROCESS_ENTRY: processDetectedURL called",
		"process_id", processID,
		"message_id", message.ID,
		"url", urlInfo.URL,
		"platform", urlInfo.Platform,
		"goroutine_id", fmt.Sprintf("%p", &ctx), // Unique per goroutine
	)

	// Check if platform is unknown and handle according to server settings
	if urlInfo.Platform == domain.PlatformUnknown {
		// Get unknown platform mode (server override or global default)
		mode := s.config.DefaultUnknownPlatformMode // Global default

		// Check for server-specific override
		if s.serverRepo != nil {
			server, err := s.serverRepo.GetByID(ctx, message.GuildID)
			if err == nil && server != nil && server.Settings != nil {
				// Check for server-specific unknown_platform_mode setting
				if serverMode, ok := server.Settings["unknown_platform_mode"].(string); ok {
					mode = serverMode
					s.logger.Debug("Using server-specific unknown platform mode",
						"server_id", message.GuildID,
						"mode", mode,
					)
				}
			}
		}

		// If strict mode, reject unknown platforms
		if mode == "strict" {
			s.logger.Info("Rejecting unknown platform URL (strict mode)",
				"url", urlInfo.URL,
				"server_id", message.GuildID,
				"message_id", message.ID,
				"mode", mode,
			)
			return nil // Don't create knok, don't queue job
		}

		// Permissive mode: Continue processing with platform="unknown"
		s.logger.Info("Accepting unknown platform URL (permissive mode)",
			"url", urlInfo.URL,
			"platform", urlInfo.Platform,
			"server_id", message.GuildID,
			"mode", mode,
		)
	}

	// Check for existing knok by URL first (to avoid duplicates)
	var knokID uuid.UUID
	var existingKnok *domain.Knok

	// Check for existing knok by Discord message ID
	if s.knokRepo != nil {
		existingKnok, err := s.knokRepo.GetByDiscordMessage(ctx, message.ID)
		if err == nil && existingKnok != nil {
			// Use existing knok ID
			knokID = existingKnok.ID
			s.logger.Debug("Found existing knok by Discord message",
				"existing_knok_id", existingKnok.ID,
				"extraction_status", existingKnok.ExtractionStatus,
			)
		}
	}

	// Check for existing knok with same URL in this server
	if existingKnok == nil && s.knokRepo != nil {
		existingKnok, err := s.knokRepo.GetByURL(ctx, message.GuildID, urlInfo.URL)
		if err == nil && existingKnok != nil {
			// Use existing knok ID
			knokID = existingKnok.ID
			s.logger.Debug("Found existing knok by URL",
				"knok_id", knokID,
				"extraction_status", existingKnok.ExtractionStatus,
			)
		}
	}

	// Generate new knok ID only if we don't have an existing one
	if knokID == uuid.Nil {
		knokID = uuid.New()
		s.logger.Debug("Generated new knok ID",
			"knok_id", knokID,
		)
	}

	// Queue metadata extraction job
	jobPayload := map[string]interface{}{
		"knok_id":            knokID.String(),
		"url":                urlInfo.URL,
		"platform":           urlInfo.Platform,
		"discord_message_id": message.ID,
		"discord_channel_id": message.ChannelID,
		"discord_guild_id":   message.GuildID,
		"discord_user_id":    message.Author.ID,
		"message_content":    message.Content,
	}

	// Ensure server exists in database before creating knok
	if s.serverRepo != nil {
		server, err := s.serverRepo.GetByID(ctx, message.GuildID)
		if err != nil {
			s.logger.Warn("Server not found in database, creating basic record",
				"guild_id", message.GuildID,
				"error", err,
			)
			// Create basic server record
			server = &domain.Server{
				ID:        message.GuildID,
				Name:      message.GuildID, // Will be updated later
				CreatedAt: time.Now(),
			}
			if err := s.serverRepo.Create(ctx, server); err != nil {
				s.logger.Error("Failed to create server record", "error", err)
			}
		}
	}

	// Only create new knok record if we don't have an existing one
	if existingKnok == nil && s.knokRepo != nil {
		// Create Knok record with basic info
		now := time.Now()
		knok := &domain.Knok{
			ID:               knokID,
			ServerID:         message.GuildID,
			URL:              urlInfo.URL,
			Platform:         urlInfo.Platform,
			DiscordMessageID: message.ID,
			DiscordChannelID: message.ChannelID,
			MessageContent:   &message.Content,
			ExtractionStatus: domain.ExtractionStatusPending,
			PostedAt:         now,
			CreatedAt:        now,
		}

		// Store knok in database
		if err := s.knokRepo.Create(ctx, knok); err != nil {
			s.logger.Debug("Knok creation failed, checking if it already exists",
				"knok_id", knokID,
				"error", err,
			)
			
			// Check if knok already exists (possible race condition)
			if existingCheck, checkErr := s.knokRepo.GetByID(ctx, knokID); checkErr != nil {
				s.logger.Error("Failed to create knok and failed to verify existence",
					"knok_id", knokID,
					"create_error", err,
					"check_error", checkErr,
				)
				return fmt.Errorf("failed to create knok: %w", err)
			} else if existingCheck != nil {
				s.logger.Debug("Knok already exists, continuing with job queue",
					"knok_id", knokID,
				)
			} else {
				s.logger.Error("Knok creation failed and knok doesn't exist",
					"knok_id", knokID,
					"error", err,
				)
				return fmt.Errorf("failed to create knok: %w", err)
			}
		} else {
			s.logger.Info("Knok record created",
				"knok_id", knokID,
				"url", urlInfo.URL,
				"platform", urlInfo.Platform,
			)
		}
	} else if existingKnok != nil {
		s.logger.Debug("Using existing knok record",
			"knok_id", knokID,
		)
	} else {
		s.logger.Debug("Knok repository not available, skipping database storage",
			"knok_id", knokID,
		)
	}

	// Decide whether to queue metadata extraction job based on existing knok status
	shouldQueueJob := true
	if existingKnok != nil {
		switch existingKnok.ExtractionStatus {
		case domain.ExtractionStatusComplete:
			// Metadata already extracted successfully, no need to re-process
			shouldQueueJob = false
			s.logger.Info("Knok already has complete metadata, skipping job queue",
				"knok_id", knokID,
				"url", urlInfo.URL,
				"title", func() string {
					if existingKnok.Title != nil {
						return *existingKnok.Title
					}
					return "N/A"
				}(),
			)
		case domain.ExtractionStatusProcessing:
			// Already being processed, don't queue duplicate job
			shouldQueueJob = false
			s.logger.Info("Knok is currently being processed, skipping job queue",
				"knok_id", knokID,
				"url", urlInfo.URL,
			)
		case domain.ExtractionStatusPending, domain.ExtractionStatusFailed:
			// Should re-process pending or failed extractions
			shouldQueueJob = true
			s.logger.Info("Knok needs metadata extraction, queuing job",
				"knok_id", knokID,
				"url", urlInfo.URL,
				"current_status", existingKnok.ExtractionStatus,
			)
		}
	}

	if !shouldQueueJob {
		s.logger.Debug("Skipping job queue for existing knok",
			"knok_id", knokID,
			"extraction_status", existingKnok.ExtractionStatus,
		)
		return nil
	}

	// Queue the metadata extraction job
	if err := s.queueRepo.Enqueue(ctx, domain.JobTypeExtractMetadata, jobPayload); err != nil {
		s.logger.Error("Failed to queue metadata extraction job",
			"error", err,
			"knok_id", knokID,
			"url", urlInfo.URL,
		)

		// Update knok status to failed if we can't queue the job (only if knokRepo available)
		if s.knokRepo != nil {
			s.knokRepo.UpdateExtractionStatus(ctx, knokID, domain.ExtractionStatusFailed)
		}

		return fmt.Errorf("failed to queue metadata extraction job: %w", err)
	}

	s.logger.Info("Metadata extraction job queued successfully",
		"knok_id", knokID,
		"url", urlInfo.URL,
		"platform", urlInfo.Platform,
		"stored_in_db", s.knokRepo != nil,
	)

	s.logger.Info("🔍 PROCESS_EXIT: processDetectedURL completed",
		"process_id", processID,
		"knok_id", knokID,
		"url", urlInfo.URL,
	)

	return nil
}

// extractURLs finds all supported music URLs in a message using centralized detector
func (s *BotService) extractURLs(content string) []urldetector.URLInfo {
	return s.urlDetector.DetectURLs(content)
}


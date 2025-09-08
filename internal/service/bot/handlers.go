package bot

import (
	"context"
	"fmt"
	"knock-fm/internal/domain"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// URL patterns for different music platforms - simplified to host-based matching
var urlPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?youtube\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?youtu\.be`),
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?soundcloud\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?mixcloud\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?[\w-]+\.bandcamp\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?(open\.)?spotify\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?music\.apple\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?nts\.live`),
	regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?dublab\.com`),
	regexp.MustCompile(`(?i)(?:https?://)?noodsradio\.com`),
}

// URLInfo contains information about a detected URL
type URLInfo struct {
	URL      string
	Platform string
}

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

	// Check if message contains any supported URLs
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
func (s *BotService) processDetectedURL(message *discordgo.MessageCreate, urlInfo URLInfo) error {
	ctx := context.Background()

	// DEBUG: Track processDetectedURL invocations
	processID := fmt.Sprintf("PROCESS_%d_%s", time.Now().UnixNano(), urlInfo.URL[len(urlInfo.URL)-8:])
	s.logger.Info("🔍 PROCESS_ENTRY: processDetectedURL called",
		"process_id", processID,
		"message_id", message.ID,
		"url", urlInfo.URL,
		"goroutine_id", fmt.Sprintf("%p", &ctx), // Unique per goroutine
	)

	// Check for existing knok by URL first (to avoid duplicates)
	var knokID uuid.UUID
	var existingKnok *domain.Knok

	// Check for existing knok by Discord message ID
	if s.knokRepo != nil {
		existingKnok, err := s.knokRepo.GetByDiscordMessage(ctx, message.ID)
		if err == nil && existingKnok != nil {
			// Use existing knok ID
			knokID = existingKnok.ID
			s.logger.Debug("Using existing knok",
				"existing_knok_id", existingKnok.ID,
			)
		}
	}

	// Check for existing knok with same URL in this server
	if existingKnok == nil && s.knokRepo != nil {
		existingKnok, err := s.knokRepo.GetByURL(ctx, message.GuildID, urlInfo.URL)
		if err == nil && existingKnok != nil {
			// Use existing knok ID
			knokID = existingKnok.ID
			s.logger.Debug("Using existing knok",
				"knok_id", knokID,
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
			s.logger.Debug("Knok already created by another process",
				"knok_id", knokID,
			)
			// Continue with job queueing even if creation fails
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

// extractURLs finds all supported music URLs in a message
func (s *BotService) extractURLs(content string) []URLInfo {
	var urls []URLInfo
	seen := make(map[string]bool)

	// Split content into words and check each for URL patterns
	words := strings.Fields(content)
	for _, word := range words {
		for _, pattern := range urlPatterns {
			if pattern.MatchString(word) {
				// Clean up the URL (remove any trailing punctuation)
				url := strings.TrimRight(word, ".,!?;:")

				// Avoid duplicates
				if !seen[url] {
					seen[url] = true
					urls = append(urls, URLInfo{
						URL:      url,
						Platform: s.detectPlatform(url),
					})
				}
				break
			}
		}
	}

	return urls
}

// detectPlatform determines the platform from a URL
func (s *BotService) detectPlatform(url string) string {
	return domain.DetectPlatformFromURL(url)
}

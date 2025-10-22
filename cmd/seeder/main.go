package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	"knock-fm/internal/pkg/logger"
	"knock-fm/internal/pkg/urldetector"
	"knock-fm/internal/repository/postgres"
	"knock-fm/internal/repository/redis"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	var (
		channelID = flag.String("channel", "", "Discord channel ID to seed from (required)")
		guildID   = flag.String("guild", "", "Discord guild/server ID (required)")
		limit     = flag.Int("limit", 0, "Maximum number of messages to fetch (0 = no limit)")
		batchSize = flag.Int("batch", 100, "Number of messages to fetch per Discord API call (max 100)")
		beforeID  = flag.String("before", "", "Fetch messages before this message ID (for pagination)")
		afterID   = flag.String("after", "", "Fetch messages after this message ID (for pagination)")
		dryRun    = flag.Bool("dry-run", false, "Print what would be done without actually creating knoks")
	)
	flag.Parse()

	// Validate required flags
	if *channelID == "" {
		fmt.Fprintln(os.Stderr, "Error: -channel flag is required")
		flag.Usage()
		os.Exit(1)
	}
	if *guildID == "" {
		fmt.Fprintln(os.Stderr, "Error: -guild flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate batch size
	if *batchSize < 1 || *batchSize > 100 {
		fmt.Fprintln(os.Stderr, "Error: -batch must be between 1 and 100")
		os.Exit(1)
	}

	// Load configuration
	cfg := config.Load()

	// Validate bot-specific configuration (need Discord token)
	if err := cfg.ValidateForBot(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	log := logger.New(cfg.LogLevel)
	log.Info("Starting Discord channel seeder...")
	log.Info("Seeder configuration",
		"channel_id", *channelID,
		"guild_id", *guildID,
		"limit", *limit,
		"batch_size", *batchSize,
		"dry_run", *dryRun,
	)

	// Connect to Discord
	discord, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Error("Failed to create Discord session", "error", err)
		os.Exit(1)
	}

	// Test Discord connection
	if _, err := discord.User("@me"); err != nil {
		log.Error("Failed to authenticate with Discord", "error", err)
		os.Exit(1)
	}
	log.Info("Successfully authenticated with Discord")

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	log.Info("Successfully connected to database")

	// Connect to Redis
	redisClient, err := redis.NewClient(cfg.RedisURL, log)
	if err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Test Redis connection
	if err := redis.HealthCheck(context.Background(), redisClient); err != nil {
		log.Error("Failed to ping Redis", "error", err)
		os.Exit(1)
	}
	log.Info("Successfully connected to Redis")

	// Create repositories
	knokRepo := postgres.NewKnokRepository(db, log)
	serverRepo := postgres.NewServerRepository(db, log)
	queueRepo := redis.NewQueueRepository(redisClient, log)

	// Create URL detector
	urlDet := urldetector.New()

	// Create seeder
	seeder := &Seeder{
		discord:      discord,
		knokRepo:     knokRepo,
		serverRepo:   serverRepo,
		queueRepo:    queueRepo,
		urlDetector:  urlDet,
		logger:       log,
		channelID:    *channelID,
		guildID:      *guildID,
		limit:        *limit,
		batchSize:    *batchSize,
		beforeID:     *beforeID,
		afterID:      *afterID,
		dryRun:       *dryRun,
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("Shutdown signal received, stopping seeder...")
		cancel()
	}()

	// Run seeder
	if err := seeder.Run(ctx); err != nil {
		log.Error("Seeder failed", "error", err)
		os.Exit(1)
	}

	log.Info("Seeder completed successfully")
}

// Seeder handles fetching Discord messages and creating knoks
type Seeder struct {
	discord     *discordgo.Session
	knokRepo    domain.KnokRepository
	serverRepo  domain.ServerRepository
	queueRepo   domain.QueueRepository
	urlDetector *urldetector.Detector
	logger      *slog.Logger

	channelID string
	guildID   string
	limit     int
	batchSize int
	beforeID  string
	afterID   string
	dryRun    bool
}

// Run executes the seeding process
func (s *Seeder) Run(ctx context.Context) error {
	// Ensure server exists in database
	if err := s.ensureServer(ctx); err != nil {
		return fmt.Errorf("failed to ensure server exists: %w", err)
	}

	// Fetch messages from Discord
	messages, err := s.fetchMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	s.logger.Info("Fetched messages from Discord",
		"total_messages", len(messages),
	)

	// Process messages and create knoks
	stats := s.processMessages(ctx, messages)

	// Print summary
	s.logger.Info("Seeding completed",
		"messages_processed", stats.MessagesProcessed,
		"urls_detected", stats.URLsDetected,
		"knoks_created", stats.KnoksCreated,
		"knoks_skipped", stats.KnoksSkipped,
		"jobs_queued", stats.JobsQueued,
		"errors", stats.Errors,
	)

	return nil
}

// ensureServer ensures the Discord server exists in the database
func (s *Seeder) ensureServer(ctx context.Context) error {
	// Try to get server from Discord API
	guild, err := s.discord.Guild(s.guildID)
	if err != nil {
		s.logger.Warn("Failed to fetch guild from Discord, using guild ID as name",
			"guild_id", s.guildID,
			"error", err,
		)
	}

	// Check if server exists in database
	server, err := s.serverRepo.GetByID(ctx, s.guildID)
	if err == nil && server != nil {
		s.logger.Info("Server already exists in database",
			"server_id", server.ID,
			"server_name", server.Name,
		)
		return nil
	}

	// Create server record
	serverName := s.guildID
	if guild != nil {
		serverName = guild.Name
	}

	if s.dryRun {
		s.logger.Info("[DRY RUN] Would create server record",
			"guild_id", s.guildID,
			"name", serverName,
		)
		return nil
	}

	newServer := &domain.Server{
		ID:        s.guildID,
		Name:      serverName,
		CreatedAt: time.Now(),
	}

	if err := s.serverRepo.Create(ctx, newServer); err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	s.logger.Info("Created server record",
		"server_id", newServer.ID,
		"server_name", newServer.Name,
	)

	return nil
}

// fetchMessages fetches messages from Discord with pagination
func (s *Seeder) fetchMessages(ctx context.Context) ([]*discordgo.Message, error) {
	var allMessages []*discordgo.Message
	beforeID := s.beforeID
	afterID := s.afterID

	s.logger.Info("Starting message fetch from Discord",
		"channel_id", s.channelID,
		"batch_size", s.batchSize,
	)

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return allMessages, ctx.Err()
		default:
		}

		// Fetch batch of messages
		messages, err := s.discord.ChannelMessages(s.channelID, s.batchSize, beforeID, afterID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages: %w", err)
		}

		// No more messages
		if len(messages) == 0 {
			s.logger.Info("No more messages to fetch")
			break
		}

		s.logger.Info("Fetched message batch",
			"batch_size", len(messages),
			"total_so_far", len(allMessages)+len(messages),
		)

		allMessages = append(allMessages, messages...)

		// Check limit
		if s.limit > 0 && len(allMessages) >= s.limit {
			s.logger.Info("Reached message limit",
				"limit", s.limit,
				"fetched", len(allMessages),
			)
			allMessages = allMessages[:s.limit]
			break
		}

		// Update pagination cursor
		// Discord returns messages in reverse chronological order (newest first)
		beforeID = messages[len(messages)-1].ID

		// Rate limiting: Discord allows 50 requests per second, but be conservative
		time.Sleep(100 * time.Millisecond)
	}

	return allMessages, nil
}

// processMessages processes Discord messages and creates knoks
func (s *Seeder) processMessages(ctx context.Context, messages []*discordgo.Message) *SeedingStats {
	stats := &SeedingStats{}

	for _, message := range messages {
		// Check for cancellation
		select {
		case <-ctx.Done():
			s.logger.Warn("Context cancelled, stopping message processing")
			return stats
		default:
		}

		stats.MessagesProcessed++

		// Skip bot messages
		if message.Author.Bot {
			continue
		}

		// Clean message content from markdown formatting
		cleanedContent := cleanMarkdownLinks(message.Content)

		// Detect URLs
		urls := s.urlDetector.DetectURLs(cleanedContent)
		if len(urls) == 0 {
			continue
		}

		stats.URLsDetected += len(urls)

		// Process each URL
		for _, urlInfo := range urls {
			if err := s.processURL(ctx, message, urlInfo, stats); err != nil {
				s.logger.Error("Failed to process URL",
					"error", err,
					"url", urlInfo.URL,
					"message_id", message.ID,
				)
				stats.Errors++
			}
		}
	}

	return stats
}

// processURL creates a knok and queues a job for a single URL
func (s *Seeder) processURL(ctx context.Context, message *discordgo.Message, urlInfo urldetector.URLInfo, stats *SeedingStats) error {
	// Skip root URLs (e.g., https://www.nts.live without a specific path)
	if isRootURL(urlInfo.URL) {
		s.logger.Debug("Skipping root URL (no specific content)",
			"url", urlInfo.URL,
			"platform", urlInfo.Platform,
		)
		stats.KnoksSkipped++
		return nil
	}

	// Check if knok already exists
	existingKnok, err := s.knokRepo.GetByURL(ctx, s.guildID, urlInfo.URL)
	if err == nil && existingKnok != nil {
		s.logger.Debug("Knok already exists, skipping",
			"knok_id", existingKnok.ID,
			"url", urlInfo.URL,
			"extraction_status", existingKnok.ExtractionStatus,
		)
		stats.KnoksSkipped++
		return nil
	}

	// Generate knok ID
	knokID := uuid.New()

	if s.dryRun {
		s.logger.Info("[DRY RUN] Would create knok",
			"knok_id", knokID,
			"url", urlInfo.URL,
			"platform", urlInfo.Platform,
			"message_id", message.ID,
		)
		stats.KnoksCreated++
		stats.JobsQueued++
		return nil
	}

	// Create knok record
	now := time.Now()
	knok := &domain.Knok{
		ID:               knokID,
		ServerID:         s.guildID,
		URL:              urlInfo.URL,
		Platform:         urlInfo.Platform,
		DiscordMessageID: message.ID,
		DiscordChannelID: message.ChannelID,
		MessageContent:   &message.Content,
		ExtractionStatus: domain.ExtractionStatusPending,
		PostedAt:         message.Timestamp,
		CreatedAt:        now,
	}

	if err := s.knokRepo.Create(ctx, knok); err != nil {
		return fmt.Errorf("failed to create knok: %w", err)
	}

	s.logger.Info("Created knok record",
		"knok_id", knokID,
		"url", urlInfo.URL,
		"platform", urlInfo.Platform,
	)
	stats.KnoksCreated++

	// Queue metadata extraction job
	jobPayload := map[string]interface{}{
		"knok_id":            knokID.String(),
		"url":                urlInfo.URL,
		"platform":           urlInfo.Platform,
		"discord_message_id": message.ID,
		"discord_channel_id": message.ChannelID,
		"discord_guild_id":   s.guildID,
		"discord_user_id":    message.Author.ID,
		"message_content":    message.Content,
	}

	if err := s.queueRepo.Enqueue(ctx, domain.JobTypeExtractMetadata, jobPayload); err != nil {
		return fmt.Errorf("failed to queue metadata extraction job: %w", err)
	}

	s.logger.Debug("Queued metadata extraction job",
		"knok_id", knokID,
		"url", urlInfo.URL,
	)
	stats.JobsQueued++

	return nil
}

// SeedingStats tracks statistics for the seeding process
type SeedingStats struct {
	MessagesProcessed int
	URLsDetected      int
	KnoksCreated      int
	KnoksSkipped      int
	JobsQueued        int
	Errors            int
}

// cleanMarkdownLinks removes markdown link formatting from Discord messages
// Converts [text](url) to just url and cleans Unicode characters
func cleanMarkdownLinks(content string) string {
	// Regex to match markdown links: [text](url)
	markdownLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)

	// Replace [text](url) with just the url
	cleaned := markdownLinkRegex.ReplaceAllString(content, "$2")

	// Remove angle brackets around URLs (Discord sometimes adds these)
	cleaned = regexp.MustCompile(`<([^>]+)>`).ReplaceAllString(cleaned, "$1")

	// Remove common emojis before URLs (simpler approach - just remove known emoji patterns)
	// Match any non-ASCII characters followed by http to catch emojis before URLs
	cleaned = regexp.MustCompile(`[^\x00-\x7F]+\s*(https?://)`).ReplaceAllString(cleaned, "$1")

	// Clean common Unicode characters that break URLs
	// Em-dash (—) and en-dash (–) before URLs
	cleaned = regexp.MustCompile(`[—–]\s*`).ReplaceAllString(cleaned, "")

	// Zero-width spaces and other invisible characters (using actual Unicode characters)
	// U+200B: Zero Width Space
	// U+200C: Zero Width Non-Joiner
	// U+200D: Zero Width Joiner
	// U+FEFF: Zero Width No-Break Space (BOM)
	cleaned = strings.ReplaceAll(cleaned, "\u200B", "")
	cleaned = strings.ReplaceAll(cleaned, "\u200C", "")
	cleaned = strings.ReplaceAll(cleaned, "\u200D", "")
	cleaned = strings.ReplaceAll(cleaned, "\uFEFF", "")

	// Expand YouTube shortened URLs
	cleaned = strings.ReplaceAll(cleaned, "youtu.be/", "youtube.com/watch?v=")

	// Clean any remaining whitespace around URLs
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// isRootURL checks if a URL is just a root domain without specific content
// Returns true for URLs like https://www.nts.live or https://soundcloud.com
// Returns false for URLs with meaningful paths like https://www.nts.live/shows/...
func isRootURL(urlStr string) bool {
	// Remove protocol
	withoutProtocol := strings.TrimPrefix(urlStr, "https://")
	withoutProtocol = strings.TrimPrefix(withoutProtocol, "http://")

	// Remove www. prefix
	withoutProtocol = strings.TrimPrefix(withoutProtocol, "www.")

	// Remove trailing slash
	withoutProtocol = strings.TrimSuffix(withoutProtocol, "/")

	// Check if there's a meaningful path (more than just domain)
	// Split by / and check if there are meaningful path segments
	parts := strings.Split(withoutProtocol, "/")

	// If only domain (no path segments), it's a root URL
	if len(parts) == 1 {
		return true
	}

	// If path is empty or just "/" it's a root URL
	if len(parts) == 2 && parts[1] == "" {
		return true
	}

	return false
}

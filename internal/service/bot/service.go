package bot

import (
	"context"
	"fmt"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// BotService handles Discord bot operations
type BotService struct {
	config     *config.Config
	logger     *slog.Logger
	session    *discordgo.Session
	queueRepo  domain.QueueRepository
	knokRepo   domain.KnokRepository // Optional - for storing knoks
	serverRepo domain.ServerRepository

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new bot service
func New(
	config *config.Config,
	logger *slog.Logger,
	queueRepo domain.QueueRepository,
	knokRepo domain.KnokRepository, // Optional - can be nil
	serverRepo domain.ServerRepository,
) (*BotService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	botService := &BotService{
		config:     config,
		logger:     logger,
		queueRepo:  queueRepo,
		knokRepo:   knokRepo,
		serverRepo: serverRepo,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		cancel()
		return nil, err
	}

	botService.session = session

	// Register handlers
	botService.registerHandlers()

	return botService, nil
}

func (s *BotService) Start() error {
	s.logger.Info("Starting Discord bot...")

	// Open connection to Discord
	if err := s.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	s.logger.Info("Discord bot connected successfully")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	s.logger.Info("Bot is running. Press Ctrl+C to stop.")
	<-stop

	s.logger.Info("Shutting down Discord bot...")
	return s.Stop()
}

func (s *BotService) Stop() error {
	if s.session != nil {
		s.logger.Info("Closing Discord connection...")
		if err := s.session.Close(); err != nil {
			s.logger.Error("Error closing Discord connection", "error", err)
			return err
		}
	}

	s.logger.Info("Discord bot stopped")
	return nil
}

func (s *BotService) registerHandlers() {
	s.session.AddHandler(s.onReady)
	s.session.AddHandler(s.onMessageCreate)
	s.session.AddHandler(s.onInteractionCreate)

}

// onReady is called when the bot successfully connects to Discord
func (s *BotService) onReady(session *discordgo.Session, ready *discordgo.Ready) {
	s.logger.Info("Bot is ready",
		"username", ready.User.Username,
		"discriminator", ready.User.Discriminator,
		"guilds", len(ready.Guilds),
	)

	// Register commands now that bot is connected
	if err := s.registerCommands(); err != nil {
		s.logger.Error("Failed to register slash commands", "error", err)
	} else {
		s.logger.Info("Slash commands registered successfully")
	}

	// Set bot status
	err := session.UpdateGameStatus(0, "🎵 Listening for music")
	if err != nil {
		s.logger.Error("Failed to set bot status", "error", err)
	}
}
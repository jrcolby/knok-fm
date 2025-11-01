package api

import (
	"context"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	knokhttp "knock-fm/internal/http"
	"knock-fm/internal/http/handlers"
	"log/slog"
	"net/http"
	"time"
)

// PlatformLoader defines the interface for platform cache management
type PlatformLoader interface {
	Refresh(ctx context.Context) error
	GetAll() ([]*domain.Platform, error)
	Count() int
}

// APIService handles HTTP API requests
type APIService struct {
	config         *config.Config
	logger         *slog.Logger
	router         *knokhttp.Router
	knokRepo       domain.KnokRepository
	serverRepo     domain.ServerRepository
	queueRepo      domain.QueueRepository
	platformRepo   handlers.PlatformRepository
	platformLoader PlatformLoader

	// HTTP server
	server *http.Server
}

// New creates a new API service
func New(
	config *config.Config,
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
	queueRepo domain.QueueRepository,
	platformRepo handlers.PlatformRepository,
	platformLoader PlatformLoader,
) (*APIService, error) {
	router := knokhttp.NewRouter(logger, serverRepo, knokRepo, queueRepo, platformRepo, platformLoader)

	apiService := &APIService{
		config:         config,
		logger:         logger,
		router:         router,
		knokRepo:       knokRepo,
		serverRepo:     serverRepo,
		queueRepo:      queueRepo,
		platformRepo:   platformRepo,
		platformLoader: platformLoader,
	}

	// Create HTTP server with router and middleware
	handler := router.SetupRoutes()
	apiService.server = &http.Server{
		Addr:         ":" + config.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return apiService, nil
}

// Start begins serving the API
func (s *APIService) Start() error {
	s.logger.Info("Starting API server", "port", s.config.Port)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the API server
func (s *APIService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server...")
	return s.server.Shutdown(ctx)
}

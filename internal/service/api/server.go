package api

import (
	"context"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	knokhttp "knock-fm/internal/http"
	"log/slog"
	"net/http"
	"time"
)

// APIService handles HTTP API requests
type APIService struct {
	config     *config.Config
	logger     *slog.Logger
	router     *knokhttp.Router
	knokRepo   domain.KnokRepository
	serverRepo domain.ServerRepository

	// HTTP server
	server *http.Server
}

// New creates a new API service
func New(
	config *config.Config,
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
) (*APIService, error) {
	router := knokhttp.NewRouter(logger, serverRepo, knokRepo)

	apiService := &APIService{
		config:     config,
		logger:     logger,
		router:     router,
		knokRepo:   knokRepo,
		serverRepo: serverRepo,
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

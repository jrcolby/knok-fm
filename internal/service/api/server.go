package api

import (
	"context"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// APIService handles HTTP API requests
type APIService struct {
	config     *config.Config
	logger     *slog.Logger
	router     *http.ServeMux
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
	router := http.NewServeMux()

	apiService := &APIService{
		config:     config,
		logger:     logger,
		router:     router,
		knokRepo:   knokRepo,
		serverRepo: serverRepo,
	}

	// Create HTTP server
	apiService.server = &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup routes
	apiService.setupRoutes()

	return apiService, nil
}

// setupRoutes configures all API routes
func (s *APIService) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth)

	// API v1 routes
	s.router.HandleFunc("/api/v1/servers", s.handleServers)
	s.router.HandleFunc("/api/v1/servers/", s.handleServerByID)
	s.router.HandleFunc("/api/v1/stats", s.handleStats)

	// Add CORS middleware
	s.server.Handler = s.corsMiddleware(s.router)
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

// handleHealth handles health check requests
func (s *APIService) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

// handleServers handles server-related requests
func (s *APIService) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getServers(w, r)
	case http.MethodPost:
		s.createServer(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleServerByID handles individual server requests
func (s *APIService) handleServerByID(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from path
	serverID := strings.TrimPrefix(r.URL.Path, "/api/v1/servers/")
	if serverID == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getServerByID(w, r, serverID)
	case http.MethodPut:
		s.updateServer(w, r, serverID)
	case http.MethodDelete:
		s.deleteServer(w, r, serverID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStats handles statistics requests
func (s *APIService) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `","message":"Stats endpoint coming soon"}`))
}

// getServers retrieves all servers
func (s *APIService) getServers(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement server listing
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// createServer creates a new server
func (s *APIService) createServer(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement server creation
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// getServerByID retrieves a server by ID
func (s *APIService) getServerByID(w http.ResponseWriter, r *http.Request, serverID string) {
	// TODO: Implement server retrieval
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// updateServer updates an existing server
func (s *APIService) updateServer(w http.ResponseWriter, r *http.Request, serverID string) {
	// TODO: Implement server update
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// deleteServer deletes a server
func (s *APIService) deleteServer(w http.ResponseWriter, r *http.Request, serverID string) {
	// TODO: Implement server deletion
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// corsMiddleware adds CORS headers to responses
func (s *APIService) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

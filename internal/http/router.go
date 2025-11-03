package http

import (
	"context"
	"knock-fm/internal/domain"
	"knock-fm/internal/http/handlers"
	"knock-fm/internal/http/middleware"
	"log/slog"
	"net/http"
)

// PlatformLoader defines the interface for platform cache management
type PlatformLoader interface {
	Refresh(ctx context.Context) error
	GetAll() ([]*domain.Platform, error)
	Count() int
}

type Router struct {
	mux                  *http.ServeMux
	logger               *slog.Logger
	healthHandler        *handlers.HealthHandler
	statsHandler         *handlers.StatsHandler
	serversHandler       *handlers.ServersHandler
	knoksHandler         *handlers.KnoksHandler
	adminPlatformHandler *handlers.AdminPlatformHandler
	adminAuth            *middleware.AdminAuth
}

func NewRouter(
	logger *slog.Logger,
	serverRepo domain.ServerRepository,
	knokRepo domain.KnokRepository,
	queueRepo domain.QueueRepository,
	platformRepo handlers.PlatformRepository,
	platformLoader PlatformLoader,
) *Router {
	mux := http.NewServeMux()

	return &Router{
		mux:                  mux,
		logger:               logger,
		healthHandler:        handlers.NewHealthHandler(logger),
		statsHandler:         handlers.NewStatsHandler(logger),
		serversHandler:       handlers.NewServersHandler(logger, serverRepo),
		knoksHandler:         handlers.NewKnoksHandler(logger, knokRepo, queueRepo),
		adminPlatformHandler: handlers.NewAdminPlatformHandler(platformRepo, platformLoader, logger),
		adminAuth:            middleware.NewAdminAuth(logger),
	}
}

func (r *Router) SetupRoutes() http.Handler {
	// Health check
	r.mux.HandleFunc("GET /health", r.healthHandler.HandleHealth)

	// API v1 routes - Server management
	r.mux.HandleFunc("GET /api/v1/servers", r.serversHandler.GetServers)
	r.mux.HandleFunc("POST /api/v1/servers", r.serversHandler.CreateServer)
	r.mux.HandleFunc("GET /api/v1/servers/{id}", r.serversHandler.GetServerByID)
	r.mux.HandleFunc("PUT /api/v1/servers/{id}", r.serversHandler.UpdateServer)
	r.mux.HandleFunc("DELETE /api/v1/servers/{id}", r.serversHandler.DeleteServer)

	// API v1 routes - Stats
	r.mux.HandleFunc("GET /api/v1/stats", r.statsHandler.HandleStats)

	// API v1 routes - Get recent knoks (global and per-server)
	r.mux.HandleFunc("GET /api/v1/knoks", r.knoksHandler.GetKnoks)                       // Global timeline
	r.mux.HandleFunc("GET /api/v1/knoks/server/{serverId}", r.knoksHandler.GetKnoksByServer) // Server-specific
	r.mux.HandleFunc("GET /api/v1/knoks/search", r.knoksHandler.SearchKnoks)
	r.mux.HandleFunc("GET /api/v1/knoks/random", r.knoksHandler.GetRandomKnok)

	// API v1 routes - Admin endpoints for managing knoks (protected by auth middleware)
	r.mux.Handle("DELETE /api/v1/admin/knoks/{id}", r.adminAuth.Middleware(http.HandlerFunc(r.knoksHandler.DeleteKnok)))
	r.mux.Handle("PATCH /api/v1/admin/knoks/{id}", r.adminAuth.Middleware(http.HandlerFunc(r.knoksHandler.UpdateKnok)))
	r.mux.Handle("POST /api/v1/admin/knoks/{id}/refresh", r.adminAuth.Middleware(http.HandlerFunc(r.knoksHandler.RefreshKnok)))

	// Admin platform management endpoints (protected by auth middleware)
	r.mux.Handle("GET /api/v1/admin/platforms", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.ListPlatforms)))
	r.mux.Handle("POST /api/v1/admin/platforms", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.CreatePlatform)))
	r.mux.Handle("PUT /api/v1/admin/platforms/{id}", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.UpdatePlatform)))
	r.mux.Handle("PATCH /api/v1/admin/platforms/{id}", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.PatchPlatform)))
	r.mux.Handle("DELETE /api/v1/admin/platforms/{id}", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.DeletePlatform)))
	r.mux.Handle("POST /api/v1/admin/platforms/refresh", r.adminAuth.Middleware(http.HandlerFunc(r.adminPlatformHandler.RefreshCache)))

	// Add CORS middleware
	return middleware.CORS(r.mux)
}

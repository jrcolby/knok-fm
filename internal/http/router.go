package http

import (
	"knock-fm/internal/domain"
	"knock-fm/internal/http/handlers"
	"knock-fm/internal/http/middleware"
	"log/slog"
	"net/http"
)

type Router struct {
	mux            *http.ServeMux
	healthHandler  *handlers.HealthHandler
	statsHandler   *handlers.StatsHandler
	serversHandler *handlers.ServersHandler
	knoksHandler   *handlers.KnoksHandler
}

func NewRouter(logger *slog.Logger, serverRepo domain.ServerRepository, knokRepo domain.KnokRepository) *Router {
	mux := http.NewServeMux()

	return &Router{
		mux:            mux,
		healthHandler:  handlers.NewHealthHandler(logger),
		statsHandler:   handlers.NewStatsHandler(logger),
		serversHandler: handlers.NewServersHandler(logger, serverRepo),
		knoksHandler:   handlers.NewKnoksHandler(logger, knokRepo),
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

	// API v1 routes - Get recent knoks, search knoks, all for one server
	r.mux.HandleFunc("GET /api/v1/knoks/server/{serverId}", r.knoksHandler.GetKnoksByServer)
	r.mux.HandleFunc("GET /api/v1/knoks/search", r.knoksHandler.SearchKnoks)
	r.mux.HandleFunc("GET /api/v1/knoks/random", r.knoksHandler.GetRandomKnok)
	// Add CORS middleware
	return middleware.CORS(r.mux)
}

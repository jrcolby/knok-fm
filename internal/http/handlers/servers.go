package handlers

import (
	"encoding/json"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"strconv"
)

type ServersHandler struct {
	logger     *slog.Logger
	serverRepo domain.ServerRepository
}

func NewServersHandler(logger *slog.Logger, serverRepo domain.ServerRepository) *ServersHandler {
	return &ServersHandler{
		logger:     logger,
		serverRepo: serverRepo,
	}
}

func (h *ServersHandler) GetServers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	offset := 0
	limit := 50 // Default limit

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get servers from repository
	servers, total, err := h.serverRepo.List(ctx, offset, limit)
	if err != nil {
		h.logger.Error("Failed to retrieve servers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"servers": servers,
		"pagination": map[string]interface{}{
			"offset": offset,
			"limit":  limit,
			"total":  total,
		},
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode servers response", "error", err)
	}
}

func (h *ServersHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement server creation
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *ServersHandler) GetServerByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract server ID from path parameter
	serverID := r.PathValue("id")
	if serverID == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	// Get server from repository
	server, err := h.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		h.logger.Error("Failed to retrieve server", "error", err, "server_id", serverID)
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(server); err != nil {
		h.logger.Error("Failed to encode server response", "error", err, "server_id", serverID)
	}
}

func (h *ServersHandler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement server update
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *ServersHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement server deletion
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

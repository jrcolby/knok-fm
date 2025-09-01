package handlers

import (
	"encoding/json"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type KnoksHandler struct {
	logger   *slog.Logger
	knokRepo domain.KnokRepository
}

// KnoksResponse represents the paginated response for knoks
type KnoksResponse struct {
	Knoks   []*KnokDto	   `json:"knoks"`
	HasMore bool           `json:"has_more"`
	Cursor  *string        `json:"cursor,omitempty"`
}

type KnokDto struct {
	Title 		string		`json:"title"`
	PostedAt	time.Time 	`json:"posted_at"`
	ID			string		`json:"id"`
	URL 		string 		`json:"url"`
}

func NewKnoksHandler(logger *slog.Logger, knokRepo domain.KnokRepository) *KnoksHandler {
	return &KnoksHandler{
		logger:   logger,
		knokRepo: knokRepo,
	}
}

func (h *KnoksHandler) GetKnoksByServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get server ID from path
	serverID := r.PathValue("serverId")
	if serverID == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	limit := 25 // Default limit for infinite scroll
	var cursor *time.Time

	// Parse cursor parameter
	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		if parsed, err := time.Parse(time.RFC3339, cursorStr); err == nil {
			cursor = &parsed
		} else {
			h.logger.Warn("Invalid cursor format", "cursor", cursorStr, "error", err)
			http.Error(w, "Invalid cursor format", http.StatusBadRequest)
			return
		}
	}

	// Parse limit parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Request one more item than the limit to determine if there are more results
	knoks, err := h.knokRepo.GetRecentByServer(ctx, serverID, cursor, limit+1)
	if err != nil {
		h.logger.Error("Failed to retrieve knoks", "error", err, "server_id", serverID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Determine if there are more results
	hasMore := len(knoks) > limit
	if hasMore {
		// Remove the extra knok that was fetched to check for more results
		knoks = knoks[:limit]
	}

	knokDtos := make([]*KnokDto, 0, len(knoks))
	for _, knok := range knoks {
		knokDtos = append(knokDtos, &KnokDto{
			Title: *knok.Title,
			PostedAt: knok.PostedAt,
			ID: knok.ID.String(),
			URL: knok.URL,
		})
	}
	h.logger.Info("Retrieved knoks", "count", len(knokDtos), "server_id", serverID, "has_more", hasMore)
	// Prepare response
	response := KnoksResponse{
		Knoks:   knokDtos,
		HasMore: hasMore,
	}

	// Set next cursor if there are more results
	if hasMore && len(knoks) > 0 {
		// Use the posted_at of the last knok as the next cursor
		lastKnok := knoks[len(knoks)-1]
		cursorStr := lastKnok.PostedAt.Format(time.RFC3339)
		response.Cursor = &cursorStr
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

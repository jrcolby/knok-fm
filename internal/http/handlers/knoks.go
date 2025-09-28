package handlers

import (
	"encoding/json"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPaginationLimit = 25
)

type KnoksHandler struct {
	logger   *slog.Logger
	knokRepo domain.KnokRepository
}

// KnoksResponse represents the paginated response for knoks
type KnoksResponse struct {
	Knoks   []*KnokDto `json:"knoks"`
	HasMore bool       `json:"has_more"`
	Cursor  *string    `json:"cursor,omitempty"`
}

type KnokDto struct {
	Title    string                 `json:"title"`
	PostedAt time.Time              `json:"posted_at"`
	ID       string                 `json:"id"`
	URL      string                 `json:"url"`
	Metadata map[string]interface{} `json:"metadata"`
}

func NewKnoksHandler(logger *slog.Logger, knokRepo domain.KnokRepository) *KnoksHandler {
	return &KnoksHandler{
		logger:   logger,
		knokRepo: knokRepo,
	}
}

// parseCursor parses a cursor string into a time.Time pointer
func (h *KnoksHandler) parseCursor(cursorStr string) (*time.Time, error) {
	if cursorStr == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, cursorStr)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// buildKnokResponse creates paginated response from domain knoks
func (h *KnoksHandler) buildKnokResponse(knoks []*domain.Knok, requestedLimit int) *KnoksResponse {
	// Determine if there are more results
	hasMore := len(knoks) > requestedLimit
	if hasMore {
		// Remove the extra knok that was fetched to check for more results
		knoks = knoks[:requestedLimit]
	}

	knokDtos := make([]*KnokDto, 0, len(knoks))
	for _, knok := range knoks {
		// Handle nil title gracefully (happens when extraction is still processing)
		title := "Processing..."
		if knok.Title != nil {
			title = *knok.Title
		}
		
		knokDtos = append(knokDtos, &KnokDto{
			Title:    title,
			PostedAt: knok.PostedAt,
			ID:       knok.ID.String(),
			URL:      knok.URL,
			Metadata: knok.Metadata,
		})
	}

	response := &KnoksResponse{
		Knoks:   knokDtos,
		HasMore: hasMore,
	}

	// Set next cursor if there are more results
	if hasMore && len(knoks) > 0 {
		lastKnok := knoks[len(knoks)-1]
		cursorStr := lastKnok.PostedAt.Format(time.RFC3339)
		response.Cursor = &cursorStr
	}

	return response
}

// writeJSONResponse writes a JSON response to the ResponseWriter
func (h *KnoksHandler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *KnoksHandler) GetRandomKnok(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	knok, err := h.knokRepo.GetRandom(ctx)
	if err != nil {
		h.logger.Error("Failed to retrieve knok", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Handle nil title gracefully (should not happen with completed knoks)
	title := "Processing..."
	if knok.Title != nil {
		title = *knok.Title
	}
	
	response := &KnokDto{
		Title:    title,
		PostedAt: knok.PostedAt,
		ID:       knok.ID.String(),
		URL:      knok.URL,
		Metadata: knok.Metadata,
	}
	h.logger.Info("Retrieved random knok", "title", response.Title)
	h.writeJSONResponse(w, response)

}
func (h *KnoksHandler) SearchKnoks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get and validate search query
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	// Trim whitespace and limit length
	query = strings.TrimSpace(query)
	if len(query) > 500 { // Reasonable search term limit
		http.Error(w, "Search query too long (max 500 characters)", http.StatusBadRequest)
		return
	}
	// Parse pagination parameters
	limit := DefaultPaginationLimit

	// Parse cursor parameter
	cursor, err := h.parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.logger.Warn("Invalid cursor format", "cursor", r.URL.Query().Get("cursor"), "error", err)
		http.Error(w, "Invalid cursor format", http.StatusBadRequest)
		return
	}

	// Parse limit parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Request one more item than the limit to determine if there are more results
	knoks, err := h.knokRepo.Search(ctx, query, cursor, limit+1)
	if err != nil {
		h.logger.Error("Failed to retrieve knoks", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := h.buildKnokResponse(knoks, limit)
	h.logger.Info("Search completed", "query", query, "count", len(response.Knoks), "has_more", response.HasMore)
	h.writeJSONResponse(w, response)
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
	limit := DefaultPaginationLimit

	// Parse cursor parameter
	cursor, err := h.parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.logger.Warn("Invalid cursor format", "cursor", r.URL.Query().Get("cursor"), "error", err)
		http.Error(w, "Invalid cursor format", http.StatusBadRequest)
		return
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

	response := h.buildKnokResponse(knoks, limit)
	h.logger.Info("Retrieved knoks", "count", len(response.Knoks), "server_id", serverID, "has_more", response.HasMore)
	h.writeJSONResponse(w, response)
}

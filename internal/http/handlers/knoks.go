package handlers

import (
	"encoding/json"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPaginationLimit = 25
)

type KnoksHandler struct {
	logger    *slog.Logger
	knokRepo  domain.KnokRepository
	queueRepo domain.QueueRepository
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

func NewKnoksHandler(logger *slog.Logger, knokRepo domain.KnokRepository, queueRepo domain.QueueRepository) *KnoksHandler {
	return &KnoksHandler{
		logger:    logger,
		knokRepo:  knokRepo,
		queueRepo: queueRepo,
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

// GetKnoks handles GET /api/v1/knoks - global timeline across all servers
func (h *KnoksHandler) GetKnoks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	knoks, err := h.knokRepo.GetRecent(ctx, cursor, limit+1)
	if err != nil {
		h.logger.Error("Failed to retrieve knoks (global)", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := h.buildKnokResponse(knoks, limit)
	h.logger.Info("Retrieved knoks (global)", "count", len(response.Knoks), "has_more", response.HasMore)
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

// DeleteKnok handles DELETE /api/knoks/:id
func (h *KnoksHandler) DeleteKnok(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get knok ID from path
	knokIDStr := r.PathValue("id")
	if knokIDStr == "" {
		http.Error(w, "Knok ID is required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	knokID, err := uuid.Parse(knokIDStr)
	if err != nil {
		h.logger.Warn("Invalid knok ID format", "id", knokIDStr, "error", err)
		http.Error(w, "Invalid knok ID format", http.StatusBadRequest)
		return
	}

	// Check if knok exists
	knok, err := h.knokRepo.GetByID(ctx, knokID)
	if err != nil {
		h.logger.Error("Failed to get knok", "error", err, "knok_id", knokID)
		http.Error(w, "Knok not found", http.StatusNotFound)
		return
	}

	// Delete the knok
	if err := h.knokRepo.Delete(ctx, knokID); err != nil {
		h.logger.Error("Failed to delete knok", "error", err, "knok_id", knokID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Knok deleted successfully", "knok_id", knokID, "url", knok.URL)
	w.WriteHeader(http.StatusNoContent)
}

// UpdateKnokRequest represents the request body for updating a knok
type UpdateKnokRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateKnok handles PATCH /api/knoks/:id
func (h *KnoksHandler) UpdateKnok(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get knok ID from path
	knokIDStr := r.PathValue("id")
	if knokIDStr == "" {
		http.Error(w, "Knok ID is required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	knokID, err := uuid.Parse(knokIDStr)
	if err != nil {
		h.logger.Warn("Invalid knok ID format", "id", knokIDStr, "error", err)
		http.Error(w, "Invalid knok ID format", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req UpdateKnokRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if knok exists
	knok, err := h.knokRepo.GetByID(ctx, knokID)
	if err != nil {
		h.logger.Error("Failed to get knok", "error", err, "knok_id", knokID)
		http.Error(w, "Knok not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	updated := false
	if req.Title != nil {
		knok.Title = req.Title
		updated = true
	}

	if req.Description != nil {
		// Update description in metadata
		if knok.Metadata == nil {
			knok.Metadata = make(map[string]interface{})
		}
		knok.Metadata["description"] = *req.Description
		updated = true
	}

	if !updated {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Update the knok
	if err := h.knokRepo.Update(ctx, knok); err != nil {
		h.logger.Error("Failed to update knok", "error", err, "knok_id", knokID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Knok updated successfully", "knok_id", knokID, "title", knok.Title)

	// Return updated knok
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

	h.writeJSONResponse(w, response)
}

// RefreshKnokRequest represents the request body for refreshing a knok's metadata
type RefreshKnokRequest struct {
	URL *string `json:"url,omitempty"`
}

// RefreshKnok handles POST /api/v1/admin/knoks/:id/refresh
func (h *KnoksHandler) RefreshKnok(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get knok ID from path
	knokIDStr := r.PathValue("id")
	if knokIDStr == "" {
		http.Error(w, "Knok ID is required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	knokID, err := uuid.Parse(knokIDStr)
	if err != nil {
		h.logger.Warn("Invalid knok ID format", "id", knokIDStr, "error", err)
		http.Error(w, "Invalid knok ID format", http.StatusBadRequest)
		return
	}

	// Parse request body (optional URL)
	var req RefreshKnokRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if knok exists
	knok, err := h.knokRepo.GetByID(ctx, knokID)
	if err != nil {
		h.logger.Error("Failed to get knok", "error", err, "knok_id", knokID)
		http.Error(w, "Knok not found", http.StatusNotFound)
		return
	}

	// Update URL if new one provided
	urlToUse := knok.URL
	if req.URL != nil && *req.URL != "" {
		urlToUse = *req.URL
		knok.URL = urlToUse
		h.logger.Info("Updating knok URL", "knok_id", knokID, "old_url", knok.URL, "new_url", urlToUse)
	} else {
		h.logger.Info("Refreshing knok with existing URL", "knok_id", knokID, "url", urlToUse)
	}

	// Set extraction status to pending
	knok.ExtractionStatus = domain.ExtractionStatusPending

	// Update the knok in database
	if err := h.knokRepo.Update(ctx, knok); err != nil {
		h.logger.Error("Failed to update knok", "error", err, "knok_id", knokID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Queue metadata extraction job
	jobPayload := map[string]interface{}{
		"knok_id":  knokID.String(),
		"url":      urlToUse,
		"platform": knok.Platform,
	}

	if err := h.queueRepo.Enqueue(ctx, domain.JobTypeExtractMetadata, jobPayload); err != nil {
		h.logger.Error("Failed to queue metadata extraction job",
			"error", err,
			"knok_id", knokID,
			"url", urlToUse,
		)

		// Rollback extraction status to failed
		knok.ExtractionStatus = domain.ExtractionStatusFailed
		h.knokRepo.Update(ctx, knok)

		http.Error(w, "Failed to queue metadata extraction job", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Knok refresh initiated successfully",
		"knok_id", knokID,
		"url", urlToUse,
		"status", knok.ExtractionStatus,
	)

	// Return updated knok
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

	h.writeJSONResponse(w, response)
}

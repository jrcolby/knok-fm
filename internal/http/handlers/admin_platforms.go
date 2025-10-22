package handlers

import (
	"context"
	"encoding/json"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"time"
)

// PlatformRepository defines the interface for platform data access
type PlatformRepository interface {
	CreatePlatform(ctx context.Context, platform *domain.Platform) error
	UpdatePlatform(ctx context.Context, platform *domain.Platform) error
	DeletePlatform(ctx context.Context, id string) error
	GetAllPlatforms(ctx context.Context) ([]*domain.Platform, error)
}

// PlatformLoader defines the interface for platform cache management
type PlatformLoader interface {
	Refresh(ctx context.Context) error
	GetAll() ([]*domain.Platform, error)
	Count() int
}

// AdminPlatformHandler handles admin operations for platform management
type AdminPlatformHandler struct {
	platformRepo   PlatformRepository
	platformLoader PlatformLoader
	logger         *slog.Logger
}

// NewAdminPlatformHandler creates a new admin platform handler
func NewAdminPlatformHandler(
	platformRepo PlatformRepository,
	platformLoader PlatformLoader,
	logger *slog.Logger,
) *AdminPlatformHandler {
	return &AdminPlatformHandler{
		platformRepo:   platformRepo,
		platformLoader: platformLoader,
		logger:         logger,
	}
}

// CreatePlatformRequest represents the request body for creating a platform
type CreatePlatformRequest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	URLPatterns []string `json:"url_patterns"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
}

// UpdatePlatformRequest represents the request body for updating a platform
type UpdatePlatformRequest struct {
	Name        string   `json:"name"`
	URLPatterns []string `json:"url_patterns"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
}

// PatchPlatformRequest represents the request body for partial updates
type PatchPlatformRequest struct {
	Name        *string   `json:"name,omitempty"`
	URLPatterns *[]string `json:"url_patterns,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

// PlatformResponse represents the response for platform operations
type PlatformResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URLPatterns []string  `json:"url_patterns"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreatePlatform handles POST /api/admin/platforms
func (h *AdminPlatformHandler) CreatePlatform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreatePlatformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(req.URLPatterns) == 0 {
		http.Error(w, "url_patterns must contain at least one pattern", http.StatusBadRequest)
		return
	}

	// Create platform
	now := time.Now()
	platform := &domain.Platform{
		ID:          req.ID,
		Name:        req.Name,
		URLPatterns: req.URLPatterns,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		CreatedAt:   now,
		UpdatedAt:   &now,
	}

	if err := h.platformRepo.CreatePlatform(ctx, platform); err != nil {
		h.logger.Error("Failed to create platform",
			"error", err,
			"id", req.ID,
		)
		http.Error(w, "Failed to create platform: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("Platform created via admin API",
		"id", platform.ID,
		"name", platform.Name,
		"patterns", len(platform.URLPatterns),
	)

	// Refresh platform loader cache
	if err := h.platformLoader.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh platform cache after create", "error", err)
	}

	// Return created platform
	response := PlatformResponse{
		ID:          platform.ID,
		Name:        platform.Name,
		URLPatterns: platform.URLPatterns,
		Priority:    platform.Priority,
		Enabled:     platform.Enabled,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   *platform.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdatePlatform handles PUT /api/admin/platforms/:id
func (h *AdminPlatformHandler) UpdatePlatform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract platform ID from URL
	platformID := r.PathValue("id")

	if platformID == "" {
		http.Error(w, "platform id is required", http.StatusBadRequest)
		return
	}

	var req UpdatePlatformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(req.URLPatterns) == 0 {
		http.Error(w, "url_patterns must contain at least one pattern", http.StatusBadRequest)
		return
	}

	// Get existing platform to preserve created_at
	existing, err := h.platformLoader.GetAll()
	if err != nil {
		http.Error(w, "Failed to get existing platform", http.StatusInternalServerError)
		return
	}

	var existingPlatform *domain.Platform
	for _, p := range existing {
		if p.ID == platformID {
			existingPlatform = p
			break
		}
	}

	if existingPlatform == nil {
		http.Error(w, "Platform not found", http.StatusNotFound)
		return
	}

	// Update platform
	now := time.Now()
	platform := &domain.Platform{
		ID:          platformID,
		Name:        req.Name,
		URLPatterns: req.URLPatterns,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		CreatedAt:   existingPlatform.CreatedAt,
		UpdatedAt:   &now,
	}

	if err := h.platformRepo.UpdatePlatform(ctx, platform); err != nil {
		h.logger.Error("Failed to update platform",
			"error", err,
			"id", platformID,
		)
		http.Error(w, "Failed to update platform: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("Platform updated via admin API",
		"id", platform.ID,
		"name", platform.Name,
	)

	// Refresh platform loader cache
	if err := h.platformLoader.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh platform cache after update", "error", err)
	}

	// Return updated platform
	response := PlatformResponse{
		ID:          platform.ID,
		Name:        platform.Name,
		URLPatterns: platform.URLPatterns,
		Priority:    platform.Priority,
		Enabled:     platform.Enabled,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   *platform.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PatchPlatform handles PATCH /api/admin/platforms/:id
func (h *AdminPlatformHandler) PatchPlatform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract platform ID from URL
	platformID := r.PathValue("id")

	if platformID == "" {
		http.Error(w, "platform id is required", http.StatusBadRequest)
		return
	}

	var req PatchPlatformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing platform
	existing, err := h.platformLoader.GetAll()
	if err != nil {
		http.Error(w, "Failed to get existing platform", http.StatusInternalServerError)
		return
	}

	var existingPlatform *domain.Platform
	for _, p := range existing {
		if p.ID == platformID {
			existingPlatform = p
			break
		}
	}

	if existingPlatform == nil {
		http.Error(w, "Platform not found", http.StatusNotFound)
		return
	}

	// Apply partial updates
	if req.Name != nil {
		existingPlatform.Name = *req.Name
	}
	if req.URLPatterns != nil {
		if len(*req.URLPatterns) == 0 {
			http.Error(w, "url_patterns must contain at least one pattern", http.StatusBadRequest)
			return
		}
		existingPlatform.URLPatterns = *req.URLPatterns
	}
	if req.Priority != nil {
		existingPlatform.Priority = *req.Priority
	}
	if req.Enabled != nil {
		existingPlatform.Enabled = *req.Enabled
	}

	// Update timestamp
	now := time.Now()
	existingPlatform.UpdatedAt = &now

	if err := h.platformRepo.UpdatePlatform(ctx, existingPlatform); err != nil {
		h.logger.Error("Failed to patch platform",
			"error", err,
			"id", platformID,
		)
		http.Error(w, "Failed to update platform: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("Platform patched via admin API",
		"id", existingPlatform.ID,
		"name", existingPlatform.Name,
	)

	// Refresh platform loader cache
	if err := h.platformLoader.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh platform cache after patch", "error", err)
	}

	// Return updated platform
	response := PlatformResponse{
		ID:          existingPlatform.ID,
		Name:        existingPlatform.Name,
		URLPatterns: existingPlatform.URLPatterns,
		Priority:    existingPlatform.Priority,
		Enabled:     existingPlatform.Enabled,
		CreatedAt:   existingPlatform.CreatedAt,
		UpdatedAt:   *existingPlatform.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePlatform handles DELETE /api/admin/platforms/:id (soft delete)
func (h *AdminPlatformHandler) DeletePlatform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract platform ID from URL
	platformID := r.PathValue("id")

	if platformID == "" {
		http.Error(w, "platform id is required", http.StatusBadRequest)
		return
	}

	// Get existing platform
	existing, err := h.platformLoader.GetAll()
	if err != nil {
		http.Error(w, "Failed to get existing platform", http.StatusInternalServerError)
		return
	}

	var existingPlatform *domain.Platform
	for _, p := range existing {
		if p.ID == platformID {
			existingPlatform = p
			break
		}
	}

	if existingPlatform == nil {
		http.Error(w, "Platform not found", http.StatusNotFound)
		return
	}

	// Soft delete: Set enabled = false
	now := time.Now()
	existingPlatform.Enabled = false
	existingPlatform.UpdatedAt = &now

	if err := h.platformRepo.UpdatePlatform(ctx, existingPlatform); err != nil {
		h.logger.Error("Failed to delete platform",
			"error", err,
			"id", platformID,
		)
		http.Error(w, "Failed to delete platform: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("Platform soft deleted via admin API",
		"id", platformID,
	)

	// Refresh platform loader cache
	if err := h.platformLoader.Refresh(ctx); err != nil {
		h.logger.Warn("Failed to refresh platform cache after delete", "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// RefreshCache handles POST /api/admin/platforms/refresh
func (h *AdminPlatformHandler) RefreshCache(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.platformLoader.Refresh(ctx); err != nil {
		h.logger.Error("Failed to refresh platform cache", "error", err)
		http.Error(w, "Failed to refresh cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	platforms, _ := h.platformLoader.GetAll()
	count := len(platforms)

	h.logger.Info("Platform cache refreshed via admin API",
		"platform_count", count,
	)

	response := map[string]interface{}{
		"message":        "Platform cache refreshed",
		"platform_count": count,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPlatforms handles GET /api/admin/platforms
func (h *AdminPlatformHandler) ListPlatforms(w http.ResponseWriter, r *http.Request) {
	platforms, err := h.platformLoader.GetAll()
	if err != nil {
		h.logger.Error("Failed to get platforms", "error", err)
		http.Error(w, "Failed to get platforms", http.StatusInternalServerError)
		return
	}

	// Convert to response DTOs
	responses := make([]PlatformResponse, 0, len(platforms))
	for _, p := range platforms {
		updatedAt := time.Time{}
		if p.UpdatedAt != nil {
			updatedAt = *p.UpdatedAt
		}

		responses = append(responses, PlatformResponse{
			ID:          p.ID,
			Name:        p.Name,
			URLPatterns: p.URLPatterns,
			Priority:    p.Priority,
			Enabled:     p.Enabled,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   updatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

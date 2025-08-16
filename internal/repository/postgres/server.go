package postgres

import (
	"context"
	"database/sql"
	"encoding/json" // Added for JSONB unmarshaling
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"time"
)

// ServerRepository implements the domain.ServerRepository interface using PostgreSQL
type ServerRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewServerRepository creates a new PostgreSQL server repository
func NewServerRepository(db *sql.DB, logger *slog.Logger) *ServerRepository {
	return &ServerRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID retrieves a server by its Discord ID
func (r *ServerRepository) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	query := `
		SELECT id, name, configured_channel_id, settings, created_at, updated_at
		FROM servers
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	server := &domain.Server{}
	var configuredChannelID sql.NullString
	var updatedAt sql.NullTime
	var settingsBytes []byte // Use []byte for JSONB column

	err := row.Scan(
		&server.ID,
		&server.Name,
		&configuredChannelID,
		&settingsBytes, // Scan into []byte first
		&server.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Server not found", "server_id", id)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query server",
			"error", err,
			"server_id", id,
		)
		return nil, fmt.Errorf("failed to query server: %w", err)
	}

	// Handle nullable fields
	if configuredChannelID.Valid {
		server.ConfiguredChannelID = &configuredChannelID.String
	}
	if updatedAt.Valid {
		server.UpdatedAt = &updatedAt.Time
	}

	// Convert JSONB bytes to map[string]interface{}
	if len(settingsBytes) > 0 {
		// Import encoding/json at the top of the file
		var settings map[string]interface{}
		if err := json.Unmarshal(settingsBytes, &settings); err != nil {
			r.logger.Warn("Failed to unmarshal server settings",
				"error", err,
				"server_id", id,
				"settings_bytes", string(settingsBytes),
			)
			// Use empty map if unmarshaling fails
			server.Settings = make(map[string]interface{})
		} else {
			server.Settings = settings
		}
	} else {
		server.Settings = make(map[string]interface{})
	}

	r.logger.Debug("Server found", "server_id", id, "name", server.Name)
	return server, nil
}

// Create inserts a new server configuration
func (r *ServerRepository) Create(ctx context.Context, server *domain.Server) error {
	query := `
		INSERT INTO servers (id, name, configured_channel_id, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	// Handle nullable fields
	var configuredChannelID interface{}
	if server.ConfiguredChannelID != nil {
		configuredChannelID = *server.ConfiguredChannelID
	}

	// Default settings if nil
	settings := server.Settings
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Set timestamps
	now := time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = now
	}
	updatedAt := server.CreatedAt
	if server.UpdatedAt != nil {
		updatedAt = *server.UpdatedAt
	}

	_, err := r.db.ExecContext(ctx, query,
		server.ID,
		server.Name,
		configuredChannelID,
		settings,
		server.CreatedAt,
		updatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create server",
			"error", err,
			"server_id", server.ID,
		)
		return fmt.Errorf("failed to create server: %w", err)
	}

	r.logger.Info("Server created successfully",
		"server_id", server.ID,
		"name", server.Name,
	)

	return nil
}

// Update modifies an existing server configuration
func (r *ServerRepository) Update(ctx context.Context, server *domain.Server) error {
	r.logger.Info("Update called (not implemented yet)", "server_id", server.ID)
	// TODO: Implement actual PostgreSQL update
	return nil
}

// Delete removes a server configuration
func (r *ServerRepository) Delete(ctx context.Context, id string) error {
	r.logger.Info("Delete called (not implemented yet)", "server_id", id)
	// TODO: Implement actual PostgreSQL delete
	return nil
}

// List retrieves all configured servers with pagination
func (r *ServerRepository) List(ctx context.Context, offset, limit int) ([]*domain.Server, int, error) {
	r.logger.Info("List called (not implemented yet)", "offset", offset, "limit", limit)
	// TODO: Implement actual PostgreSQL query
	return nil, 0, nil
}

// GetByChannelID finds a server that has the specified channel configured
func (r *ServerRepository) GetByChannelID(ctx context.Context, channelID string) (*domain.Server, error) {
	r.logger.Info("GetByChannelID called (not implemented yet)", "channel_id", channelID)
	// TODO: Implement actual PostgreSQL query
	return nil, sql.ErrNoRows
}

// UpdateSettings updates just the settings field for a server
func (r *ServerRepository) UpdateSettings(ctx context.Context, id string, settings map[string]interface{}) error {
	r.logger.Info("UpdateSettings called (not implemented yet)",
		"server_id", id,
		"settings", settings,
	)
	// TODO: Implement actual PostgreSQL update
	return nil
}

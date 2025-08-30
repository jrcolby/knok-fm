package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// KnokRepository implements the domain.KnokRepository interface using PostgreSQL
type KnokRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewKnokRepository creates a new PostgreSQL knok repository
func NewKnokRepository(db *sql.DB, logger *slog.Logger) *KnokRepository {
	return &KnokRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID retrieves a knok by its UUID
func (r *KnokRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Knok, error) {
	query := `
		SELECT id, server_id, url, platform, title,
		       discord_message_id, discord_channel_id,
		       message_content, metadata, extraction_status, posted_at,
		       created_at, updated_at
		FROM knoks
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	knok := &domain.Knok{}
	var title, messageContent sql.NullString
	// var duration sql.NullInt32
	var updatedAt sql.NullTime
	var metadataBytes []byte // Use []byte for JSONB column

	err := row.Scan(
		&knok.ID,
		&knok.ServerID,
		&knok.URL,
		&knok.Platform,
		&title,
		// &duration,
		// &thumbnailURL,
		&knok.DiscordMessageID,
		&knok.DiscordChannelID,
		&messageContent,
		&metadataBytes, // Scan into []byte first
		&knok.ExtractionStatus,
		&knok.PostedAt,
		&knok.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Knok not found", "knok_id", id)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok",
			"error", err,
			"knok_id", id,
		)
		return nil, fmt.Errorf("failed to query knok: %w", err)
	}

	// Handle nullable fields
	if title.Valid {
		knok.Title = &title.String
	}
	// if duration.Valid {
	// 	durationInt := int(duration.Int32)
	// 	knok.Duration = &durationInt
	// }
	if messageContent.Valid {
		knok.MessageContent = &messageContent.String
	}
	// if thumbnailURL.Valid {
	// 	knok.ThumbnailURL = &thumbnailURL.String
	// }
	if updatedAt.Valid {
		knok.UpdatedAt = &updatedAt.Time
	}

	// Convert JSONB bytes to map[string]interface{}
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			r.logger.Warn("Failed to unmarshal knok metadata",
				"error", err,
				"knok_id", id,
				"metadata_bytes", string(metadataBytes),
			)
			// Use empty map if unmarshaling fails
			knok.Metadata = make(map[string]interface{})
		} else {
			knok.Metadata = metadata
		}
	} else {
		knok.Metadata = make(map[string]interface{})
	}

	r.logger.Debug("Knok found", "knok_id", id, "url", knok.URL)
	return knok, nil
}

// GetByServerID retrieves knoks for a Discord server with pagination
func (r *KnokRepository) GetByServerID(ctx context.Context, serverID string, offset, limit int) ([]*domain.Knok, int, error) {
	r.logger.Info("GetByServerID called (not implemented yet)",
		"server_id", serverID,
		"offset", offset,
		"limit", limit,
	)
	// TODO: Implement actual PostgreSQL query
	return nil, 0, nil
}

// GetByDiscordMessage retrieves a knok by Discord message ID
func (r *KnokRepository) GetByDiscordMessage(ctx context.Context, messageID string) (*domain.Knok, error) {
	query := `
		SELECT id, server_id, url, platform, title,
		       discord_message_id, discord_channel_id,
		       message_content, metadata, extraction_status, posted_at,
		       created_at, updated_at
		FROM knoks
		WHERE discord_message_id = $1`

	row := r.db.QueryRowContext(ctx, query, messageID)

	knok := &domain.Knok{}
	var title, messageContent sql.NullString
	var updatedAt sql.NullTime
	var metadataBytes []byte // Use []byte for JSONB column

	err := row.Scan(
		&knok.ID,
		&knok.ServerID,
		&knok.URL,
		&knok.Platform,
		&title,
		&knok.DiscordMessageID,
		&knok.DiscordChannelID,
		&messageContent,
		&metadataBytes, // Scan into []byte first
		&knok.ExtractionStatus,
		&knok.PostedAt,
		&knok.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Knok not found by message ID", "message_id", messageID)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok by message ID",
			"error", err,
			"message_id", messageID,
		)
		return nil, fmt.Errorf("failed to query knok by message ID: %w", err)
	}

	// Handle nullable fields
	if title.Valid {
		knok.Title = &title.String
	}
	// if duration.Valid {
	// 	durationInt := int(duration.Int32)
	// 	knok.Duration = &durationInt
	// }
	if messageContent.Valid {
		knok.MessageContent = &messageContent.String
	}
	// if thumbnailURL.Valid {
	// 	knok.ThumbnailURL = &thumbnailURL.String
	// }
	if updatedAt.Valid {
		knok.UpdatedAt = &updatedAt.Time
	}

	// Convert JSONB bytes to map[string]interface{}
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			r.logger.Warn("Failed to unmarshal knok metadata",
				"error", err,
				"message_id", messageID,
				"metadata_bytes", string(metadataBytes),
			)
			// Use empty map if unmarshaling fails
			knok.Metadata = make(map[string]interface{})
		} else {
			knok.Metadata = metadata
		}
	} else {
		knok.Metadata = make(map[string]interface{})
	}

	r.logger.Debug("Knok found by message ID", "message_id", messageID, "knok_id", knok.ID)
	return knok, nil
}

// Search performs full-text search on knoks within a server
func (r *KnokRepository) Search(ctx context.Context, serverID, query string, limit int) ([]*domain.Knok, error) {
	r.logger.Info("Search called (not implemented yet)",
		"server_id", serverID,
		"query", query,
		"limit", limit,
	)
	// TODO: Implement actual PostgreSQL query
	return nil, nil
}

// Create inserts a new knok
func (r *KnokRepository) Create(ctx context.Context, knok *domain.Knok) error {
	query := `
		INSERT INTO knoks (
			id, server_id, url, platform, title,
			discord_message_id, discord_channel_id,
			message_content, metadata, extraction_status, posted_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	// Handle nullable fields
	var title, messageContent interface{}

	if knok.Title != nil {
		title = *knok.Title
	}
	// if knok.Duration != nil {
	// 	duration = *knok.Duration
	// }
	if knok.MessageContent != nil {
		messageContent = *knok.MessageContent
	}
	// if knok.ThumbnailURL != nil {
	// 	thumbnailURL = *knok.ThumbnailURL
	// }

	// Convert metadata to JSON
	metadata := knok.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Convert metadata map to JSON for JSONB column
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		r.logger.Error("Failed to marshal knok metadata",
			"error", err,
			"knok_id", knok.ID,
			"metadata", metadata,
		)
		return fmt.Errorf("failed to marshal knok metadata: %w", err)
	}

	// Set updated_at to same as created_at for new records
	updatedAt := knok.CreatedAt
	if knok.UpdatedAt != nil {
		updatedAt = *knok.UpdatedAt
	}

	_, err = r.db.ExecContext(ctx, query,
		knok.ID,
		knok.ServerID,
		knok.URL,
		knok.Platform,
		title,
		// duration,
		// thumbnailURL,
		knok.DiscordMessageID,
		knok.DiscordChannelID,
		messageContent,
		metadataJSON,
		knok.ExtractionStatus,
		knok.PostedAt,
		knok.CreatedAt,
		updatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create knok",
			"error", err,
			"knok_id", knok.ID,
			"url", knok.URL,
		)
		return fmt.Errorf("failed to create knok: %w", err)
	}

	r.logger.Info("Knok created successfully",
		"knok_id", knok.ID,
		"url", knok.URL,
		"platform", knok.Platform,
		"server_id", knok.ServerID,
	)

	return nil
}

// Update modifies an existing knok
func (r *KnokRepository) Update(ctx context.Context, knok *domain.Knok) error {
	query := `
		UPDATE knoks SET
			server_id = $2,
			url = $3,
			platform = $4,
			title = $5,
			discord_message_id = $6,
			discord_channel_id = $7,
			message_content = $8,
			metadata = $9,
			extraction_status = $10,
			posted_at = $11,
			updated_at = $12
		WHERE id = $1`

	// Handle nullable fields
	var title, messageContent interface{}

	if knok.Title != nil {
		title = *knok.Title
	}
	// if knok.Duration != nil {
	// 	duration = *knok.Duration
	// }
	if knok.MessageContent != nil {
		messageContent = *knok.MessageContent
	}
	// if knok.ThumbnailURL != nil {
	// 	thumbnailURL = *knok.ThumbnailURL
	// }

	// Convert metadata to JSON
	metadata := knok.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Convert metadata map to JSON for JSONB column
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		r.logger.Error("Failed to marshal knok metadata",
			"error", err,
			"knok_id", knok.ID,
			"metadata", metadata,
		)
		return fmt.Errorf("failed to marshal knok metadata: %w", err)
	}

	// Set updated_at to current time
	now := time.Now()
	knok.UpdatedAt = &now

	_, err = r.db.ExecContext(ctx, query,
		knok.ID,
		knok.ServerID,
		knok.URL,
		knok.Platform,
		title,
		// duration,
		// thumbnailURL,
		knok.DiscordMessageID,
		knok.DiscordChannelID,
		messageContent,
		metadataJSON,
		knok.ExtractionStatus,
		knok.PostedAt,
		knok.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update knok",
			"error", err,
			"knok_id", knok.ID,
		)
		return fmt.Errorf("failed to update knok: %w", err)
	}

	r.logger.Info("Knok updated successfully",
		"knok_id", knok.ID,
		"url", knok.URL,
		"status", knok.ExtractionStatus,
	)

	return nil
}

// Delete removes a knok by ID
func (r *KnokRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.logger.Info("Delete called (not implemented yet)", "knok_id", id)
	// TODO: Implement actual PostgreSQL delete
	return nil
}

// GetByURL finds knoks by URL within a server (for duplicate detection)
func (r *KnokRepository) GetByURL(ctx context.Context, serverID, url string) (*domain.Knok, error) {
	query := `
		SELECT id, server_id, url, platform, title,
		       discord_message_id, discord_channel_id,
		       message_content, metadata, extraction_status, posted_at,
		       created_at, updated_at
		FROM knoks
		WHERE server_id = $1 AND url = $2
		ORDER BY created_at DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, serverID, url)

	knok := &domain.Knok{}
	var title, messageContent sql.NullString
	var updatedAt sql.NullTime
	var metadataBytes []byte // Use []byte for JSONB column

	err := row.Scan(
		&knok.ID,
		&knok.ServerID,
		&knok.URL,
		&knok.Platform,
		&title,
		// &duration,
		// &thumbnailURL,
		&knok.DiscordMessageID,
		&knok.DiscordChannelID,
		&messageContent,
		&metadataBytes, // Scan into []byte first
		&knok.ExtractionStatus,
		&knok.PostedAt,
		&knok.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("No duplicate knok found",
				"server_id", serverID,
				"url", url,
			)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok by URL",
			"error", err,
			"server_id", serverID,
			"url", url,
		)
		return nil, fmt.Errorf("failed to query knok by URL: %w", err)
	}

	// Handle nullable fields
	if title.Valid {
		knok.Title = &title.String
	}
	// if duration.Valid {
	// 	dur := int(duration.Int32)
	// 	knok.Duration = &dur
	// }
	if messageContent.Valid {
		knok.MessageContent = &messageContent.String
	}
	// if thumbnailURL.Valid {
	// 	knok.ThumbnailURL = &thumbnailURL.String
	// }
	if updatedAt.Valid {
		knok.UpdatedAt = &updatedAt.Time
	}

	// Convert JSONB bytes to map[string]interface{}
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			r.logger.Warn("Failed to unmarshal knok metadata",
				"error", err,
				"server_id", serverID,
				"url", url,
				"metadata_bytes", string(metadataBytes),
			)
			// Use empty map if unmarshaling fails
			knok.Metadata = make(map[string]interface{})
		} else {
			knok.Metadata = metadata
		}
	} else {
		knok.Metadata = make(map[string]interface{})
	}

	r.logger.Debug("Found existing knok",
		"knok_id", knok.ID,
		"server_id", serverID,
		"url", url,
	)

	return knok, nil
}

// GetRecentByServer gets the most recent knoks for a server
func (r *KnokRepository) GetRecentByServer(ctx context.Context, serverID string, limit int) ([]*domain.Knok, error) {
	r.logger.Info("GetRecentByServer called (not implemented yet)",
		"server_id", serverID,
		"limit", limit,
	)
	// TODO: Implement actual PostgreSQL query
	return nil, nil
}

// GetByPlatform gets knoks filtered by platform within a server
func (r *KnokRepository) GetByPlatform(ctx context.Context, serverID, platform string, offset, limit int) ([]*domain.Knok, int, error) {
	r.logger.Info("GetByPlatform called (not implemented yet)",
		"server_id", serverID,
		"platform", platform,
		"offset", offset,
		"limit", limit,
	)
	// TODO: Implement actual PostgreSQL query
	return nil, 0, nil
}

// UpdateExtractionStatus updates the metadata extraction status
func (r *KnokRepository) UpdateExtractionStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE knoks 
		SET extraction_status = $1, updated_at = NOW()
		WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		r.logger.Error("Failed to update extraction status",
			"error", err,
			"knok_id", id,
			"status", status,
		)
		return fmt.Errorf("failed to update extraction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("No knok found for status update", "knok_id", id)
		return fmt.Errorf("knok not found: %s", id)
	}

	r.logger.Info("Extraction status updated successfully",
		"knok_id", id,
		"status", status,
		"rows_affected", rowsAffected,
	)

	return nil
}

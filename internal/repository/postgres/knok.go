package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const knokSelectFields = `
	SELECT id, server_id, url, platform, title,
		   discord_message_id, discord_channel_id,
		   message_content, metadata, extraction_status, posted_at,
		   created_at, updated_at
	FROM knoks`

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

// sanitizeSearchQuery escapes special tsquery characters and prepares for prefix search
func (r *KnokRepository) sanitizeSearchQuery(query string) string {
	// Remove leading/trailing whitespace
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Escape special tsquery characters: & | ! ( ) : *
	// Replace with spaces to avoid breaking the query
	specialChars := regexp.MustCompile(`[&|!():*]`)
	query = specialChars.ReplaceAllString(query, " ")

	// Replace multiple spaces with single space
	multiSpace := regexp.MustCompile(`\s+`)
	query = multiSpace.ReplaceAllString(query, " ")

	// Split into words and join with :* for prefix matching
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	// Add :* suffix to each word for prefix matching, then join with & (AND)
	var prefixWords []string
	for _, word := range words {
		if word != "" {
			prefixWords = append(prefixWords, word+":*")
		}
	}

	return strings.Join(prefixWords, " & ")
}

// scanKnokRow scans a database row into a Knok struct and handles nullable fields
func (r *KnokRepository) scanKnokRow(scanner interface{ Scan(...interface{}) error }) (*domain.Knok, error) {
	knok := &domain.Knok{}
	var title, messageContent sql.NullString
	var updatedAt sql.NullTime
	var metadataBytes []byte

	err := scanner.Scan(
		&knok.ID,
		&knok.ServerID,
		&knok.URL,
		&knok.Platform,
		&title,
		&knok.DiscordMessageID,
		&knok.DiscordChannelID,
		&messageContent,
		&metadataBytes,
		&knok.ExtractionStatus,
		&knok.PostedAt,
		&knok.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	r.processNullableFields(knok, title, messageContent, updatedAt)
	if err := r.processMetadata(knok, metadataBytes); err != nil {
		return nil, err
	}

	return knok, nil
}

// processNullableFields handles nullable string and time fields
func (r *KnokRepository) processNullableFields(knok *domain.Knok, title, messageContent sql.NullString, updatedAt sql.NullTime) {
	if title.Valid {
		knok.Title = &title.String
	}
	if messageContent.Valid {
		knok.MessageContent = &messageContent.String
	}
	if updatedAt.Valid {
		knok.UpdatedAt = &updatedAt.Time
	}
}

// processMetadata unmarshals JSONB metadata into the knok struct
func (r *KnokRepository) processMetadata(knok *domain.Knok, metadataBytes []byte) error {
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			r.logger.Warn("Failed to unmarshal knok metadata", "error", err, "knok_id", knok.ID)
			knok.Metadata = make(map[string]interface{})
		} else {
			knok.Metadata = metadata
		}
	} else {
		knok.Metadata = make(map[string]interface{})
	}
	return nil
}

// Uses Postgres random query ability to get a random knok row
func (r *KnokRepository) GetRandom(ctx context.Context) (*domain.Knok, error) {
	// First get total completed knoks count
	var count int
	countQuery := "SELECT COUNT(*) FROM knoks WHERE extraction_status = 'complete'"
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count knoks", "error", err)
		return nil, fmt.Errorf("failed to count knoks: %w", err)
	}

	if count == 0 {
		return nil, sql.ErrNoRows
	}

	// Generate random offset
	offset := rand.Intn(count)

	query := knokSelectFields + `
		WHERE extraction_status = 'complete'
		OFFSET $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, offset)

	knok, err := r.scanKnokRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("No knoks to randomly choose")
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query random knok", "error", err)
		return nil, fmt.Errorf("failed to query knok: %w", err)
	}

	r.logger.Debug("Knok found", "knok_id", knok.ID, "url", knok.URL)
	return knok, nil
}

// GetByID retrieves a knok by its UUID
func (r *KnokRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Knok, error) {
	query := knokSelectFields + `
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	knok, err := r.scanKnokRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Knok not found", "knok_id", id)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok", "error", err, "knok_id", id)
		return nil, fmt.Errorf("failed to query knok: %w", err)
	}

	r.logger.Debug("Knok found", "knok_id", id, "url", knok.URL)
	return knok, nil
}

// GetByDiscordMessage retrieves a knok by Discord message ID
func (r *KnokRepository) GetByDiscordMessage(ctx context.Context, messageID string) (*domain.Knok, error) {
	query := knokSelectFields + `
		WHERE discord_message_id = $1`

	row := r.db.QueryRowContext(ctx, query, messageID)

	knok, err := r.scanKnokRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Knok not found by message ID", "message_id", messageID)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok by message ID", "error", err, "message_id", messageID)
		return nil, fmt.Errorf("failed to query knok by message ID: %w", err)
	}

	r.logger.Debug("Knok found by message ID", "message_id", messageID, "knok_id", knok.ID)
	return knok, nil
}

// Search performs full-text search on knoks within a server with cursor pagination
func (r *KnokRepository) Search(ctx context.Context, searchQuery string, cursor *time.Time, limit int) ([]*domain.Knok, error) {
	r.logger.Info("Search called", "query", searchQuery, "cursor", cursor, "limit", limit)

	// Sanitize and prepare the search query for prefix matching
	sanitizedQuery := r.sanitizeSearchQuery(searchQuery)
	if sanitizedQuery == "" {
		r.logger.Debug("Empty search query after sanitization", "original", searchQuery)
		return []*domain.Knok{}, nil
	}

	r.logger.Debug("Search query sanitized", "original", searchQuery, "sanitized", sanitizedQuery)

	var query string
	var args []interface{}

	if cursor == nil {
		query = knokSelectFields + `
			WHERE search_vector @@ to_tsquery('english', $1)
			ORDER BY posted_at DESC
			LIMIT $2`
		args = []interface{}{sanitizedQuery, limit}
	} else {
		query = knokSelectFields + `
			WHERE search_vector @@ to_tsquery('english', $1) AND posted_at < $2
			ORDER BY posted_at DESC
			LIMIT $3`
		args = []interface{}{sanitizedQuery, *cursor, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to search knoks", "error", err, "query", searchQuery)
		return nil, fmt.Errorf("failed to search knoks: %w", err)
	}
	defer rows.Close()

	var knoks []*domain.Knok
	for rows.Next() {
		knok, err := r.scanKnokRow(rows)
		if err != nil {
			r.logger.Error("Failed to scan search result", "error", err)
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		knoks = append(knoks, knok)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error occurred during search iteration", "error", err)
		return nil, fmt.Errorf("error occurred during search iteration: %w", err)
	}

	r.logger.Debug("Search completed successfully", "query", searchQuery, "results_count", len(knoks))
	return knoks, nil
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

	if knok.MessageContent != nil {
		messageContent = *knok.MessageContent
	}

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

	if knok.MessageContent != nil {
		messageContent = *knok.MessageContent
	}

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
	query := `DELETE FROM knoks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete knok from database", "error", err, "knok_id", id)
		return fmt.Errorf("failed to delete knok: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", "error", err, "knok_id", id)
		return fmt.Errorf("failed to verify deletion: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("No knok found to delete", "knok_id", id)
		return fmt.Errorf("knok not found")
	}

	r.logger.Info("Knok deleted from database", "knok_id", id, "rows_affected", rowsAffected)
	return nil
}

// GetByURL finds knoks by URL within a server (for duplicate detection)
func (r *KnokRepository) GetByURL(ctx context.Context, serverID, url string) (*domain.Knok, error) {
	query := knokSelectFields + `
		WHERE server_id = $1 AND url = $2
		ORDER BY created_at DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, serverID, url)

	knok, err := r.scanKnokRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("No duplicate knok found", "server_id", serverID, "url", url)
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to query knok by URL", "error", err, "server_id", serverID, "url", url)
		return nil, fmt.Errorf("failed to query knok by URL: %w", err)
	}

	r.logger.Debug("Found existing knok", "knok_id", knok.ID, "server_id", serverID, "url", url)
	return knok, nil
}

// GetRecentByServer gets the most recent knoks for a server with cursor pagination
func (r *KnokRepository) GetRecentByServer(ctx context.Context, serverID string, cursor *time.Time, limit int) ([]*domain.Knok, error) {
	r.logger.Info("GetRecentByServer called", "server_id", serverID, "cursor", cursor, "limit", limit)

	var query string
	var args []interface{}

	if cursor == nil {
		query = knokSelectFields + `
			WHERE server_id = $1 AND extraction_status = 'complete'
			ORDER BY posted_at DESC
			LIMIT $2`
		args = []interface{}{serverID, limit}
	} else {
		query = knokSelectFields + `
			WHERE server_id = $1 AND posted_at < $2 AND extraction_status = 'complete'
			ORDER BY posted_at DESC
			LIMIT $3`
		args = []interface{}{serverID, *cursor, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to query recent knoks", "error", err, "server_id", serverID, "limit", limit)
		return nil, fmt.Errorf("failed to query recent knoks: %w", err)
	}
	defer rows.Close()

	var knoks []*domain.Knok
	for rows.Next() {
		knok, err := r.scanKnokRow(rows)
		if err != nil {
			r.logger.Error("Failed to scan knok", "error", err)
			return nil, fmt.Errorf("failed to scan knok: %w", err)
		}
		knoks = append(knoks, knok)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error occurred during rows iteration", "error", err)
		return nil, fmt.Errorf("error occurred during rows iteration: %w", err)
	}

	r.logger.Debug("Recent knoks retrieved successfully", "server_id", serverID, "limit", limit, "knoks_count", len(knoks))
	return knoks, nil
}

// GetRecent gets recent knoks across all servers (global timeline)
func (r *KnokRepository) GetRecent(ctx context.Context, cursor *time.Time, limit int) ([]*domain.Knok, error) {
	r.logger.Info("GetRecent called (global)", "cursor", cursor, "limit", limit)

	var query string
	var args []interface{}

	if cursor == nil {
		query = knokSelectFields + `
			WHERE extraction_status = 'complete'
			ORDER BY posted_at DESC
			LIMIT $1`
		args = []interface{}{limit}
	} else {
		query = knokSelectFields + `
			WHERE posted_at < $1 AND extraction_status = 'complete'
			ORDER BY posted_at DESC
			LIMIT $2`
		args = []interface{}{*cursor, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to query recent knoks (global)", "error", err, "limit", limit)
		return nil, fmt.Errorf("failed to query recent knoks: %w", err)
	}
	defer rows.Close()

	var knoks []*domain.Knok
	for rows.Next() {
		knok, err := r.scanKnokRow(rows)
		if err != nil {
			r.logger.Error("Failed to scan knok", "error", err)
			return nil, fmt.Errorf("failed to scan knok: %w", err)
		}
		knoks = append(knoks, knok)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error occurred during rows iteration", "error", err)
		return nil, fmt.Errorf("error occurred during rows iteration: %w", err)
	}

	r.logger.Debug("Recent knoks retrieved successfully (global)", "limit", limit, "knoks_count", len(knoks))
	return knoks, nil
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

package postgres

import (
	"context"
	"database/sql"
	"encoding/json" // Added for JSONB unmarshaling
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"time"

	"github.com/lib/pq"
)

type PlatformRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPlatformRepository creates a new PostgreSQL platform repository
func NewPlatformRepository(db *sql.DB, logger *slog.Logger) *PlatformRepository {
	return &PlatformRepository{
		db:     db,
		logger: logger,
	}
}

const platformSelectFields = `
	SELECT id, name, url_patterns, priority, enabled, extraction_patterns, created_at, updated_at
	FROM platforms
`

// GetAllPlatforms fetches all platform configurations from the database.
// This function assumes the number of platforms will not be excessively large (e.g., < 100).
func (r *PlatformRepository) GetAllPlatforms(ctx context.Context) ([]*domain.Platform, error) {
	r.logger.Info("GetAllPlatforms called")

	// No cursor or limit needed for getting all platforms
	rows, err := r.db.QueryContext(ctx, platformSelectFields)
	if err != nil {
		r.logger.Error("Failed to query all platforms", "error", err)
		return nil, fmt.Errorf("failed to query all platforms: %w", err)
	}
	defer rows.Close()

	var platforms []*domain.Platform
	for rows.Next() {
		platform, err := r.scanPlatformRow(rows)
		if err != nil {
			r.logger.Error("Failed to scan platform row", "error", err)
			return nil, fmt.Errorf("failed to scan platform: %w", err)
		}
		platforms = append(platforms, platform)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error occurred during platform iteration", "error", err)
		return nil, fmt.Errorf("error occurred during platform iteration: %w", err)
	}

	r.logger.Info("Successfully fetched all platforms", "count", len(platforms))
	return platforms, nil
}

func (r *PlatformRepository) scanPlatformRow(scanner interface{ Scan(...interface{}) error }) (*domain.Platform, error) {
	platform := &domain.Platform{}
	var updatedAt sql.NullTime
	var extractionPatternsJSON sql.NullString
	err := scanner.Scan(
		&platform.ID,
		&platform.Name,
		pq.Array(&platform.URLPatterns), // Use pq.Array for PostgreSQL TEXT[] type
		&platform.Priority,
		&platform.Enabled,
		&extractionPatternsJSON,
		&platform.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan platform row: %w", err)
	}

	if updatedAt.Valid {
		platform.UpdatedAt = &updatedAt.Time
	} else {
		platform.UpdatedAt = nil
	}

	if extractionPatternsJSON.Valid {
		// If the JSONB column was NOT NULL, unmarshal its string content into []string
		var patterns []string
		if err := json.Unmarshal([]byte(extractionPatternsJSON.String), &patterns); err != nil {
			// This would indicate malformed JSON in the database, which is an error
			return nil, fmt.Errorf("failed to unmarshal extraction patterns from JSONB: %w", err)
		}
		platform.ExtractionPatterns = patterns
	} else {
		// If the JSONB column was NULL, set the slice to nil
		platform.ExtractionPatterns = nil
	}

	return platform, nil

}

// CreatePlatform inserts a new platform into the database
func (r *PlatformRepository) CreatePlatform(ctx context.Context, platform *domain.Platform) error {
	query := `
        INSERT INTO platforms (id, name, url_patterns, priority, enabled, extraction_patterns, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	// Handle ExtractionPatterns ([]string -> JSONB)
	var extractionPatternsJSON []byte
	var err error

	if platform.ExtractionPatterns != nil {
		extractionPatternsJSON, err = json.Marshal(platform.ExtractionPatterns)
		if err != nil {
			r.logger.Error("Failed to marshal platform extraction patterns",
				"error", err,
				"platform_id", platform.ID,
			)
			return fmt.Errorf("failed to marshal platform extraction patterns: %w", err)
		}
	} else {
		// If the slice is nil, we want to store NULL in the JSONB column
		extractionPatternsJSON = nil
	}

	// Set created_at and updated_at to current time if not already set
	now := time.Now()
	if platform.CreatedAt.IsZero() {
		platform.CreatedAt = now
	}
	platform.UpdatedAt = &now // Always set UpdatedAt on creation as well

	_, err = r.db.ExecContext(ctx, query,
		platform.ID,
		platform.Name,
		pq.Array(platform.URLPatterns), // Use pq.Array for PostgreSQL TEXT[] type
		platform.Priority,
		platform.Enabled,
		extractionPatternsJSON,
		platform.CreatedAt,
		platform.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create platform",
			"error", err,
			"platform_id", platform.ID,
			"platform_name", platform.Name,
		)
		return fmt.Errorf("failed to create platform: %w", err)
	}

	r.logger.Info("Platform created successfully",
		"platform_id", platform.ID,
		"platform_name", platform.Name,
	)

	return nil
}

// UpdatePlatform modifies an existing platform
func (r *PlatformRepository) UpdatePlatform(ctx context.Context, platform *domain.Platform) error {
	query := `
		UPDATE platforms SET
			name = $2,
			url_patterns = $3,
			priority = $4,
			enabled = $5,
			extraction_patterns = $6,
			updated_at = $7
		WHERE id = $1`

	// Handle ExtractionPatterns ([]string -> JSONB)
	var extractionPatternsJSON []byte
	var err error

	if platform.ExtractionPatterns != nil {
		extractionPatternsJSON, err = json.Marshal(platform.ExtractionPatterns)
		if err != nil {
			r.logger.Error("Failed to marshal platform extraction patterns",
				"error", err,
				"platform_id", platform.ID,
			)
			return fmt.Errorf("failed to marshal platform extraction patterns: %w", err)
		}
	} else {
		// If the slice is nil, we want to store NULL in the JSONB column
		extractionPatternsJSON = nil
	}

	// Set updated_at to current time
	now := time.Now()
	platform.UpdatedAt = &now // Ensure updated_at is set for the database

	_, err = r.db.ExecContext(ctx, query,
		platform.ID,
		platform.Name,
		pq.Array(platform.URLPatterns), // Use pq.Array for PostgreSQL TEXT[] type
		platform.Priority,
		platform.Enabled,
		extractionPatternsJSON,
		platform.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update platform",
			"error", err,
			"platform_id", platform.ID,
			"platform_name", platform.Name,
		)
		return fmt.Errorf("failed to update platform: %w", err)
	}

	r.logger.Info("Platform updated successfully",
		"platform_id", platform.ID,
		"platform_name", platform.Name,
	)

	return nil
}

// DeletePlatform removes a platform by ID
func (r *PlatformRepository) DeletePlatform(ctx context.Context, id string) error {
	query := `DELETE FROM platforms WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete platform",
			"error", err,
			"platform_id", id,
		)
		return fmt.Errorf("failed to delete platform: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected after deleting platform",
			"error", err,
			"platform_id", id,
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Debug("No platform found to delete", "platform_id", id)
		return sql.ErrNoRows // Or a custom "not found" error
	}

	r.logger.Info("Platform deleted successfully", "platform_id", id)
	return nil
}

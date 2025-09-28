package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "initial_schema",
		SQL: `
			-- Create servers table
			CREATE TABLE IF NOT EXISTS servers (
				id VARCHAR(20) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				configured_channel_id VARCHAR(20),
				settings JSONB DEFAULT '{}',
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_servers_channel
			ON servers(configured_channel_id);

			-- Create knoks table (slimmed down, no duration/thumbnail_url columns)
			CREATE TABLE IF NOT EXISTS knoks (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				server_id VARCHAR(20) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
				url TEXT NOT NULL,
				platform VARCHAR(50) NOT NULL,
				title VARCHAR(500),
				
				-- Discord-specific fields
				discord_message_id VARCHAR(20) NOT NULL,
				discord_channel_id VARCHAR(20) NOT NULL,
				message_content TEXT,
				
				-- Metadata and processing
				metadata JSONB DEFAULT '{}',
				extraction_status VARCHAR(20) DEFAULT 'pending',
				
				-- Timestamps
				posted_at TIMESTAMP WITH TIME ZONE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				
				-- Search vector for full-text search
				search_vector tsvector,

				-- Constraints
				UNIQUE(discord_message_id, url),
				CHECK (platform IN ('youtube', 'soundcloud', 'mixcloud', 'bandcamp', 'spotify', 'apple_music', 'nts', 'dublab', 'noods')),
				CHECK (extraction_status IN ('pending', 'processing', 'complete', 'failed'))
			);

			-- Create indexes
			CREATE INDEX IF NOT EXISTS idx_knoks_server_posted
			ON knoks(server_id, posted_at DESC);

			CREATE INDEX IF NOT EXISTS idx_knoks_platform
			ON knoks(platform);

			CREATE INDEX IF NOT EXISTS idx_knoks_status
			ON knoks(extraction_status);

			CREATE INDEX IF NOT EXISTS idx_knoks_message
			ON knoks(discord_message_id);
			
			CREATE INDEX IF NOT EXISTS idx_knoks_search
			ON knoks USING GIN(search_vector);

			-- Create search vector update function
			CREATE OR REPLACE FUNCTION update_knoks_search_vector()
			RETURNS trigger AS $$
			BEGIN
				NEW.search_vector := to_tsvector('english',
					coalesce(NEW.title,''));
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;

			-- Create trigger for automatic search vector updates
			CREATE TRIGGER knoks_search_vector_update
				BEFORE INSERT OR UPDATE ON knoks
				FOR EACH ROW EXECUTE FUNCTION update_knoks_search_vector();
		`,
	},
	{
		Version: 2,
		Name:    "update_platform_constraint",
		SQL: `
			-- Drop the old platform constraint
			ALTER TABLE knoks DROP CONSTRAINT IF EXISTS knoks_platform_check;
			
			-- Add new platform constraint with all supported platforms
			ALTER TABLE knoks ADD CONSTRAINT knoks_platform_check ` + domain.GetPlatformConstraintSQL() + `;
		`,
	},
}

// RunMigrations executes all pending database migrations
func RunMigrations(db *sql.DB, logger *slog.Logger) error {
	logger.Info("Running database migrations...")

	// Create migrations table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	logger.Info("Current migration version", "version", currentVersion)

	// Apply pending migrations
	applied := 0
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		logger.Info("Applying migration",
			"version", migration.Version,
			"name", migration.Name,
		)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		if _, err := tx.Exec("INSERT INTO migrations (version, name) VALUES ($1, $2)",
			migration.Version, migration.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		applied++
		logger.Info("Migration applied successfully", "version", migration.Version)
	}

	if applied == 0 {
		logger.Info("No migrations to apply - database is up to date")
	} else {
		logger.Info("Database migrations completed", "applied", applied)
	}

	return nil
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration status: %w", err)
	}
	return version, nil
}

// ResetDatabase drops all tables (for testing)
func ResetDatabase(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	logger.Warn("Resetting database - all data will be lost")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Drop tables in reverse dependency order
	dropSQL := []string{
		"DROP TABLE IF EXISTS knoks CASCADE",
		"DROP TABLE IF EXISTS servers CASCADE",
		"DROP TABLE IF EXISTS migrations CASCADE",
		"DROP FUNCTION IF EXISTS update_knoks_search_vector() CASCADE",
	}

	for _, sql := range dropSQL {
		if _, err := tx.ExecContext(ctx, sql); err != nil {
			return fmt.Errorf("failed to execute drop statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reset transaction: %w", err)
	}

	logger.Info("Database reset completed")
	return nil
}

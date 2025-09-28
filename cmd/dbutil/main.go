package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"knock-fm/internal/pkg/logger"
	"knock-fm/internal/repository/postgres"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	var (
		reset      = flag.Bool("reset", false, "Reset database (WARNING: destroys all data)")
		clearKnoks = flag.Bool("clear-knoks", false, "Clear only knoks table (keeps servers)")
		migrate    = flag.Bool("migrate", false, "Run database migrations")
		status     = flag.Bool("status", false, "Show migration status")
		dbURL      = flag.String("db", "", "Database URL (defaults to DATABASE_URL env var)")
	)
	flag.Parse()

	// Get database URL
	databaseURL := *dbURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			databaseURL = "postgres://dev:devpass@localhost:5432/knockfm?sslmode=disable"
		}
	}

	// Setup logger
	log := logger.New("dbutil")

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Execute commands
	switch {
	case *clearKnoks:
		if err := confirmClearKnoks(); err != nil {
			log.Error("Clear knoks cancelled", "error", err)
			os.Exit(1)
		}
		
		log.Warn("Clearing knoks table...")
		if _, err := db.ExecContext(ctx, "DELETE FROM knoks"); err != nil {
			log.Error("Failed to clear knoks table", "error", err)
			os.Exit(1)
		}
		
		log.Info("Knoks table cleared successfully (servers preserved)")

	case *reset:
		if err := confirmReset(); err != nil {
			log.Error("Reset cancelled", "error", err)
			os.Exit(1)
		}
		
		log.Warn("Resetting database...")
		if err := postgres.ResetDatabase(ctx, db, log); err != nil {
			log.Error("Failed to reset database", "error", err)
			os.Exit(1)
		}
		
		log.Info("Database reset completed successfully")
		log.Info("Run with -migrate to recreate tables")

	case *migrate:
		if err := postgres.RunMigrations(db, log); err != nil {
			log.Error("Failed to run migrations", "error", err)
			os.Exit(1)
		}
		log.Info("Migrations completed successfully")

	case *status:
		version, err := postgres.GetMigrationStatus(db)
		if err != nil {
			log.Error("Failed to get migration status", "error", err)
			os.Exit(1)
		}
		log.Info("Migration status", "current_version", version)

	default:
		fmt.Println("Database utility for Knok FM")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  -clear-knoks Clear only knoks table (keeps servers)")
		fmt.Println("  -reset       Reset database (WARNING: destroys all data)")
		fmt.Println("  -migrate     Run database migrations")
		fmt.Println("  -status      Show migration status")
		fmt.Println("  -db       Database URL (optional)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run cmd/dbutil/main.go -status")
		fmt.Println("  go run cmd/dbutil/main.go -clear-knoks")
		fmt.Println("  go run cmd/dbutil/main.go -reset")
		fmt.Println("  go run cmd/dbutil/main.go -migrate")
		os.Exit(0)
	}
}

func confirmClearKnoks() error {
	fmt.Print("This will delete all knoks but keep servers. Type 'yes' to confirm: ")
	var response string
	fmt.Scanln(&response)
	
	if response != "yes" {
		return fmt.Errorf("clear knoks not confirmed")
	}
	
	return nil
}

func confirmReset() error {
	fmt.Print("WARNING: This will delete ALL data in the database. Type 'yes' to confirm: ")
	var response string
	fmt.Scanln(&response)
	
	if response != "yes" {
		return fmt.Errorf("reset not confirmed")
	}
	
	return nil
}
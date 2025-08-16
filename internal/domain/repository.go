package domain

import (
	"context"

	"github.com/google/uuid"
)

// KnokRepository defines the interface for knok data operations
type KnokRepository interface {
	// GetByID retrieves a knok by its UUID
	GetByID(ctx context.Context, id uuid.UUID) (*Knok, error)

	// GetByServerID retrieves knoks for a Discord server with pagination
	GetByServerID(ctx context.Context, serverID string, offset, limit int) ([]*Knok, int, error)

	// GetByDiscordMessage retrieves a knok by Discord message ID
	GetByDiscordMessage(ctx context.Context, messageID string) (*Knok, error)

	// Search performs full-text search on knoks within a server
	Search(ctx context.Context, serverID, query string, limit int) ([]*Knok, error)

	// Create inserts a new knok
	Create(ctx context.Context, knok *Knok) error

	// Update modifies an existing knok
	Update(ctx context.Context, knok *Knok) error

	// Delete removes a knok by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByURL finds knoks by URL within a server (for duplicate detection)
	GetByURL(ctx context.Context, serverID, url string) (*Knok, error)

	// GetRecentByServer gets the most recent knoks for a server
	GetRecentByServer(ctx context.Context, serverID string, limit int) ([]*Knok, error)

	// GetByPlatform gets knoks filtered by platform within a server
	GetByPlatform(ctx context.Context, serverID, platform string, offset, limit int) ([]*Knok, int, error)

	// UpdateExtractionStatus updates the metadata extraction status
	UpdateExtractionStatus(ctx context.Context, id uuid.UUID, status string) error
}

// ServerRepository defines the interface for Discord server data operations
type ServerRepository interface {
	// GetByID retrieves a server by its Discord ID
	GetByID(ctx context.Context, id string) (*Server, error)

	// Create inserts a new server configuration
	Create(ctx context.Context, server *Server) error

	// Update modifies an existing server configuration
	Update(ctx context.Context, server *Server) error

	// Delete removes a server configuration
	Delete(ctx context.Context, id string) error

	// List retrieves all configured servers with pagination
	List(ctx context.Context, offset, limit int) ([]*Server, int, error)

	// GetByChannelID finds a server that has the specified channel configured
	GetByChannelID(ctx context.Context, channelID string) (*Server, error)

	// UpdateSettings updates just the settings field for a server
	UpdateSettings(ctx context.Context, id string, settings map[string]interface{}) error
}

// QueueRepository defines the interface for job queue operations
type QueueRepository interface {
	// Enqueue adds a new job to the queue
	Enqueue(ctx context.Context, jobType string, payload interface{}) error

	// Dequeue retrieves the next job from the queue
	Dequeue(ctx context.Context, jobType string) (*QueueJob, error)

	// Complete marks a job as completed
	Complete(ctx context.Context, jobID string) error

	// Fail marks a job as failed with error details
	Fail(ctx context.Context, jobID string, errorMsg string) error

	// GetPendingCount returns the number of pending jobs
	GetPendingCount(ctx context.Context, jobType string) (int, error)
}

// QueueJob represents a job in the processing queue
type QueueJob struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Status    string                 `json:"status"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt *string                `json:"updated_at"`
}

// Job types
const (
	JobTypeExtractMetadata = "extract_metadata"
	JobTypeProcessKnok     = "process_knok"
	JobTypeNotifyComplete  = "notify_complete"
)

// Job statuses
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

package worker

import (
	"context"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// JobProcessor handles different types of background jobs
type JobProcessor struct {
	logger     *slog.Logger
	knokRepo   domain.KnokRepository
	serverRepo domain.ServerRepository
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
) *JobProcessor {
	return &JobProcessor{
		logger:     logger,
		knokRepo:   knokRepo,
		serverRepo: serverRepo,
	}
}

// ProcessMetadataExtraction processes metadata extraction jobs
func (p *JobProcessor) ProcessMetadataExtraction(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// Extract job parameters
	knokIDStr, ok := payload["knok_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid knok_id in payload")
	}

	knokID, err := uuid.Parse(knokIDStr)
	if err != nil {
		return fmt.Errorf("invalid knok_id format: %w", err)
	}

	url, ok := payload["url"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid url in payload")
	}

	platform, ok := payload["platform"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid platform in payload")
	}

	logger.Info("Processing metadata extraction job",
		"knok_id", knokID,
		"url", url,
		"platform", platform,
	)

	// Update knok status to processing (if knok repo is available)
	if p.knokRepo != nil {
		if err := p.knokRepo.UpdateExtractionStatus(ctx, knokID, domain.ExtractionStatusProcessing); err != nil {
			logger.Warn("Failed to update knok status to processing", "error", err)
		}
	}

	// Simulate metadata extraction (replace with actual implementation)
	time.Sleep(2 * time.Second)

	// For now, just create basic metadata
	metadata := map[string]interface{}{
		"title":         "Sample Title",                  // This would come from actual extraction
		"duration":      180,                             // This would come from actual extraction
		"thumbnail_url": "https://example.com/thumb.jpg", // This would come from actual extraction
		"extracted_at":  time.Now().Unix(),
	}

	// Update knok with extracted metadata (if knok repo is available)
	if p.knokRepo != nil {
		knok, err := p.knokRepo.GetByID(ctx, knokID)
		if err != nil {
			return fmt.Errorf("failed to get knok for update: %w", err)
		}

		// Update knok fields with extracted metadata
		if title, ok := metadata["title"].(string); ok {
			knok.Title = &title
		}
		if duration, ok := metadata["duration"].(int); ok {
			knok.Duration = &duration
		}
		if thumbnailURL, ok := metadata["thumbnail_url"].(string); ok {
			knok.ThumbnailURL = &thumbnailURL
		}

		// Update metadata field
		knok.Metadata = map[string]interface{}{
			"extraction_method": "worker_processor",
			"extraction_time":   time.Now().Unix(),
			"raw_metadata":      metadata,
		}

		// Update extraction status
		knok.ExtractionStatus = domain.ExtractionStatusComplete

		// Update knok in database
		if err := p.knokRepo.Update(ctx, knok); err != nil {
			return fmt.Errorf("failed to update knok: %w", err)
		}

		logger.Info("Metadata extraction completed successfully",
			"knok_id", knokID,
			"title", knok.Title,
			"duration", knok.Duration,
		)
	} else {
		logger.Info("Metadata extraction completed (no knok repo available)",
			"knok_id", knokID,
		)
	}

	return nil
}

// ProcessKnok processes knok processing jobs
func (p *JobProcessor) ProcessKnok(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// This would handle additional knok processing beyond metadata extraction
	// For now, just log that we received the job
	logger.Info("Processing knok job", "payload", payload)
	return nil
}

// ProcessNotification processes notification jobs
func (p *JobProcessor) ProcessNotification(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// This would handle sending notifications about completed jobs
	// For now, just log that we received the job
	logger.Info("Processing notification job", "payload", payload)
	return nil
}

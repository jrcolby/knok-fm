package worker

import (
	"context"
	"fmt"
	"knock-fm/internal/config"
	"knock-fm/internal/domain"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// WorkerService processes background jobs
type WorkerService struct {
	config *config.Config
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc

	// Repositories
	knokRepo   domain.KnokRepository
	serverRepo domain.ServerRepository
	queueRepo  domain.QueueRepository

	// Discord session for notifications
	discordSession *discordgo.Session

	// Job processor
	processor *JobProcessor

	// WorkerStats tracks worker performance metrics
	stats *WorkerStats
}

// WorkerStats tracks worker performance metrics
type WorkerStats struct {
	JobsProcessed  int64
	JobsSucceeded  int64
	JobsFailed     int64
	LastJobTime    time.Time
	AverageJobTime time.Duration
}

// New creates a new worker service
func New(
	config *config.Config,
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
	queueRepo domain.QueueRepository,
) (*WorkerService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create Discord session for notifications (optional)
	var discordSession *discordgo.Session
	if config.DiscordToken != "" {
		var err error
		discordSession, err = discordgo.New("Bot " + config.DiscordToken)
		if err != nil {
			logger.Warn("Failed to create Discord session for notifications", "error", err)
		}
	}

	workerService := &WorkerService{
		config:         config,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		knokRepo:       knokRepo,
		serverRepo:     serverRepo,
		queueRepo:      queueRepo,
		discordSession: discordSession,
		stats:          &WorkerStats{},
	}

	// Create job processor
	processor := NewJobProcessor(logger, knokRepo, serverRepo)
	workerService.processor = processor

	return workerService, nil
}

// Start begins processing jobs
func (w *WorkerService) Start() error {
	w.logger.Info("Starting worker service...")

	// Start job processing goroutines
	go w.processJobs()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	w.logger.Info("Worker service is running. Press Ctrl+C to stop.")
	<-stop

	w.logger.Info("Shutting down worker service...")
	return w.Stop()
}

// Stop gracefully shuts down the worker service
func (w *WorkerService) Stop() error {
	w.logger.Info("Stopping worker service...")

	// Cancel context to stop all goroutines
	w.cancel()

	// Close Discord session if open
	if w.discordSession != nil {
		w.discordSession.Close()
	}

	w.logger.Info("Worker service stopped")
	return nil
}

// processJobs continuously processes jobs from the queue
func (w *WorkerService) processJobs() {
	ticker := time.NewTicker(5 * time.Second) // Check for jobs every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Job processing stopped")
			return
		case <-ticker.C:
			w.processPendingJobs()
		}
	}
}

// processPendingJobs processes all pending jobs of a specific type
func (w *WorkerService) processPendingJobs() {
	// Process metadata extraction jobs
	w.processJobType(domain.JobTypeExtractMetadata)

	// Process knok processing jobs
	w.processJobType(domain.JobTypeProcessKnok)

	// Process notification jobs
	w.processJobType(domain.JobTypeNotifyComplete)
}

// processJobType processes all pending jobs of a specific type
func (w *WorkerService) processJobType(jobType string) {
	ctx := w.ctx

	// Get pending job count
	pendingCount, err := w.queueRepo.GetPendingCount(ctx, jobType)
	if err != nil {
		w.logger.Error("Failed to get pending job count",
			"error", err,
			"job_type", jobType,
		)
		return
	}

	if pendingCount == 0 {
		return
	}

	w.logger.Debug("Processing pending jobs",
		"job_type", jobType,
		"count", pendingCount,
	)

	// Process jobs (limit to 10 per cycle to avoid overwhelming the system)
	maxJobs := 10
	if pendingCount < maxJobs {
		maxJobs = pendingCount
	}

	for i := 0; i < maxJobs; i++ {
		// Try to dequeue a job
		job, err := w.queueRepo.Dequeue(ctx, jobType)
		if err != nil {
			if err.Error() == "no jobs available" {
				break // No more jobs to process
			}
			w.logger.Error("Failed to dequeue job",
				"error", err,
				"job_type", jobType,
			)
			continue
		}

		if job == nil {
			break // No more jobs
		}

		// Process the job
		w.processJob(job)
	}
}

// processJob processes a single job
func (w *WorkerService) processJob(job *domain.QueueJob) {
	startTime := time.Now()
	jobLogger := w.logger.With(
		"job_id", job.ID,
		"job_type", job.Type,
	)

	jobLogger.Info("Processing job")

	// Update job status to processing
	if err := w.queueRepo.Complete(w.ctx, job.ID); err != nil {
		jobLogger.Error("Failed to mark job as processing", "error", err)
	}

	// Process the job based on type
	var processingErr error
	switch job.Type {
	case domain.JobTypeExtractMetadata:
		processingErr = w.processor.ProcessMetadataExtraction(w.ctx, job.Payload, jobLogger)
	case domain.JobTypeProcessKnok:
		processingErr = w.processor.ProcessKnok(w.ctx, job.Payload, jobLogger)
	case domain.JobTypeNotifyComplete:
		processingErr = w.processor.ProcessNotification(w.ctx, job.Payload, jobLogger)
	default:
		processingErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Update job status based on result
	if processingErr != nil {
		jobLogger.Error("Job processing failed", "error", processingErr)

		// Mark job as failed
		if err := w.queueRepo.Fail(w.ctx, job.ID, processingErr.Error()); err != nil {
			jobLogger.Error("Failed to mark job as failed", "error", err)
		}

		// Update stats
		w.stats.JobsFailed++
	} else {
		jobLogger.Info("Job processed successfully")

		// Mark job as completed
		if err := w.queueRepo.Complete(w.ctx, job.ID); err != nil {
			jobLogger.Error("Failed to mark job as completed", "error", err)
		}

		// Update stats
		w.stats.JobsSucceeded++
	}

	// Update overall stats
	w.stats.JobsProcessed++
	w.stats.LastJobTime = time.Now()

	// Calculate average job time
	jobDuration := time.Since(startTime)
	if w.stats.JobsProcessed > 1 {
		w.stats.AverageJobTime = time.Duration(
			(int64(w.stats.AverageJobTime) + int64(jobDuration)) / w.stats.JobsProcessed,
		)
	} else {
		w.stats.AverageJobTime = jobDuration
	}

	jobLogger.Debug("Job processing completed",
		"duration", jobDuration,
		"success", processingErr == nil,
	)
}

// GetStats returns current worker statistics
func (w *WorkerService) GetStats() *WorkerStats {
	return w.stats
}

// HealthCheck performs a health check on the worker service
func (w *WorkerService) HealthCheck() error {
	// Check if context is cancelled
	if w.ctx.Err() != nil {
		return fmt.Errorf("worker context cancelled: %w", w.ctx.Err())
	}

	// Check queue connectivity
	if _, err := w.queueRepo.GetPendingCount(w.ctx, domain.JobTypeExtractMetadata); err != nil {
		return fmt.Errorf("queue connectivity check failed: %w", err)
	}

	// Check database connectivity (if knokRepo available)
	if w.knokRepo != nil {
		// Try a simple query to check connectivity
		// This is a placeholder - implement actual health check query
	}

	return nil
}

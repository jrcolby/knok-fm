package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// QueueRepository implements the domain.QueueRepository interface using Redis
type QueueRepository struct {
	client *redis.Client
	logger *slog.Logger
}

// NewQueueRepository creates a new Redis queue repository
func NewQueueRepository(client *redis.Client, logger *slog.Logger) *QueueRepository {
	return &QueueRepository{
		client: client,
		logger: logger,
	}
}

// Redis key patterns
const (
	queueKeyPrefix   = "queue:"      // queue:job_type
	jobKeyPrefix     = "job:"        // job:job_id
	processingPrefix = "processing:" // processing:job_type
	retryKeyPrefix   = "retry:"      // retry:job_type
	deadLetterPrefix = "dead:"       // dead:job_type
	statsKeyPrefix   = "stats:"      // stats:job_type
)

// Job retry configuration
const (
	maxRetries        = 5
	initialBackoffSec = 1
	maxBackoffSec     = 300       // 5 minutes
	jobTTLSec         = 3600 * 24 // 24 hours
)

// QueueJob represents a job in the Redis queue with metadata
type QueueJob struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	NextRetry  *time.Time             `json:"next_retry,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// Enqueue adds a new job to the queue
func (r *QueueRepository) Enqueue(ctx context.Context, jobType string, payload interface{}) error {
	// Convert payload to map[string]interface{}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
		return fmt.Errorf("failed to unmarshal payload to map: %w", err)
	}

	// Create job
	job := &QueueJob{
		ID:         uuid.New().String(),
		Type:       jobType,
		Payload:    payloadMap,
		Status:     domain.JobStatusPending,
		CreatedAt:  time.Now(),
		RetryCount: 0,
		MaxRetries: maxRetries,
	}

	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	pipe := r.client.TxPipeline()

	// Store job metadata in hash
	jobKey := jobKeyPrefix + job.ID
	pipe.HMSet(ctx, jobKey, map[string]interface{}{
		"data":        string(jobData),
		"status":      job.Status,
		"type":        job.Type,
		"created_at":  job.CreatedAt.Unix(),
		"retry_count": job.RetryCount,
	})
	pipe.Expire(ctx, jobKey, time.Duration(jobTTLSec)*time.Second)

	// Add job ID to queue
	queueKey := queueKeyPrefix + jobType
	pipe.LPush(ctx, queueKey, job.ID)

	// Update stats
	statsKey := statsKeyPrefix + jobType
	pipe.HIncrBy(ctx, statsKey, "total_enqueued", 1)
	pipe.HIncrBy(ctx, statsKey, "pending", 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	r.logger.Info("Job enqueued",
		"job_id", job.ID,
		"job_type", jobType,
		"payload_size", len(payloadBytes),
	)

	return nil
}

// Dequeue retrieves the next job from the queue with blocking
func (r *QueueRepository) Dequeue(ctx context.Context, jobType string) (*domain.QueueJob, error) {
	queueKey := queueKeyPrefix + jobType
	processingKey := processingPrefix + jobType

	// Use BRPOPLPUSH for atomic move from queue to processing list
	// This ensures jobs aren't lost if worker crashes
	result, err := r.client.BRPopLPush(ctx, queueKey, processingKey, 30*time.Second).Result()
	if err != nil {
		if err == redis.Nil {
			// No jobs available (timeout)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	jobID := result

	// Get job data
	jobKey := jobKeyPrefix + jobID
	jobData, err := r.client.HGet(ctx, jobKey, "data").Result()
	if err != nil {
		if err == redis.Nil {
			r.logger.Warn("Job data not found, removing from processing", "job_id", jobID)
			r.client.LRem(ctx, processingKey, 1, jobID)
			return nil, fmt.Errorf("job data not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	// Parse job
	var queueJob QueueJob
	if err := json.Unmarshal([]byte(jobData), &queueJob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status to processing
	now := time.Now()
	queueJob.Status = domain.JobStatusProcessing
	queueJob.UpdatedAt = &now

	// Update job in Redis
	updatedData, _ := json.Marshal(queueJob)
	pipe := r.client.TxPipeline()
	pipe.HMSet(ctx, jobKey, map[string]interface{}{
		"data":       string(updatedData),
		"status":     queueJob.Status,
		"updated_at": now.Unix(),
	})

	// Update stats
	statsKey := statsKeyPrefix + jobType
	pipe.HIncrBy(ctx, statsKey, "pending", -1)
	pipe.HIncrBy(ctx, statsKey, "processing", 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to update job status", "error", err, "job_id", jobID)
	}

	// Convert to domain.QueueJob
	domainJob := &domain.QueueJob{
		ID:        queueJob.ID,
		Type:      queueJob.Type,
		Payload:   queueJob.Payload,
		Status:    queueJob.Status,
		CreatedAt: queueJob.CreatedAt.Format(time.RFC3339),
	}
	if queueJob.UpdatedAt != nil {
		updatedAtStr := queueJob.UpdatedAt.Format(time.RFC3339)
		domainJob.UpdatedAt = &updatedAtStr
	}

	r.logger.Info("Job dequeued",
		"job_id", queueJob.ID,
		"job_type", jobType,
		"retry_count", queueJob.RetryCount,
	)

	return domainJob, nil
}

// Complete marks a job as completed and removes it from processing
func (r *QueueRepository) Complete(ctx context.Context, jobID string) error {
	jobKey := jobKeyPrefix + jobID

	// Get job to determine type
	jobData, err := r.client.HGet(ctx, jobKey, "data").Result()
	if err != nil {
		return fmt.Errorf("failed to get job for completion: %w", err)
	}

	var job QueueJob
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job for completion: %w", err)
	}

	processingKey := processingPrefix + job.Type
	now := time.Now()

	// Update job status
	job.Status = domain.JobStatusCompleted
	job.UpdatedAt = &now

	updatedData, _ := json.Marshal(job)

	pipe := r.client.TxPipeline()

	// Update job data
	pipe.HMSet(ctx, jobKey, map[string]interface{}{
		"data":       string(updatedData),
		"status":     job.Status,
		"updated_at": now.Unix(),
	})

	// Remove from processing list
	pipe.LRem(ctx, processingKey, 1, jobID)

	// Update stats
	statsKey := statsKeyPrefix + job.Type
	pipe.HIncrBy(ctx, statsKey, "processing", -1)
	pipe.HIncrBy(ctx, statsKey, "completed", 1)

	// Set shorter TTL for completed jobs
	pipe.Expire(ctx, jobKey, time.Hour*6)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	r.logger.Info("Job completed", "job_id", jobID, "job_type", job.Type)
	return nil
}

// Fail marks a job as failed and handles retry logic
func (r *QueueRepository) Fail(ctx context.Context, jobID string, errorMsg string) error {
	jobKey := jobKeyPrefix + jobID

	// Get current job data
	jobData, err := r.client.HGet(ctx, jobKey, "data").Result()
	if err != nil {
		return fmt.Errorf("failed to get job for failure: %w", err)
	}

	var job QueueJob
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job for failure: %w", err)
	}

	processingKey := processingPrefix + job.Type
	now := time.Now()

	// Update job with error
	job.Error = errorMsg
	job.UpdatedAt = &now
	job.RetryCount++

	pipe := r.client.TxPipeline()

	// Check if we should retry
	if job.RetryCount <= job.MaxRetries {
		// Calculate backoff delay with exponential backoff
		backoffSec := int(math.Min(
			float64(initialBackoffSec)*math.Pow(2, float64(job.RetryCount-1)),
			float64(maxBackoffSec),
		))
		nextRetry := now.Add(time.Duration(backoffSec) * time.Second)
		job.NextRetry = &nextRetry
		job.Status = domain.JobStatusPending

		// Re-queue the job for retry (with delay)
		retryKey := retryKeyPrefix + job.Type
		pipe.ZAdd(ctx, retryKey, redis.Z{
			Score:  float64(nextRetry.Unix()),
			Member: jobID,
		})

		r.logger.Info("Job scheduled for retry",
			"job_id", jobID,
			"job_type", job.Type,
			"retry_count", job.RetryCount,
			"next_retry", nextRetry,
			"error", errorMsg,
		)
	} else {
		// Max retries exceeded, move to dead letter queue
		job.Status = domain.JobStatusFailed
		deadKey := deadLetterPrefix + job.Type
		pipe.LPush(ctx, deadKey, jobID)

		// Update stats
		statsKey := statsKeyPrefix + job.Type
		pipe.HIncrBy(ctx, statsKey, "failed", 1)

		r.logger.Error("Job failed permanently",
			"job_id", jobID,
			"job_type", job.Type,
			"retry_count", job.RetryCount,
			"error", errorMsg,
		)
	}

	// Update job data
	updatedData, _ := json.Marshal(job)
	pipe.HMSet(ctx, jobKey, map[string]interface{}{
		"data":        string(updatedData),
		"status":      job.Status,
		"updated_at":  now.Unix(),
		"retry_count": job.RetryCount,
		"error":       errorMsg,
	})

	// Remove from processing list
	pipe.LRem(ctx, processingKey, 1, jobID)

	// Update stats
	statsKey := statsKeyPrefix + job.Type
	pipe.HIncrBy(ctx, statsKey, "processing", -1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to handle job failure: %w", err)
	}

	return nil
}

// GetPendingCount returns the number of pending jobs for a job type
func (r *QueueRepository) GetPendingCount(ctx context.Context, jobType string) (int, error) {
	queueKey := queueKeyPrefix + jobType
	count, err := r.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending count: %w", err)
	}
	return int(count), nil
}

// ProcessRetryJobs moves jobs from retry queue back to main queue when ready
func (r *QueueRepository) ProcessRetryJobs(ctx context.Context, jobType string) error {
	retryKey := retryKeyPrefix + jobType
	queueKey := queueKeyPrefix + jobType
	now := time.Now()

	// Get jobs ready for retry (score <= current timestamp)
	jobs, err := r.client.ZRangeByScoreWithScores(ctx, retryKey, &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now.Unix(), 10),
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to get retry jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil // No jobs ready for retry
	}

	pipe := r.client.TxPipeline()

	for _, job := range jobs {
		jobID := job.Member.(string)

		// Move from retry queue to main queue
		pipe.ZRem(ctx, retryKey, jobID)
		pipe.LPush(ctx, queueKey, jobID)

		// Update stats
		statsKey := statsKeyPrefix + jobType
		pipe.HIncrBy(ctx, statsKey, "pending", 1)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to process retry jobs: %w", err)
	}

	r.logger.Info("Processed retry jobs",
		"job_type", jobType,
		"count", len(jobs),
	)

	return nil
}

// GetQueueStats returns statistics for a job type
func (r *QueueRepository) GetQueueStats(ctx context.Context, jobType string) (map[string]int64, error) {
	statsKey := statsKeyPrefix + jobType
	stats, err := r.client.HGetAll(ctx, statsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	result := make(map[string]int64)
	for key, value := range stats {
		if val, err := strconv.ParseInt(value, 10, 64); err == nil {
			result[key] = val
		}
	}

	// Add current queue lengths
	queueKey := queueKeyPrefix + jobType
	processingKey := processingPrefix + jobType
	retryKey := retryKeyPrefix + jobType
	deadKey := deadLetterPrefix + jobType

	if pending, err := r.client.LLen(ctx, queueKey).Result(); err == nil {
		result["current_pending"] = pending
	}

	if processing, err := r.client.LLen(ctx, processingKey).Result(); err == nil {
		result["current_processing"] = processing
	}

	if retrying, err := r.client.ZCard(ctx, retryKey).Result(); err == nil {
		result["current_retrying"] = retrying
	}

	if dead, err := r.client.LLen(ctx, deadKey).Result(); err == nil {
		result["current_dead"] = dead
	}

	return result, nil
}

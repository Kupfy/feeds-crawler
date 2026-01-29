package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RecipeJob represents a recipe processing job
type RecipeJob struct {
	URL        string    `json:"url"`
	SiteID     string    `json:"site_id,omitempty"`
	Priority   int       `json:"priority,omitempty"`
	Retries    int       `json:"retries,omitempty"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// RedisQueue provides a Redis-backed message queue for recipe processing
type RedisQueue interface {
	// Enqueue adds a URL to the processing queue
	Enqueue(ctx context.Context, url string) error

	// EnqueueJob adds a full job to the queue
	EnqueueJob(ctx context.Context, job *RecipeJob) error

	// Dequeue retrieves and removes a job from the queue (blocking)
	Dequeue(ctx context.Context, timeout time.Duration) (*RecipeJob, error)

	// DequeueNonBlocking retrieves and removes a job without blocking
	DequeueNonBlocking(ctx context.Context) (*RecipeJob, error)

	// Peek views the next job without removing it
	Peek(ctx context.Context) (*RecipeJob, error)

	// Size returns the number of jobs in the queue
	Size(ctx context.Context) (int64, error)

	// Ack acknowledges successful processing (removes from processing set)
	Ack(ctx context.Context, job *RecipeJob) error

	// Nack rejects a job and optionally re-queues it
	Nack(ctx context.Context, job *RecipeJob, requeue bool) error

	// Clear removes all jobs from the queue
	Clear(ctx context.Context) error

	// Close closes the Redis connection
	Close() error

	// RecoverStaleJobs moves jobs from processing back to queue if they've been stuck'
	RecoverStaleJobs(ctx context.Context, staleDuration time.Duration) (int, error)
}

type redisQueue struct {
	client        *redis.Client
	queueKey      string
	processingKey string
	deadLetterKey string
	maxRetries    int
}

// RedisQueueConfig holds configuration for the Redis queue
type RedisQueueConfig struct {
	Addr       string // Redis address (e.g., "localhost:6379")
	Password   string
	DB         int
	QueueName  string // Queue name (e.g., "recipe_jobs")
	MaxRetries int    // Maximum retry attempts before dead letter
}

// NewRedisQueue creates a new Redis-backed queue
func NewRedisQueue(config RedisQueueConfig) (RedisQueue, error) {
	if config.QueueName == "" {
		config.QueueName = "recipe_jobs"
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisQueue{
		client:        client,
		queueKey:      config.QueueName,
		processingKey: config.QueueName + ":processing",
		deadLetterKey: config.QueueName + ":dead_letter",
		maxRetries:    config.MaxRetries,
	}, nil
}

// Enqueue adds a URL to the processing queue
func (q *redisQueue) Enqueue(ctx context.Context, url string) error {
	job := &RecipeJob{
		URL:        url,
		EnqueuedAt: time.Now().UTC(),
	}
	return q.EnqueueJob(ctx, job)
}

// EnqueueJob adds a full job to the queue
func (q *redisQueue) EnqueueJob(ctx context.Context, job *RecipeJob) error {
	if job.URL == "" {
		return fmt.Errorf("job URL cannot be empty")
	}

	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = time.Now().UTC()
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Use RPUSH for FIFO queue (or LPUSH for LIFO/stack)
	if err := q.client.RPush(ctx, q.queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// Dequeue retrieves and removes a job from the queue (blocking)
func (q *redisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*RecipeJob, error) {
	// BLPOP blocks until an item is available or timeout
	result, err := q.client.BLPop(ctx, timeout, q.queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Timeout, no job available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// result[0] is the key, result[1] is the value
	if len(result) < 2 {
		return nil, fmt.Errorf("unexpected redis response")
	}

	var job RecipeJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Move to processing set for reliability (optional, for at-least-once semantics)
	if err := q.addToProcessing(ctx, &job); err != nil {
		// Log error but continue - job is already dequeued
		fmt.Printf("warning: failed to add job to processing set: %v\n", err)
	}

	return &job, nil
}

// DequeueNonBlocking retrieves and removes a job without blocking
func (q *redisQueue) DequeueNonBlocking(ctx context.Context) (*RecipeJob, error) {
	result, err := q.client.LPop(ctx, q.queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Queue is empty
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	var job RecipeJob
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	if err := q.addToProcessing(ctx, &job); err != nil {
		fmt.Printf("warning: failed to add job to processing set: %v\n", err)
	}

	return &job, nil
}

// Peek views the next job without removing it
func (q *redisQueue) Peek(ctx context.Context) (*RecipeJob, error) {
	result, err := q.client.LIndex(ctx, q.queueKey, 0).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Queue is empty
		}
		return nil, fmt.Errorf("failed to peek job: %w", err)
	}

	var job RecipeJob
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// Size returns the number of jobs in the queue
func (q *redisQueue) Size(ctx context.Context) (int64, error) {
	size, err := q.client.LLen(ctx, q.queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return size, nil
}

// Ack acknowledges successful processing
func (q *redisQueue) Ack(ctx context.Context, job *RecipeJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Remove from processing set
	if err := q.client.SRem(ctx, q.processingKey, data).Err(); err != nil {
		return fmt.Errorf("failed to remove job from processing set: %w", err)
	}

	return nil
}

// Nack rejects a job and optionally re-queues it
func (q *redisQueue) Nack(ctx context.Context, job *RecipeJob, requeue bool) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Remove from processing set
	if err := q.client.SRem(ctx, q.processingKey, data).Err(); err != nil {
		return fmt.Errorf("failed to remove job from processing set: %w", err)
	}

	if requeue {
		job.Retries++

		// Check if max retries exceeded
		if job.Retries >= q.maxRetries {
			// Move to dead letter queue
			if err := q.client.RPush(ctx, q.deadLetterKey, data).Err(); err != nil {
				return fmt.Errorf("failed to add job to dead letter queue: %w", err)
			}
			return nil
		}

		// Re-queue the job
		return q.EnqueueJob(ctx, job)
	}

	return nil
}

// Clear removes all jobs from the queue
func (q *redisQueue) Clear(ctx context.Context) error {
	pipe := q.client.Pipeline()
	pipe.Del(ctx, q.queueKey)
	pipe.Del(ctx, q.processingKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (q *redisQueue) Close() error {
	return q.client.Close()
}

// addToProcessing adds a job to the processing set
func (q *redisQueue) addToProcessing(ctx context.Context, job *RecipeJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.SAdd(ctx, q.processingKey, data).Err()
}

// RecoverStaleJobs moves jobs from processing back to queue if they've been stuck
// This should be called periodically to handle worker crashes
func (q *redisQueue) RecoverStaleJobs(ctx context.Context, staleDuration time.Duration) (int, error) {
	members, err := q.client.SMembers(ctx, q.processingKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get processing jobs: %w", err)
	}

	recovered := 0
	now := time.Now().UTC()

	for _, member := range members {
		var job RecipeJob
		if err := json.Unmarshal([]byte(member), &job); err != nil {
			continue
		}

		// Check if job has been processing too long
		if now.Sub(job.EnqueuedAt) > staleDuration {
			// Remove from processing
			q.client.SRem(ctx, q.processingKey, member)

			// Re-queue
			if err := q.EnqueueJob(ctx, &job); err != nil {
				fmt.Printf("warning: failed to recover stale job: %v\n", err)
				continue
			}
			recovered++
		}
	}

	return recovered, nil
}

package jobqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	maxRetries     = 3
	idempotencyTTL = 24 * time.Hour
)

var backoffDurations = [maxRetries]time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
}

// Queue manages a pool of workers that process Jobs.
// Workers read from a buffered channel; the pool size and channel buffer are
// both configurable at construction time.
type Queue struct {
	jobs        chan Job
	redisClient *redis.Client
	gatewayURL  string
	httpClient  *http.Client
	wg          sync.WaitGroup
}

// New creates a Queue, starts workerCount workers, and returns it.
// bufferSize controls the depth of the buffered job channel (backpressure).
func New(workerCount, bufferSize int, redisClient *redis.Client, gatewayURL string) *Queue {
	q := &Queue{
		jobs:        make(chan Job, bufferSize),
		redisClient: redisClient,
		gatewayURL:  gatewayURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
	for i := 0; i < workerCount; i++ {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			for job := range q.jobs {
				q.processJob(job)
			}
		}()
	}
	return q
}

// Stop closes the job channel and waits for all in-flight workers to finish.
func (q *Queue) Stop() {
	close(q.jobs)
	q.wg.Wait()
}

// Enqueue checks the idempotency key in Redis and, if the job has not been
// processed already, places it onto the worker channel.
func (q *Queue) Enqueue(job Job) {
	ctx := context.Background()
	idemRedisKey := "notification:idem:" + job.IdempotencyKey

	val, err := q.redisClient.Get(ctx, idemRedisKey).Result()
	if err == nil && val == "done" {
		// Already processed — drop silently with an info log.
		writeLog("info", job.IdempotencyKey, 1, "dropped", "duplicate idempotency key — job dropped silently")
		return
	}

	writeLog("info", job.IdempotencyKey, 1, "enqueued", "")

	select {
	case q.jobs <- job:
	default:
		// Channel is full — log the backpressure event and drop.
		log.Printf("jobqueue: WARNING: worker channel full, dropping job %s", job.IdempotencyKey)
	}
}

// processJob executes a job with up to maxRetries attempts and exponential backoff.
func (q *Queue) processJob(job Job) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		writeLog("info", job.IdempotencyKey, attempt, "processing", "")

		err := q.callGateway(job)
		if err == nil {
			// Mark as done in Redis (TTL: 24 h) so replays are idempotent.
			ctx := context.Background()
			idemKey := "notification:idem:" + job.IdempotencyKey
			if setErr := q.redisClient.Set(ctx, idemKey, "done", idempotencyTTL).Err(); setErr != nil {
				log.Printf("jobqueue: WARNING: failed to store idempotency key %s: %v", job.IdempotencyKey, setErr)
			}
			writeLog("info", job.IdempotencyKey, attempt, "success", "")
			return
		}

		if attempt < maxRetries {
			writeLog("warn", job.IdempotencyKey, attempt, "retry", err.Error())
			time.Sleep(backoffDurations[attempt-1])
		}
	}

	// All attempts exhausted — write dead-letter entry to stderr.
	writeLog("error", job.IdempotencyKey, maxRetries, "dead_letter", "exceeded max retries")
}

// gatewayPayload is the JSON body sent to POST /notify.
type gatewayPayload struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

// callGateway sends the job to the Mock Notification Gateway and returns nil on HTTP 200.
func (q *Queue) callGateway(job Job) error {
	payload := gatewayPayload{
		IdempotencyKey: job.IdempotencyKey,
		Channel:        job.Channel,
		Recipient:      job.Recipient,
		Message:        job.Message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := q.httpClient.Post(q.gatewayURL+"/notify", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("gateway returned HTTP %d", resp.StatusCode)
}

// jobLogLine is the structured log format required by the assignment (Section 6.4).
type jobLogLine struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	JobID   string `json:"job_id"`
	Attempt int    `json:"attempt"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// writeLog emits one structured JSON line.
// dead_letter entries go to stderr; all others go to stdout.
func writeLog(level, jobID string, attempt int, jobStatus, errMsg string) {
	line := jobLogLine{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		JobID:   jobID,
		Attempt: attempt,
		Status:  jobStatus,
		Error:   errMsg,
	}
	b, _ := json.Marshal(line)
	if jobStatus == "dead_letter" {
		fmt.Fprintln(os.Stderr, string(b))
	} else {
		fmt.Println(string(b))
	}
}

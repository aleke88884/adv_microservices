package main

import (
	"context"
	"log"
	"notification-service/internal/jobqueue"
	"notification-service/internal/subscriber"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

const maxRetries = 7 // 1s + 2s + 4s + 8s + 16s + 32s + 64s ≈ 127 s total

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	workerPoolSize := 3
	if v := os.Getenv("WORKER_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workerPoolSize = n
		}
	}

	// ── Redis (required for job queue idempotency) ───────────────────────────
	var queue *jobqueue.Queue
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("notification-service: REDIS_URL not set — job queue disabled")
	} else {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("notification-service: WARNING: invalid REDIS_URL %q: %v — job queue disabled", redisURL, err)
		} else {
			rdb := redis.NewClient(opt)
			pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if pingErr := rdb.Ping(pingCtx).Err(); pingErr != nil {
				log.Printf("notification-service: WARNING: cannot reach Redis at %s: %v — job queue disabled", redisURL, pingErr)
			} else {
				log.Printf("notification-service: connected to Redis at %s", redisURL)
				bufferSize := workerPoolSize * 10
				queue = jobqueue.New(workerPoolSize, bufferSize, rdb, gatewayURL)
				log.Printf("notification-service: job queue started with %d workers (buffer %d)", workerPoolSize, bufferSize)
			}
		}
	}

	// ── Connect with exponential backoff ─────────────────────────────────────
	nc, err := subscriber.Connect(natsURL, maxRetries)
	if err != nil {
		log.Fatalf("notification-service: failed to connect to NATS after %d attempts: %v", maxRetries, err)
	}
	defer func() {
		if err := nc.Drain(); err != nil {
			log.Printf("notification-service: error draining NATS connection: %v", err)
		}
		log.Println("notification-service: NATS connection closed")
	}()

	// ── Subscribe to all event subjects ──────────────────────────────────────
	subs, err := subscriber.Subscribe(nc, queue)
	if err != nil {
		log.Fatalf("notification-service: failed to subscribe: %v", err)
	}
	log.Printf("notification-service: listening for events on %d subjects", len(subs))

	// ── Block until SIGTERM or SIGINT ────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("notification-service: received shutdown signal — draining in-flight messages")

	if queue != nil {
		log.Println("notification-service: stopping job queue workers")
		queue.Stop()
		log.Println("notification-service: job queue stopped")
	}
}

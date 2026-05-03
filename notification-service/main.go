package main

import (
	"log"
	"notification-service/internal/subscriber"
	"os"
	"os/signal"
	"syscall"
)

const maxRetries = 7 // delays: 1s, 2s, 4s, 8s, 16s, 32s, 64s ≈ 127 s total

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Connect with exponential backoff.
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

	// Subscribe to all event subjects.
	subs, err := subscriber.Subscribe(nc)
	if err != nil {
		log.Fatalf("notification-service: failed to subscribe: %v", err)
	}
	log.Printf("notification-service: listening for events on %d subjects", len(subs))

	// Block until SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("notification-service: received shutdown signal — draining in-flight messages")
}

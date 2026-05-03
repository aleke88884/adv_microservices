package subscriber

import (
	"encoding/json"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
)

// logLine is the structured log record printed to stdout for every received event.
type logLine struct {
	Time    string          `json:"time"`
	Subject string          `json:"subject"`
	Event   json.RawMessage `json:"event"`
}

// subjects the Notification Service subscribes to.
var subjects = []string{
	"doctors.created",
	"appointments.created",
	"appointments.status_updated",
}

// Connect establishes a NATS connection with exponential backoff.
// It retries up to maxAttempts times (delays: 1s, 2s, 4s, 8s, …).
// Returns an error if all attempts are exhausted.
func Connect(url string, maxAttempts int) (*nats.Conn, error) {
	delay := time.Second
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		nc, err := nats.Connect(url)
		if err == nil {
			log.Printf("notification-service: connected to NATS at %s (attempt %d)", url, attempt)
			return nc, nil
		}
		lastErr = err
		log.Printf("notification-service: cannot connect to NATS (attempt %d/%d): %v — retrying in %s",
			attempt, maxAttempts, err, delay)
		time.Sleep(delay)
		delay *= 2
	}
	return nil, lastErr
}

// Subscribe registers message handlers for all three subjects.
// Each received message is deserialized and logged as a single JSON line to stdout.
// Returns the slice of active subscriptions so the caller can unsubscribe on shutdown.
func Subscribe(nc *nats.Conn) ([]*nats.Subscription, error) {
	var subs []*nats.Subscription
	for _, subject := range subjects {
		sub := subject // capture for closure
		s, err := nc.Subscribe(sub, func(msg *nats.Msg) {
			handleMessage(sub, msg.Data)
		})
		if err != nil {
			return subs, err
		}
		subs = append(subs, s)
		log.Printf("notification-service: subscribed to subject %q", sub)
	}
	return subs, nil
}

// handleMessage deserialises the raw JSON payload and prints one log line to stdout.
func handleMessage(subject string, data []byte) {
	// Validate that the payload is valid JSON; log an error if not.
	if !json.Valid(data) {
		log.Printf("notification-service: ERROR received invalid JSON on subject %q: %s", subject, data)
		return
	}

	line := logLine{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Subject: subject,
		Event:   json.RawMessage(data),
	}

	out, err := json.Marshal(line)
	if err != nil {
		log.Printf("notification-service: ERROR marshalling log line for subject %q: %v", subject, err)
		return
	}

	// Single JSON line written to stdout — this is the required output.
	log.SetFlags(0) // remove timestamps from the standard logger for clean output
	log.Println(string(out))
}

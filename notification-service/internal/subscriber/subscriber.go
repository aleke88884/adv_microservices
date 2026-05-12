package subscriber

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/jobqueue"
	"notification-service/internal/logger"
	"time"

	nats "github.com/nats-io/nats.go"
)

// subjects the Notification Service subscribes to.
var subjects = []string{
	"doctors.created",
	"appointments.created",
	"appointments.status_updated",
}

// statusUpdatedPayload is used to parse appointments.status_updated events.
type statusUpdatedPayload struct {
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	ID         string `json:"id"`
	DoctorID   string `json:"doctor_id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
}

// Connect establishes a NATS connection with exponential backoff.
// It retries up to maxAttempts times (delays: 1s, 2s, 4s, 8s, …).
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

// Subscribe registers message handlers for all subjects.
// queue is optional: if non-nil, appointments.status_updated events with
// new_status="done" are forwarded to the job queue.
func Subscribe(nc *nats.Conn, queue *jobqueue.Queue) ([]*nats.Subscription, error) {
	var subs []*nats.Subscription
	for _, subject := range subjects {
		sub := subject // capture for closure
		s, err := nc.Subscribe(sub, func(msg *nats.Msg) {
			handleMessage(sub, msg.Data, queue)
		})
		if err != nil {
			return subs, err
		}
		subs = append(subs, s)
		log.Printf("notification-service: subscribed to subject %q", sub)
	}
	return subs, nil
}

// handleMessage logs the event and, for status_updated=done events, enqueues a job.
func handleMessage(subject string, data []byte, queue *jobqueue.Queue) {
	if !json.Valid(data) {
		log.Printf("notification-service: ERROR received invalid JSON on subject %q: %s", subject, data)
		return
	}

	// Log the event (unchanged from Assignment 3).
	logger.LogEvent(subject, json.RawMessage(data))

	// Trigger job queue only for appointments.status_updated with new_status="done".
	if subject == "appointments.status_updated" && queue != nil {
		var evt statusUpdatedPayload
		if err := json.Unmarshal(data, &evt); err != nil {
			log.Printf("notification-service: ERROR parsing status_updated payload: %v", err)
			return
		}
		if evt.NewStatus == "done" {
			job := buildJob(evt)
			queue.Enqueue(job)
		}
	}
}

// buildJob constructs a Job from the status_updated event payload.
func buildJob(evt statusUpdatedPayload) jobqueue.Job {
	idemKey := computeIdempotencyKey(evt.EventType, evt.ID, evt.OccurredAt)
	return jobqueue.Job{
		IdempotencyKey: idemKey,
		AppointmentID:  evt.ID,
		DoctorID:       evt.DoctorID,
		OccurredAt:     evt.OccurredAt,
		Channel:        "email",
		Recipient:      "patient@clinic.kz",
		Message: fmt.Sprintf("Your appointment %s with doctor %s is complete.",
			evt.ID, evt.DoctorID),
	}
}

// computeIdempotencyKey returns the SHA-256 hex of event_type + id + occurred_at.
func computeIdempotencyKey(eventType, id, occurredAt string) string {
	h := sha256.Sum256([]byte(eventType + id + occurredAt))
	return hex.EncodeToString(h[:])
}

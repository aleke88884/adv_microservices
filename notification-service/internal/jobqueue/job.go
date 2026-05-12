// Package jobqueue implements a background worker-pool-based job queue for
// dispatching notifications to the external Mock Notification Gateway.
package jobqueue

// Job represents a single notification task derived from an appointments.status_updated event.
type Job struct {
	// IdempotencyKey is the SHA-256 hex of event_type + id + occurred_at.
	// It is used to prevent duplicate gateway calls on retry or event replay.
	IdempotencyKey string

	AppointmentID string
	DoctorID      string
	OccurredAt    string // RFC3339

	Channel   string // always "email" for this assignment
	Recipient string // always "patient@clinic.kz"
	Message   string // "Your appointment <id> with doctor <doctor_id> is complete."
}

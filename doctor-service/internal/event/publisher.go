package event

import (
	"encoding/json"
	"log"

	nats "github.com/nats-io/nats.go"
)

// Publisher is the abstraction injected into use cases.
type Publisher interface {
	Publish(subject string, payload interface{}) error
	Close()
}

// NATSPublisher publishes events to a NATS Core broker.
type NATSPublisher struct {
	conn *nats.Conn
}

// NewNATSPublisher connects to NATS and returns a NATSPublisher.
func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NATSPublisher{conn: nc}, nil
}

// Publish serialises payload as JSON and fires it to subject (fire-and-forget).
func (p *NATSPublisher) Publish(subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.conn.Publish(subject, data)
}

// Close drains the NATS connection before shutdown.
func (p *NATSPublisher) Close() {
	if err := p.conn.Drain(); err != nil {
		log.Printf("event: error draining NATS connection: %v", err)
	}
}

// NoOpPublisher is used when the broker is unavailable — it silently discards events.
type NoOpPublisher struct{}

func (p *NoOpPublisher) Publish(_ string, _ interface{}) error { return nil }
func (p *NoOpPublisher) Close()                                 {}

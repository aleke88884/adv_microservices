// Package logger provides structured JSON event logging for the Notification Service.
package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

type eventLogLine struct {
	Time    string          `json:"time"`
	Subject string          `json:"subject"`
	Event   json.RawMessage `json:"event"`
}

// LogEvent writes a single structured JSON line to stdout for the given subject and raw event payload.
func LogEvent(subject string, data json.RawMessage) {
	line := eventLogLine{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Subject: subject,
		Event:   data,
	}
	b, err := json.Marshal(line)
	if err != nil {
		fmt.Printf(`{"time":%q,"level":"error","msg":"failed to marshal log line","subject":%q}`+"\n",
			time.Now().UTC().Format(time.RFC3339), subject)
		return
	}
	fmt.Println(string(b))
}

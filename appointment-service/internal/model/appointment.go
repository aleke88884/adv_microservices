package model

import (
	"errors"
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) IsValid() bool {
	return s == StatusNew || s == StatusInProgress || s == StatusDone
}

type Appointment struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DoctorID    string    `json:"doctor_id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ValidateStatusTransition(currentStatus, newStatus Status) error {
	if currentStatus == StatusDone && newStatus == StatusNew {
		return errors.New("cannot transition from done to new")
	}
	return nil
}

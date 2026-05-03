package usecase

import (
	"appointment-service/internal/client"
	"appointment-service/internal/event"
	"appointment-service/internal/model"
	"appointment-service/internal/repository"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

// appointmentCreatedEvent is the payload published to "appointments.created".
type appointmentCreatedEvent struct {
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	DoctorID   string `json:"doctor_id"`
	Status     string `json:"status"`
}

// appointmentStatusUpdatedEvent is the payload published to "appointments.status_updated".
type appointmentStatusUpdatedEvent struct {
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	ID         string `json:"id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
}

// AppointmentUseCase contains business logic for the Appointment domain.
type AppointmentUseCase struct {
	repo         repository.AppointmentRepository
	doctorClient client.DoctorClient
	publisher    event.Publisher
}

// NewAppointmentUseCase wires up the use case with its dependencies.
func NewAppointmentUseCase(repo repository.AppointmentRepository, doctorClient client.DoctorClient, publisher event.Publisher) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:         repo,
		doctorClient: doctorClient,
		publisher:    publisher,
	}
}

// CreateAppointment validates inputs, persists the appointment, and publishes a domain event.
func (uc *AppointmentUseCase) CreateAppointment(title, description, doctorID string) (*model.Appointment, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if doctorID == "" {
		return nil, errors.New("doctor_id is required")
	}

	exists, err := uc.doctorClient.DoctorExists(doctorID)
	if err != nil {
		return nil, errors.New("failed to validate doctor: " + err.Error())
	}
	if !exists {
		return nil, errors.New("doctor does not exist")
	}

	now := time.Now()
	appointment := &model.Appointment{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(appointment); err != nil {
		return nil, err
	}

	// Publish event — best-effort, failure must not affect the RPC response.
	evt := appointmentCreatedEvent{
		EventType:  "appointments.created",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         appointment.ID,
		Title:      appointment.Title,
		DoctorID:   appointment.DoctorID,
		Status:     string(appointment.Status),
	}
	if err := uc.publisher.Publish("appointments.created", evt); err != nil {
		log.Printf("usecase: failed to publish appointments.created for appointment %s: %v", appointment.ID, err)
	}

	return appointment, nil
}

// GetAppointmentByID retrieves an appointment by its ID.
func (uc *AppointmentUseCase) GetAppointmentByID(id string) (*model.Appointment, error) {
	return uc.repo.GetByID(id)
}

// GetAllAppointments returns every appointment in the repository.
func (uc *AppointmentUseCase) GetAllAppointments() ([]*model.Appointment, error) {
	return uc.repo.GetAll()
}

// UpdateAppointmentStatus applies a status transition and publishes a domain event.
func (uc *AppointmentUseCase) UpdateAppointmentStatus(id string, newStatus model.Status) (*model.Appointment, error) {
	if !newStatus.IsValid() {
		return nil, errors.New("invalid status")
	}

	appointment, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	oldStatus := appointment.Status

	if err := model.ValidateStatusTransition(appointment.Status, newStatus); err != nil {
		return nil, err
	}

	appointment.Status = newStatus
	appointment.UpdatedAt = time.Now()

	if err := uc.repo.Update(appointment); err != nil {
		return nil, err
	}

	// Publish event — best-effort, failure must not affect the RPC response.
	evt := appointmentStatusUpdatedEvent{
		EventType:  "appointments.status_updated",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         appointment.ID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(newStatus),
	}
	if err := uc.publisher.Publish("appointments.status_updated", evt); err != nil {
		log.Printf("usecase: failed to publish appointments.status_updated for appointment %s: %v", appointment.ID, err)
	}

	return appointment, nil
}

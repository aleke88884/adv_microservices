package usecase

import (
	"doctor-service/internal/event"
	"doctor-service/internal/model"
	"doctor-service/internal/repository"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

// doctorCreatedEvent is the payload published to "doctors.created".
type doctorCreatedEvent struct {
	EventType      string `json:"event_type"`
	OccurredAt     string `json:"occurred_at"`
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

// DoctorUseCase contains the business logic for the Doctor domain.
type DoctorUseCase struct {
	repo      repository.DoctorRepository
	publisher event.Publisher
}

// NewDoctorUseCase wires up the use case with its dependencies.
func NewDoctorUseCase(repo repository.DoctorRepository, publisher event.Publisher) *DoctorUseCase {
	return &DoctorUseCase{repo: repo, publisher: publisher}
}

// CreateDoctor validates inputs, persists the doctor, and publishes a domain event.
func (uc *DoctorUseCase) CreateDoctor(fullName, specialization, email string) (*model.Doctor, error) {
	if fullName == "" {
		return nil, errors.New("full_name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if uc.repo.EmailExists(email) {
		return nil, errors.New("email must be unique")
	}

	doctor := &model.Doctor{
		ID:             uuid.New().String(),
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}

	if err := uc.repo.Create(doctor); err != nil {
		return nil, err
	}

	// Publish event — best-effort, failure must not affect the RPC response.
	evt := doctorCreatedEvent{
		EventType:      "doctors.created",
		OccurredAt:     time.Now().UTC().Format(time.RFC3339),
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
	if err := uc.publisher.Publish("doctors.created", evt); err != nil {
		log.Printf("usecase: failed to publish doctors.created for doctor %s: %v", doctor.ID, err)
	}

	return doctor, nil
}

// GetDoctorByID retrieves a doctor by its ID.
func (uc *DoctorUseCase) GetDoctorByID(id string) (*model.Doctor, error) {
	return uc.repo.GetByID(id)
}

// GetAllDoctors returns every doctor in the repository.
func (uc *DoctorUseCase) GetAllDoctors() ([]*model.Doctor, error) {
	return uc.repo.GetAll()
}

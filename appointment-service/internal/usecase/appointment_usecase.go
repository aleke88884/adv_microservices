package usecase

import (
	"appointment-service/internal/client"
	"appointment-service/internal/model"
	"appointment-service/internal/repository"
	"errors"
	"time"

	"github.com/google/uuid"
)

type AppointmentUseCase struct {
	repo         repository.AppointmentRepository
	doctorClient client.DoctorClient
}

func NewAppointmentUseCase(repo repository.AppointmentRepository, doctorClient client.DoctorClient) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:         repo,
		doctorClient: doctorClient,
	}
}

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

	err = uc.repo.Create(appointment)
	if err != nil {
		return nil, err
	}

	return appointment, nil
}

func (uc *AppointmentUseCase) GetAppointmentByID(id string) (*model.Appointment, error) {
	return uc.repo.GetByID(id)
}

func (uc *AppointmentUseCase) GetAllAppointments() ([]*model.Appointment, error) {
	return uc.repo.GetAll()
}

func (uc *AppointmentUseCase) UpdateAppointmentStatus(id string, newStatus model.Status) (*model.Appointment, error) {
	if !newStatus.IsValid() {
		return nil, errors.New("invalid status")
	}

	appointment, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := model.ValidateStatusTransition(appointment.Status, newStatus); err != nil {
		return nil, err
	}

	appointment.Status = newStatus
	appointment.UpdatedAt = time.Now()

	err = uc.repo.Update(appointment)
	if err != nil {
		return nil, err
	}

	return appointment, nil
}

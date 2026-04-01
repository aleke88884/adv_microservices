package repository

import (
	"appointment-service/internal/model"
	"errors"
	"sync"
)

type AppointmentRepository interface {
	Create(appointment *model.Appointment) error
	GetByID(id string) (*model.Appointment, error)
	GetAll() ([]*model.Appointment, error)
	Update(appointment *model.Appointment) error
}

type InMemoryAppointmentRepository struct {
	appointments map[string]*model.Appointment
	mu           sync.RWMutex
}

func NewInMemoryAppointmentRepository() *InMemoryAppointmentRepository {
	return &InMemoryAppointmentRepository{
		appointments: make(map[string]*model.Appointment),
	}
}

func (r *InMemoryAppointmentRepository) Create(appointment *model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.appointments[appointment.ID] = appointment
	return nil
}

func (r *InMemoryAppointmentRepository) GetByID(id string) (*model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	appointment, exists := r.appointments[id]
	if !exists {
		return nil, errors.New("appointment not found")
	}
	return appointment, nil
}

func (r *InMemoryAppointmentRepository) GetAll() ([]*model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	appointments := make([]*model.Appointment, 0, len(r.appointments))
	for _, appointment := range r.appointments {
		appointments = append(appointments, appointment)
	}
	return appointments, nil
}

func (r *InMemoryAppointmentRepository) Update(appointment *model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.appointments[appointment.ID]; !exists {
		return errors.New("appointment not found")
	}
	r.appointments[appointment.ID] = appointment
	return nil
}

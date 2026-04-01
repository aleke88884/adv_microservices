package repository

import (
	"doctor-service/internal/model"
	"errors"
	"sync"
)

type DoctorRepository interface {
	Create(doctor *model.Doctor) error
	GetByID(id string) (*model.Doctor, error)
	GetAll() ([]*model.Doctor, error)
	EmailExists(email string) (bool, error)
}

type InMemoryDoctorRepository struct {
	doctors map[string]*model.Doctor
	mu      sync.RWMutex
}

func NewInMemoryDoctorRepository() *InMemoryDoctorRepository {
	return &InMemoryDoctorRepository{
		doctors: make(map[string]*model.Doctor),
	}
}

func (r *InMemoryDoctorRepository) Create(doctor *model.Doctor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.doctors[doctor.ID] = doctor
	return nil
}

func (r *InMemoryDoctorRepository) GetByID(id string) (*model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doctor, exists := r.doctors[id]
	if !exists {
		return nil, errors.New("doctor not found")
	}
	return doctor, nil
}

func (r *InMemoryDoctorRepository) GetAll() ([]*model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doctors := make([]*model.Doctor, 0, len(r.doctors))
	for _, doctor := range r.doctors {
		doctors = append(doctors, doctor)
	}
	return doctors, nil
}

func (r *InMemoryDoctorRepository) EmailExists(email string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, doctor := range r.doctors {
		if doctor.Email == email {
			return true
		}
	}
	return false
}

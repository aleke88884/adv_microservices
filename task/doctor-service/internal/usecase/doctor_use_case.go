package usecase

import (
	"doctor-service/internal/model"
	"doctor-service/internal/repository"
	"errors"

	"github.com/google/uuid"
)

type DoctorUseCase struct {
	repo repository.DoctorRepository
}

func NewDoctorUseCase(repo repository.DoctorRepository) *DoctorUseCase {
	return &DoctorUseCase{repo: repo}
}

func (uc *DoctorUseCase) CreateDoctor(fullName, specialization, email string) (*model.Doctor, error) {
	if fullName == "" {
		return nil, errors.New("full name is required")
	}
	if specialization == "" {
		return nil, errors.New("specialization is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	doctor := &model.Doctor{
		ID:             uuid.New().String(),
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}

	err := uc.repo.Create(doctor)
	if err != nil {
		return nil, err
	}
	return doctor, nil
}

func (uc *DoctorUseCase) GetDoctorByID(id string) (*model.Doctor, error) {
	return uc.repo.GetByID(id)
}

func (uc *DoctorUseCase) GetAllDoctors() ([]*model.Doctor, error) {
	return uc.repo.GetAll()
}

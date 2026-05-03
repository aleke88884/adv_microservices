package repository

import (
	"database/sql"
	"doctor-service/internal/model"
	"errors"

	_ "github.com/lib/pq"
)

// PostgresDoctorRepository implements DoctorRepository using PostgreSQL.
type PostgresDoctorRepository struct {
	db *sql.DB
}

// NewPostgresDoctorRepository creates a new repository backed by the given *sql.DB.
func NewPostgresDoctorRepository(db *sql.DB) *PostgresDoctorRepository {
	return &PostgresDoctorRepository{db: db}
}

// Create inserts a doctor row. Returns an error if the email is already taken.
func (r *PostgresDoctorRepository) Create(doctor *model.Doctor) error {
	_, err := r.db.Exec(
		`INSERT INTO doctors (id, full_name, specialization, email) VALUES ($1, $2, $3, $4)`,
		doctor.ID, doctor.FullName, doctor.Specialization, doctor.Email,
	)
	return err
}

// GetByID fetches a single doctor by primary key.
func (r *PostgresDoctorRepository) GetByID(id string) (*model.Doctor, error) {
	row := r.db.QueryRow(
		`SELECT id, full_name, specialization, email FROM doctors WHERE id = $1`, id,
	)
	d := &model.Doctor{}
	if err := row.Scan(&d.ID, &d.FullName, &d.Specialization, &d.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("doctor not found")
		}
		return nil, err
	}
	return d, nil
}

// GetAll returns every doctor in the table.
func (r *PostgresDoctorRepository) GetAll() ([]*model.Doctor, error) {
	rows, err := r.db.Query(
		`SELECT id, full_name, specialization, email FROM doctors ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doctors []*model.Doctor
	for rows.Next() {
		d := &model.Doctor{}
		if err := rows.Scan(&d.ID, &d.FullName, &d.Specialization, &d.Email); err != nil {
			return nil, err
		}
		doctors = append(doctors, d)
	}
	return doctors, rows.Err()
}

// EmailExists checks whether the email is already registered.
func (r *PostgresDoctorRepository) EmailExists(email string) bool {
	var count int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM doctors WHERE email = $1`, email,
	).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

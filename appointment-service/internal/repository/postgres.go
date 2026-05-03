package repository

import (
	"appointment-service/internal/model"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq"
)

// PostgresAppointmentRepository implements AppointmentRepository using PostgreSQL.
type PostgresAppointmentRepository struct {
	db *sql.DB
}

// NewPostgresAppointmentRepository creates a repository backed by the given *sql.DB.
func NewPostgresAppointmentRepository(db *sql.DB) *PostgresAppointmentRepository {
	return &PostgresAppointmentRepository{db: db}
}

// Create inserts a new appointment row inside a transaction.
func (r *PostgresAppointmentRepository) Create(appointment *model.Appointment) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(
		`INSERT INTO appointments (id, title, description, doctor_id, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		appointment.ID,
		appointment.Title,
		appointment.Description,
		appointment.DoctorID,
		string(appointment.Status),
		appointment.CreatedAt,
		appointment.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID fetches a single appointment by primary key.
func (r *PostgresAppointmentRepository) GetByID(id string) (*model.Appointment, error) {
	row := r.db.QueryRow(
		`SELECT id, title, description, doctor_id, status, created_at, updated_at
		 FROM appointments WHERE id = $1`, id,
	)
	a := &model.Appointment{}
	var status string
	var createdAt, updatedAt time.Time
	if err := row.Scan(&a.ID, &a.Title, &a.Description, &a.DoctorID, &status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("appointment not found")
		}
		return nil, err
	}
	a.Status = model.Status(status)
	a.CreatedAt = createdAt
	a.UpdatedAt = updatedAt
	return a, nil
}

// GetAll returns every appointment ordered by creation time.
func (r *PostgresAppointmentRepository) GetAll() ([]*model.Appointment, error) {
	rows, err := r.db.Query(
		`SELECT id, title, description, doctor_id, status, created_at, updated_at
		 FROM appointments ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []*model.Appointment
	for rows.Next() {
		a := &model.Appointment{}
		var status string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.DoctorID, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		a.Status = model.Status(status)
		a.CreatedAt = createdAt
		a.UpdatedAt = updatedAt
		appointments = append(appointments, a)
	}
	return appointments, rows.Err()
}

// Update writes the new status and updated_at timestamp inside a transaction.
func (r *PostgresAppointmentRepository) Update(appointment *model.Appointment) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.Exec(
		`UPDATE appointments SET status = $1, updated_at = $2 WHERE id = $3`,
		string(appointment.Status), appointment.UpdatedAt, appointment.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		err = errors.New("appointment not found")
		return err
	}

	return tx.Commit()
}

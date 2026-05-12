package cache

import (
	"appointment-service/internal/model"
	"appointment-service/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CachedAppointmentRepository wraps an AppointmentRepository with a Redis caching layer.
// It implements repository.AppointmentRepository so the use case is unaware of caching.
//
// Cache strategies per operation (Section 5.2 of the assignment):
//   - GetByID  → Cache-Aside: key "appointment:<id>"
//   - GetAll   → Cache-Aside: key "appointments:list"
//   - Create   → Write-Around: write only to DB, then evict "appointments:list"
//   - Update   → Write-Through: update DB + cache "appointment:<id>", evict "appointments:list"
type CachedAppointmentRepository struct {
	db  repository.AppointmentRepository
	c   Cache
	ttl time.Duration
}

// NewCachedAppointmentRepository returns a caching wrapper around db.
func NewCachedAppointmentRepository(db repository.AppointmentRepository, c Cache, ttl time.Duration) *CachedAppointmentRepository {
	return &CachedAppointmentRepository{db: db, c: c, ttl: ttl}
}

// Create uses Write-Around: persists to DB only, then evicts the list cache.
// The individual appointment is NOT cached on creation (Write-Around).
func (r *CachedAppointmentRepository) Create(appt *model.Appointment) error {
	if err := r.db.Create(appt); err != nil {
		return err
	}
	ctx := context.Background()
	if err := r.c.Delete(ctx, "appointments:list"); err != nil {
		log.Printf("cache: WARNING: failed to invalidate appointments:list after create: %v", err)
	}
	return nil
}

// GetByID implements Cache-Aside for a single appointment.
func (r *CachedAppointmentRepository) GetByID(id string) (*model.Appointment, error) {
	ctx := context.Background()
	key := fmt.Sprintf("appointment:%s", id)

	data, err := r.c.Get(ctx, key)
	if err == nil {
		var a model.Appointment
		if jsonErr := json.Unmarshal(data, &a); jsonErr == nil {
			return &a, nil
		}
	}
	// Cache miss — fall through to database.
	appt, err := r.db.GetByID(id)
	if err != nil {
		return nil, err
	}
	if b, jsonErr := json.Marshal(appt); jsonErr == nil {
		if setErr := r.c.Set(ctx, key, b, r.ttl); setErr != nil {
			log.Printf("cache: WARNING: failed to set %s: %v", key, setErr)
		}
	}
	return appt, nil
}

// GetAll implements Cache-Aside for the full appointment list.
func (r *CachedAppointmentRepository) GetAll() ([]*model.Appointment, error) {
	ctx := context.Background()
	key := "appointments:list"

	data, err := r.c.Get(ctx, key)
	if err == nil {
		var appts []*model.Appointment
		if jsonErr := json.Unmarshal(data, &appts); jsonErr == nil {
			return appts, nil
		}
	}
	// Cache miss.
	appts, err := r.db.GetAll()
	if err != nil {
		return nil, err
	}
	if b, jsonErr := json.Marshal(appts); jsonErr == nil {
		if setErr := r.c.Set(ctx, key, b, r.ttl); setErr != nil {
			log.Printf("cache: WARNING: failed to set %s: %v", key, setErr)
		}
	}
	return appts, nil
}

// Update uses Write-Through: persist to DB, then update the individual cache
// entry and evict the list cache.
func (r *CachedAppointmentRepository) Update(appt *model.Appointment) error {
	if err := r.db.Update(appt); err != nil {
		return err
	}
	ctx := context.Background()
	key := fmt.Sprintf("appointment:%s", appt.ID)

	// Update individual entry in cache.
	if b, jsonErr := json.Marshal(appt); jsonErr == nil {
		if setErr := r.c.Set(ctx, key, b, r.ttl); setErr != nil {
			log.Printf("cache: WARNING: failed to update %s: %v", key, setErr)
		}
	}
	// Evict list cache.
	if err := r.c.Delete(ctx, "appointments:list"); err != nil {
		log.Printf("cache: WARNING: failed to invalidate appointments:list after update: %v", err)
	}
	return nil
}

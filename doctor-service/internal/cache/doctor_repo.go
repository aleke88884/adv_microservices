package cache

import (
	"context"
	"doctor-service/internal/model"
	"doctor-service/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CachedDoctorRepository wraps a DoctorRepository with a Redis caching layer.
// It implements repository.DoctorRepository so the use case is unaware of caching.
//
// Cache strategies per operation (Section 5.1 of the assignment):
//   - GetByID  → Cache-Aside (Read-Through): key "doctor:<id>"
//   - GetAll   → Cache-Aside: key "doctors:list"
//   - Create   → Write-Through: invalidate "doctors:list" after successful DB write
type CachedDoctorRepository struct {
	db  repository.DoctorRepository
	c   Cache
	ttl time.Duration
}

// NewCachedDoctorRepository returns a caching wrapper around db.
func NewCachedDoctorRepository(db repository.DoctorRepository, c Cache, ttl time.Duration) *CachedDoctorRepository {
	return &CachedDoctorRepository{db: db, c: c, ttl: ttl}
}

// Create persists the doctor to the database, then immediately evicts the list
// cache (Write-Through / invalidate-on-write).
// Cache write failures are logged but do not block the response.
func (r *CachedDoctorRepository) Create(doctor *model.Doctor) error {
	if err := r.db.Create(doctor); err != nil {
		return err
	}
	ctx := context.Background()
	if err := r.c.Delete(ctx, "doctors:list"); err != nil {
		log.Printf("cache: WARNING: failed to invalidate doctors:list after create: %v", err)
	}
	return nil
}

// GetByID implements Cache-Aside: checks cache first; on miss, fetches from DB
// and populates the cache. A cache miss never returns an error to the caller.
func (r *CachedDoctorRepository) GetByID(id string) (*model.Doctor, error) {
	ctx := context.Background()
	key := fmt.Sprintf("doctor:%s", id)

	data, err := r.c.Get(ctx, key)
	if err == nil {
		var d model.Doctor
		if jsonErr := json.Unmarshal(data, &d); jsonErr == nil {
			return &d, nil
		}
	}
	// Cache miss (or unmarshal error) — fall through to database.
	doc, err := r.db.GetByID(id)
	if err != nil {
		return nil, err
	}
	// Best-effort cache population.
	if b, jsonErr := json.Marshal(doc); jsonErr == nil {
		if setErr := r.c.Set(ctx, key, b, r.ttl); setErr != nil {
			log.Printf("cache: WARNING: failed to set %s: %v", key, setErr)
		}
	}
	return doc, nil
}

// GetAll implements Cache-Aside for the full list.
func (r *CachedDoctorRepository) GetAll() ([]*model.Doctor, error) {
	ctx := context.Background()
	key := "doctors:list"

	data, err := r.c.Get(ctx, key)
	if err == nil {
		var docs []*model.Doctor
		if jsonErr := json.Unmarshal(data, &docs); jsonErr == nil {
			return docs, nil
		}
	}
	// Cache miss — fall through.
	docs, err := r.db.GetAll()
	if err != nil {
		return nil, err
	}
	if b, jsonErr := json.Marshal(docs); jsonErr == nil {
		if setErr := r.c.Set(ctx, key, b, r.ttl); setErr != nil {
			log.Printf("cache: WARNING: failed to set %s: %v", key, setErr)
		}
	}
	return docs, nil
}

// EmailExists delegates directly to the database (no caching for uniqueness checks).
func (r *CachedDoctorRepository) EmailExists(email string) bool {
	return r.db.EmailExists(email)
}

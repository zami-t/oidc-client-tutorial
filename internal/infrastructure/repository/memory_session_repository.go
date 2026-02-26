package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

type sessionEntry struct {
	session   model.AppSession
	expiresAt time.Time
}

type memorySessionRepository struct {
	mu    sync.Mutex
	store map[string]sessionEntry
}

// NewMemorySessionRepository creates an in-memory SessionRepository.
func NewMemorySessionRepository() port.SessionRepository {
	return &memorySessionRepository{
		store: make(map[string]sessionEntry),
	}
}

func (r *memorySessionRepository) Save(_ context.Context, session model.AppSession, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[session.Id()] = sessionEntry{
		session:   session,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (r *memorySessionRepository) FindById(_ context.Context, id string) (model.AppSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.store[id]
	if !ok {
		return model.AppSession{}, model.NewAppError(
			model.ErrCodeSessionNotFound,
			"session not found",
			fmt.Errorf("session %q not found", id),
		)
	}
	if time.Now().After(entry.expiresAt) {
		delete(r.store, id)
		return model.AppSession{}, model.NewAppError(
			model.ErrCodeSessionNotFound,
			"session has expired",
			fmt.Errorf("session %q has expired", id),
		)
	}
	return entry.session, nil
}

func (r *memorySessionRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

type transactionEntry struct {
	tx        model.AuthorizationTransaction
	expiresAt time.Time
}

type memoryTransactionRepository struct {
	mu    sync.Mutex
	store map[string]transactionEntry
}

// NewMemoryTransactionRepository creates an in-memory TransactionRepository.
func NewMemoryTransactionRepository() port.TransactionRepository {
	return &memoryTransactionRepository{
		store: make(map[string]transactionEntry),
	}
}

func (r *memoryTransactionRepository) Save(_ context.Context, tx model.AuthorizationTransaction, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[tx.State()] = transactionEntry{
		tx:        tx,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (r *memoryTransactionRepository) FindByState(_ context.Context, state string) (model.AuthorizationTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.store[state]
	if !ok {
		return model.AuthorizationTransaction{}, model.NewAppError(
			model.ErrCodeStateMismatch,
			"state not found",
			fmt.Errorf("state %q not found in store", state),
		)
	}
	if time.Now().After(entry.expiresAt) {
		delete(r.store, state)
		return model.AuthorizationTransaction{}, model.NewAppError(
			model.ErrCodeStateMismatch,
			"state has expired",
			fmt.Errorf("state %q has expired", state),
		)
	}
	return entry.tx, nil
}

func (r *memoryTransactionRepository) Delete(_ context.Context, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, state)
	return nil
}

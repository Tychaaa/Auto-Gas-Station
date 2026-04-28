package repository

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type TransactionStore struct {
	mu      sync.RWMutex
	items   map[string]*model.Transaction
	counter uint64
}

func NewTransactionStore() *TransactionStore {
	return &TransactionStore{items: make(map[string]*model.Transaction)}
}

func (s *TransactionStore) Create(tx *model.Transaction) *model.Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	copyTx := *tx
	copyTx.ID = s.nextID()
	copyTx.CreatedAt = now
	copyTx.UpdatedAt = now

	s.items[copyTx.ID] = &copyTx
	return cloneTransaction(&copyTx)
}

func (s *TransactionStore) Get(id string) (*model.Transaction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.items[id]
	if !ok {
		return nil, false
	}
	return cloneTransaction(tx), true
}

func (s *TransactionStore) Update(id string, apply func(*model.Transaction) error) (*model.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.items[id]
	if !ok {
		return nil, ErrTransactionNotFound
	}

	next := *tx
	if err := apply(&next); err != nil {
		return nil, err
	}

	next.UpdatedAt = time.Now()
	s.items[id] = &next
	return cloneTransaction(&next), nil
}

func (s *TransactionStore) nextID() string {
	n := atomic.AddUint64(&s.counter, 1)
	return fmt.Sprintf("tx_%d_%06d", time.Now().UnixNano(), n)
}

func cloneTransaction(tx *model.Transaction) *model.Transaction {
	if tx == nil {
		return nil
	}
	copyTx := *tx
	return &copyTx
}

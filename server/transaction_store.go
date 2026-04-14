package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TransactionStore is an in-memory storage for transactions.
type TransactionStore struct {
	mu      sync.RWMutex
	items   map[string]*Transaction
	counter uint64
}

func NewTransactionStore() *TransactionStore {
	return &TransactionStore{
		items: make(map[string]*Transaction),
	}
}

func (s *TransactionStore) Create(tx *Transaction) *Transaction {
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

func (s *TransactionStore) Get(id string) (*Transaction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.items[id]
	if !ok {
		return nil, false
	}
	return cloneTransaction(tx), true
}

func (s *TransactionStore) nextID() string {
	n := atomic.AddUint64(&s.counter, 1)
	return fmt.Sprintf("tx_%d_%06d", time.Now().UnixNano(), n)
}

func cloneTransaction(tx *Transaction) *Transaction {
	if tx == nil {
		return nil
	}
	copyTx := *tx
	return &copyTx
}

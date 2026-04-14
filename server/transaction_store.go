package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TransactionStore хранит транзакции в памяти.
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

	// Фиксируем время создания и обновления.
	now := time.Now()
	// Делаем копию, чтобы не менять исходный объект.
	copyTx := *tx
	copyTx.ID = s.nextID()
	copyTx.CreatedAt = now
	copyTx.UpdatedAt = now

	// Сохраняем копию в хранилище.
	s.items[copyTx.ID] = &copyTx
	return cloneTransaction(&copyTx)
}

func (s *TransactionStore) Get(id string) (*Transaction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ищем транзакцию по ID.
	tx, ok := s.items[id]
	if !ok {
		return nil, false
	}
	// Возвращаем копию, чтобы внешние изменения не затронули хранилище.
	return cloneTransaction(tx), true
}

func (s *TransactionStore) nextID() string {
	// Увеличиваем счетчик атомарно, чтобы ID не повторялись.
	n := atomic.AddUint64(&s.counter, 1)
	return fmt.Sprintf("tx_%d_%06d", time.Now().UnixNano(), n)
}

func cloneTransaction(tx *Transaction) *Transaction {
	if tx == nil {
		return nil
	}
	// Возвращаем отдельную копию структуры.
	copyTx := *tx
	return &copyTx
}

package store

import (
	"sync"
)

type Store struct {
	data map[string]int64
	mu   sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]int64),
	}
}

func (s *Store) LastBlock() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if value, exists := s.data["last_block"]; exists {
		return uint64(value), nil
	}
	return 0, nil
}

func (s *Store) SetLastBlock(blockNum uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data["last_block"] = int64(blockNum)
	return nil
}

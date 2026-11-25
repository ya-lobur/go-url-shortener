package repository

import (
	"sync"
)

// MemoryStorage keeps URL mappings for the lifetime of the process.
type MemoryStorage struct {
	mu   sync.Mutex
	urls map[string]string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{urls: make(map[string]string)}
}

func (s *MemoryStorage) Save(id, originalURL string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.urls[id]; exists {
		return false, nil
	}
	s.urls[id] = originalURL
	return true, nil
}

func (s *MemoryStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	originalURL, ok := s.urls[id]
	return originalURL, ok
}

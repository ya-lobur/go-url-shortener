package repository

import (
	"strconv"
	"sync"
)

// MemoryStorage keeps URL mappings for the lifetime of the process.
type MemoryStorage struct {
	mu     sync.Mutex
	urls   map[string]string
	nextID uint64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{urls: make(map[string]string)}
}

func (s *MemoryStorage) Save(originalURL string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := strconv.FormatUint(s.nextID, 36)
	s.urls[id] = originalURL
	return id
}

func (s *MemoryStorage) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	originalURL, ok := s.urls[id]
	return originalURL, ok
}

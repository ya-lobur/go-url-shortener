package service

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var ErrInvalidURL = errors.New("invalid URL")

// Storage atomically saves a mapping, returning false if the ID is occupied.
type Storage interface {
	Save(id, originalURL string) (bool, error)
	Get(id string) (string, bool)
}

type Service struct {
	storage Storage
	mu      sync.Mutex
	nextID  uint64
}

func New(storage Storage) *Service { return &Service{storage: storage} }

func (s *Service) Shorten(originalURL string) (string, error) {
	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(originalURL, " \t\r\n") {
		return "", ErrInvalidURL
	}
	s.mu.Lock()
	s.nextID++
	id := strconv.FormatUint(s.nextID, 36)
	s.mu.Unlock()
	saved, err := s.storage.Save(id, originalURL)
	if err != nil {
		return "", fmt.Errorf("save URL: %w", err)
	}
	if !saved {
		return "", errors.New("URL identifier already exists")
	}
	return id, nil
}

func (s *Service) Resolve(id string) (string, bool) { return s.storage.Get(id) }

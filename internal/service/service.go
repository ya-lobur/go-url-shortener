package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

var ErrInvalidURL = errors.New("invalid URL")

// Storage atomically saves a mapping, returning false if the ID is occupied.
type Storage interface {
	Save(id, originalURL string) (bool, error)
	Get(id string) (string, bool)
}

const (
	idLength      = 8
	idAlphabet    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxIDAttempts = 10
)

type Service struct {
	storage    Storage
	generateID func() (string, error)
}

func New(storage Storage) *Service {
	return &Service{storage: storage, generateID: randomID}
}

func randomID() (string, error) {
	var id [idLength]byte
	limit := big.NewInt(int64(len(idAlphabet)))
	for i := range id {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		id[i] = idAlphabet[n.Int64()]
	}
	return string(id[:]), nil
}

func (s *Service) Shorten(originalURL string) (string, error) {
	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(originalURL, " \t\r\n") {
		return "", ErrInvalidURL
	}
	for attempt := 0; attempt < maxIDAttempts; attempt++ {
		id, err := s.generateID()
		if err != nil {
			return "", fmt.Errorf("generate URL identifier: %w", err)
		}
		saved, err := s.storage.Save(id, originalURL)
		if err != nil {
			return "", fmt.Errorf("save URL: %w", err)
		}
		if saved {
			return id, nil
		}
	}
	return "", errors.New("URL identifier generation attempts exhausted")
}

func (s *Service) Resolve(id string) (string, bool) { return s.storage.Get(id) }

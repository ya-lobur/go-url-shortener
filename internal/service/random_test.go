package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
)

func TestRandomIDFormat(t *testing.T) {
	s := New(repository.NewMemoryStorage())
	for i := 0; i < 100; i++ {
		id, err := s.Shorten("https://example.com/")
		require.NoError(t, err)
		require.Regexp(t, `^[0-9a-zA-Z]{8}$`, id)
	}
}

func TestCollisionRetry(t *testing.T) {
	storage := repository.NewMemoryStorage()
	saved, err := storage.Save("occupied", "https://first.example/")
	require.NoError(t, err)
	require.True(t, saved)
	s := New(storage)
	calls := 0
	s.generateID = func() (string, error) {
		calls++
		if calls == 1 {
			return "occupied", nil
		}
		return "available", nil
	}
	id, err := s.Shorten("https://second.example/")
	require.NoError(t, err)
	require.Equal(t, "available", id)
	require.Equal(t, 2, calls)
	original, found := s.Resolve("occupied")
	require.True(t, found)
	require.Equal(t, "https://first.example/", original)
	original, found = s.Resolve(id)
	require.True(t, found)
	require.Equal(t, "https://second.example/", original)
}

func TestCollisionLimit(t *testing.T) {
	storage := repository.NewMemoryStorage()
	_, err := storage.Save("occupied", "https://first.example/")
	require.NoError(t, err)
	s := New(storage)
	calls := 0
	s.generateID = func() (string, error) { calls++; return "occupied", nil }
	id, err := s.Shorten("https://second.example/")
	require.Error(t, err)
	require.Empty(t, id)
	require.Equal(t, 10, calls)
}

type countingStorage struct {
	calls int
	err   error
}

func (s *countingStorage) Save(string, string) (bool, error) { s.calls++; return false, s.err }
func (s *countingStorage) Get(string) (string, bool)         { return "", false }

func TestGeneratorFailure(t *testing.T) {
	storage := &countingStorage{}
	s := New(storage)
	failure := errors.New("entropy unavailable")
	calls := 0
	s.generateID = func() (string, error) { calls++; return "", failure }
	id, err := s.Shorten("https://example.com/")
	require.ErrorIs(t, err, failure)
	require.Empty(t, id)
	require.Equal(t, 1, calls)
	require.Zero(t, storage.calls)
}

func TestStorageFailureDoesNotRetry(t *testing.T) {
	failure := errors.New("write failed")
	storage := &countingStorage{err: failure}
	s := New(storage)
	calls := 0
	s.generateID = func() (string, error) { calls++; return "abcdefgh", nil }
	id, err := s.Shorten("https://example.com/")
	require.ErrorIs(t, err, failure)
	require.Empty(t, id)
	require.Equal(t, 1, calls)
	require.Equal(t, 1, storage.calls)
}

package service_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
	"github.com/ya-lobur/go-url-shortener/internal/service"
)

type failedStorage struct{ err error }

func (s failedStorage) Save(string, string) (bool, error) { return false, s.err }
func (s failedStorage) Get(string) (string, bool)         { return "", false }

func TestShortenAndResolve(t *testing.T) {
	s := service.New(repository.NewMemoryStorage())
	id, err := s.Shorten("https://example.com/")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	original, found := s.Resolve(id)
	require.True(t, found)
	require.Equal(t, "https://example.com/", original)
	_, found = s.Resolve("unknown")
	require.False(t, found)
}

func TestShortenInvalidURL(t *testing.T) {
	s := service.New(repository.NewMemoryStorage())
	for _, original := range []string{"", "not-a-url", "/relative", "ftp://example.com/", "https:///path", "https://example.com/a b", "https://example.com/\t", "https://example.com/\r", "https://example.com/\n"} {
		_, err := s.Shorten(original)
		require.ErrorIs(t, err, service.ErrInvalidURL)
	}
}

func TestShortenStorageError(t *testing.T) {
	failure := errors.New("write failed")
	_, err := service.New(failedStorage{failure}).Shorten("https://example.com/")
	require.ErrorIs(t, err, failure)
}

func TestConcurrentShorten(t *testing.T) {
	s := service.New(repository.NewMemoryStorage())
	var wg sync.WaitGroup
	ids := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := s.Shorten("https://example.com/")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]bool)
	for id := range ids {
		require.False(t, seen[id])
		seen[id] = true
	}
	require.Len(t, seen, 100)
}

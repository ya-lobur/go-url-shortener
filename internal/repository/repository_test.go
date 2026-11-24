package repository_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
)

func TestSaveDoesNotOverwrite(t *testing.T) {
	s := repository.NewMemoryStorage()
	saved, err := s.Save("id", "https://first.example/")
	require.NoError(t, err)
	require.True(t, saved)
	saved, err = s.Save("id", "https://second.example/")
	require.NoError(t, err)
	require.False(t, saved)
	original, found := s.Get("id")
	require.True(t, found)
	require.Equal(t, "https://first.example/", original)
}

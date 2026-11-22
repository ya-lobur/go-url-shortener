package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ya-lobur/go-url-shortener/internal/handler"
)

type storageStub struct {
	id       string
	original string
	found    bool
	saved    []string
	lookedUp []string
}

func (s *storageStub) Save(original string) string {
	s.saved = append(s.saved, original)
	return s.id
}

func (s *storageStub) Get(id string) (string, bool) {
	s.lookedUp = append(s.lookedUp, id)
	return s.original, s.found
}

func TestHandler_Shorten(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		contentType string
	}{
		{"http", "http://example.com/path", "text/plain"},
		{"https", "https://practicum.yandex.ru/", "text/plain"},
		{"query", "https://example.com/search?q=go&lang=en+us", "text/plain"},
		{"charset", "https://example.com/", "text/plain; charset=utf-8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &storageStub{id: "abc123"}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.original))
			request.Header.Set("Content-Type", tt.contentType)
			recorder := httptest.NewRecorder()

			handler.New(storage).ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, response.StatusCode)
			assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))
			assert.Equal(t, "http://localhost:8080/abc123", string(body))
			assert.Equal(t, []string{tt.original}, storage.saved)
			assert.Empty(t, storage.lookedUp)
		})
	}
}

func TestHandler_InvalidShorten(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
	}{
		{"missing_content_type", "https://example.com/", ""},
		{"wrong_content_type", "https://example.com/", "application/json"},
		{"malformed_content_type", "https://example.com/", "text/plain; charset"},
		{"empty_body", "", "text/plain"},
		{"invalid_url", "not-a-url", "text/plain"},
		{"relative_url", "/relative", "text/plain"},
		{"unsupported_scheme", "ftp://example.com/file", "text/plain"},
		{"missing_host", "https:///path", "text/plain"},
		{"space", "https://example.com/a b", "text/plain"},
		{"tab", "https://example.com/\t", "text/plain"},
		{"carriage_return", "https://example.com/\r", "text/plain"},
		{"newline", "https://example.com/\n", "text/plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &storageStub{}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			recorder := httptest.NewRecorder()

			handler.New(storage).ServeHTTP(recorder, request)

			assertBadRequest(t, recorder)
			assert.Empty(t, storage.saved)
			assert.Empty(t, storage.lookedUp)
		})
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) {
	return 0, errors.New("request body read failure")
}

func (failingBody) Close() error { return nil }

func TestHandler_BodyReadError(t *testing.T) {
	storage := &storageStub{}
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Content-Type", "text/plain")
	request.Body = failingBody{}
	recorder := httptest.NewRecorder()

	handler.New(storage).ServeHTTP(recorder, request)

	assertBadRequest(t, recorder)
	assert.Empty(t, storage.saved)
	assert.Empty(t, storage.lookedUp)
}

func TestHandler_Redirect(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		original string
	}{
		{"http", "/abc123", "http://example.com/path"},
		{"https_query", "/abc123", "https://example.com/search?q=go&lang=en+us"},
		{"query_in_request", "/abc123?preview=true", "https://example.com/"},
		{"escaped_id", "/%61bc123", "https://example.com/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &storageStub{original: tt.original, found: true}
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.New(storage).ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, http.StatusTemporaryRedirect, response.StatusCode)
			assert.Equal(t, tt.original, response.Header.Get("Location"))
			assert.Empty(t, body)
			assert.Equal(t, []string{"abc123"}, storage.lookedUp)
			assert.Empty(t, storage.saved)
		})
	}
}

func TestHandler_UnknownID(t *testing.T) {
	storage := &storageStub{}
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()

	handler.New(storage).ServeHTTP(recorder, request)

	assertBadRequest(t, recorder)
	assert.Equal(t, []string{"unknown"}, storage.lookedUp)
	assert.Empty(t, storage.saved)
}

func TestHandler_InvalidRoute(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/"},
		{http.MethodPost, "/abc123"},
		{http.MethodGet, "/a/b"},
		{http.MethodGet, "/abc123/"},
		{http.MethodGet, "/a%2Fb"},
		{http.MethodGet, "//abc123"},
		{http.MethodPost, "/a/b"},
		{http.MethodPut, "/"},
		{http.MethodDelete, "/abc123"},
		{http.MethodPatch, "/"},
		{http.MethodHead, "/abc123"},
		{http.MethodOptions, "/"},
		{http.MethodTrace, "/abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			storage := &storageStub{}
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader("https://example.com/"))
			request.Header.Set("Content-Type", "text/plain")
			recorder := httptest.NewRecorder()

			handler.New(storage).ServeHTTP(recorder, request)

			assertBadRequest(t, recorder)
			assert.Equal(t, "invalid method or path\n", recorder.Body.String())
			assert.Empty(t, recorder.Header().Get("Allow"))
			assert.Empty(t, storage.saved)
			assert.Empty(t, storage.lookedUp)
		})
	}
}

func assertBadRequest(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	response := recorder.Result()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", response.Header.Get("Content-Type"))
	assert.NotEmpty(t, body)
	assert.Empty(t, response.Header.Get("Location"))
}

package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/ya-lobur/go-url-shortener/internal/config"
	"github.com/ya-lobur/go-url-shortener/internal/handler"
	"github.com/ya-lobur/go-url-shortener/internal/middleware"
	"github.com/ya-lobur/go-url-shortener/internal/service"
)

func TestLoggingMiddleware(t *testing.T) {
	for _, tt := range []struct {
		name   string
		handle http.HandlerFunc
		status int
		body   string
	}{
		{"empty", func(w http.ResponseWriter, r *http.Request) {}, http.StatusOK, ""},
		{"implicit_ok_multiple_writes", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Go "))
			_, _ = w.Write([]byte("Привет"))
		}, http.StatusOK, "Go Привет"},
		{"explicit_status", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}, http.StatusCreated, "created"},
		{"repeated_status", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusInternalServerError)
		}, http.StatusAccepted, ""},
		{"status_after_body", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
			w.WriteHeader(http.StatusBadRequest)
		}, http.StatusOK, "ok"},
		{"error", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "invalid request", http.StatusBadRequest)
		}, http.StatusBadRequest, "invalid request\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/?check=1", nil)
			middleware.RequestLogger(zap.New(core))(tt.handle).ServeHTTP(recorder, request)

			require.Equal(t, tt.status, recorder.Code)
			require.Equal(t, tt.body, recorder.Body.String())
			require.Equal(t, 1, logs.Len())
			entry := logs.All()[0]
			require.Equal(t, zapcore.InfoLevel, entry.Level)
			fields := entry.ContextMap()
			assert.Equal(t, "/?check=1", fields["uri"])
			assert.Equal(t, http.MethodPost, fields["method"])
			assert.EqualValues(t, tt.status, fields["status"])
			assert.EqualValues(t, len(tt.body), fields["size"])
			duration, ok := fields["duration"].(time.Duration)
			require.True(t, ok)
			assert.GreaterOrEqual(t, duration, time.Duration(0))
		})
	}
}

type serviceStub struct {
	err      error
	id       string
	original string
	found    bool
	saved    []string
	lookedUp []string
}

func (s *serviceStub) Shorten(original string) (string, error) {
	s.saved = append(s.saved, original)
	return s.id, s.err
}

func (s *serviceStub) Resolve(id string) (string, bool) {
	s.lookedUp = append(s.lookedUp, id)
	return s.original, s.found
}

func TestHandler_Shorten(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		contentType string
		baseURL     string
		wantURL     string
	}{
		{"http", "http://example.com/path", "text/plain", "http://localhost:8080", "http://localhost:8080/abc123"},
		{"https", "https://practicum.yandex.ru/", "text/plain", "http://localhost:8080", "http://localhost:8080/abc123"},
		{"query", "https://example.com/search?q=go&lang=en+us", "text/plain", "http://localhost:8080", "http://localhost:8080/abc123"},
		{"charset", "https://example.com/", "text/plain; charset=utf-8", "http://localhost:8080", "http://localhost:8080/abc123"},
		{"trailing_slash", "https://example.com/", "text/plain", "https://short.example/", "https://short.example/abc123"},
		{"trailing_slashes_with_prefix", "https://example.com/", "text/plain", "https://short.example/links///", "https://short.example/links/abc123"},
		{"custom_base_url", "https://example.com/", "text/plain", "https://short.example:8443", "https://short.example:8443/abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &serviceStub{id: "abc123"}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.original))
			request.Header.Set("Content-Type", tt.contentType)
			recorder := httptest.NewRecorder()

			handler.New(storage, config.HandlerConfig{BaseURL: tt.baseURL}, zap.NewNop()).ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, response.StatusCode)
			assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))
			assert.Equal(t, tt.wantURL, string(body))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &serviceStub{}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			recorder := httptest.NewRecorder()

			handler.New(storage, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)

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
	storage := &serviceStub{}
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Content-Type", "text/plain")
	request.Body = failingBody{}
	recorder := httptest.NewRecorder()

	handler.New(storage, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)

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
			storage := &serviceStub{original: tt.original, found: true}
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.New(storage, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)

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
	storage := &serviceStub{}
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	recorder := httptest.NewRecorder()

	handler.New(storage, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)

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
			storage := &serviceStub{}
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader("https://example.com/"))
			request.Header.Set("Content-Type", "text/plain")
			recorder := httptest.NewRecorder()

			handler.New(storage, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)

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

func TestHandler_ServiceError(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"invalid_url", service.ErrInvalidURL, http.StatusBadRequest},
		{"storage_failure", errors.New("storage unavailable"), http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &serviceStub{err: tt.err}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/"))
			request.Header.Set("Content-Type", "text/plain")
			recorder := httptest.NewRecorder()
			handler.New(svc, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, request)
			require.Equal(t, tt.status, recorder.Code)
			require.Empty(t, recorder.Header().Get("Location"))
			require.NotContains(t, recorder.Body.String(), "storage unavailable")
		})
	}
}

func TestHandler_PercentID(t *testing.T) {
	for _, tt := range []struct{ path, id string }{
		{"/%25", "%"},
		{"/%2561bc123", "%61bc123"},
		{"/%2561%62c123", "%61bc123"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			svc := &serviceStub{original: "https://example.com/", found: true}
			recorder := httptest.NewRecorder()
			handler.New(svc, config.HandlerConfig{BaseURL: "http://localhost:8080"}, zap.NewNop()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
			require.Equal(t, []string{tt.id}, svc.lookedUp)
		})
	}
}

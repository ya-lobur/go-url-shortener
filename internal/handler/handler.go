package handler

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ya-lobur/go-url-shortener/internal/config"
	"github.com/ya-lobur/go-url-shortener/internal/service"
)

// Service provides URL shortening and resolution for the HTTP handlers.
type Service interface {
	Shorten(string) (string, error)
	Resolve(string) (string, bool)
}

type Handler struct {
	service Service
	baseURL string
	logger  *zap.Logger
}

func New(service Service, cfg config.HandlerConfig, logger *zap.Logger) http.Handler {
	h := &Handler{service: service, baseURL: strings.TrimRight(cfg.BaseURL, "/"), logger: logger}
	router := chi.NewRouter()
	router.Post("/", h.shorten)
	router.Get("/{id}", h.redirect)
	router.NotFound(invalidRoute)
	router.MethodNotAllowed(invalidRoute)
	return router
}

func invalidRoute(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "invalid method or path", http.StatusBadRequest)
}

func (h *Handler) shorten(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/plain" {
		http.Error(w, "expected text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read request body", http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	id, err := h.service.Shorten(originalURL)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			http.Error(w, "invalid URL", http.StatusBadRequest)
		} else {
			h.logger.Info("Failed to shorten URL", zap.Error(err))
			http.Error(w, "cannot shorten URL", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(w, h.baseURL+"/"+id); err != nil {
		h.logger.Info("Failed to write shortened URL", zap.Error(err))
	}
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// chi routes by RawPath when present; otherwise the parameter is already decoded.
	if r.URL.RawPath != "" {
		decoded, err := url.PathUnescape(id)
		if err != nil {
			invalidRoute(w, r)
			return
		}
		id = decoded
	}
	if strings.Contains(id, "/") {
		invalidRoute(w, r)
		return
	}
	originalURL, ok := h.service.Resolve(id)
	if !ok {
		http.Error(w, "unknown URL identifier", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

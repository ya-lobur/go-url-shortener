package handler

import (
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
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
}

func New(service Service, cfg config.HandlerConfig) http.Handler {
	h := &Handler{service: service, baseURL: cfg.BaseURL}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Preserve routing by the decoded path used before switching to chi.
			chi.RouteContext(r.Context()).RoutePath = r.URL.Path
			next.ServeHTTP(w, r)
		})
	})
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
			log.Printf("shorten URL: %v", err)
			http.Error(w, "cannot shorten URL", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(w, h.baseURL+"/"+id); err != nil {
		log.Printf("write shortened URL: %v", err)
	}
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	originalURL, ok := h.service.Resolve(chi.URLParam(r, "id"))
	if !ok {
		http.Error(w, "unknown URL identifier", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

package handler

import (
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Storage provides the URL mappings used by the HTTP handlers.
type Storage interface {
	Save(originalURL string) string
	Get(id string) (string, bool)
}

type Handler struct {
	storage Storage
}

func New(storage Storage) http.Handler {
	h := &Handler{storage: storage}
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
	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(originalURL, " \t\r\n") {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	id := h.storage.Save(originalURL)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(w, "http://localhost:8080/"+id); err != nil {
		log.Printf("write shortened URL: %v", err)
	}
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	originalURL, ok := h.storage.Get(chi.URLParam(r, "id"))
	if !ok {
		http.Error(w, "unknown URL identifier", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

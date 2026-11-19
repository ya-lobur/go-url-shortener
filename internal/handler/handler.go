package handler

import (
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/ya-lobur/go-url-shortener/internal/repository"
)

type Handler struct {
	storage *repository.MemoryStorage
}

func New(storage *repository.MemoryStorage) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/":
		h.shorten(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/") && len(r.URL.Path) > 1 && !strings.Contains(r.URL.Path[1:], "/"):
		h.redirect(w, r)
	default:
		http.Error(w, "invalid method or path", http.StatusBadRequest)
	}
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
	originalURL, ok := h.storage.Get(strings.TrimPrefix(r.URL.Path, "/"))
	if !ok {
		http.Error(w, "unknown URL identifier", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

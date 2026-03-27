package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/tradekmv/shortener.git/internal/repository/storage"
)

// ShortenerHandler обрабатывает HTTP-запросы для сокращения URL
type ShortenerHandler struct {
	storage storage.Storage // используем интерфейс вместо конкретного типа
	baseURL string
}

func New(storage storage.Storage, baseURL string) *ShortenerHandler {
	return &ShortenerHandler{
		storage: storage,
		baseURL: baseURL,
	}
}

// PostHandler обрабатывает POST запросы для создания короткой ссылки
func (h *ShortenerHandler) PostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(r.Body)

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	shortID := h.storage.Save(originalURL)

	shortURL := fmt.Sprintf("%s/%s", h.baseURL, shortID)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortURL))
	if err != nil {
		return
	}
}

// GetHandler обрабатывает GET запросы для редиректа по короткому ID
func (h *ShortenerHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := r.PathValue("id")
	if shortID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	originalURL, exists := h.storage.Get(shortID)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

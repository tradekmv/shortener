package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tradekmv/shortener.git/internal/db"
	"github.com/tradekmv/shortener.git/internal/middleware"
	"github.com/tradekmv/shortener.git/internal/service"
)

type ShortenerRequest struct {
	URL string `json:"url"`
}

type ShortenerResponse struct {
	Result string `json:"result"`
}

// ShortenerHandler обрабатывает HTTP-запросы для сокращения URL
type ShortenerHandler struct {
	service *service.Service
	baseURL string
	db      db.Pinger
}

func New(service *service.Service, baseURL string, db db.Pinger) *ShortenerHandler {
	return &ShortenerHandler{
		service: service,
		baseURL: baseURL,
		db:      db,
	}
}

// PingHandler проверяет соединение с базой данных
func (h *ShortenerHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.db.Ping(); err != nil {
		middleware.Log.Printf("Ошибка проверки соединения с БД: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PostHandler обрабатывает POST запросы для создания короткой ссылки
func (h *ShortenerHandler) PostHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело запроса", http.StatusBadRequest)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			middleware.Log.Printf("Ошибка закрытия тела запроса: %v", err)
		}
	}(r.Body)

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "URL не может быть пустым", http.StatusBadRequest)
		return
	}

	shortID, err := h.service.Save(r.Context(), originalURL)
	if err != nil {
		middleware.Log.Printf("Ошибка сохранения URL: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	shortURL, err := url.JoinPath(h.baseURL, shortID)
	if err != nil {
		http.Error(w, "Не удалось построить короткий URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(shortURL))
	if err != nil {
		return
	}
}

// GetHandler обрабатывает GET запросы для редиректа по короткому ID
func (h *ShortenerHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.PathValue("id")
	if shortID == "" {
		http.Error(w, "ID обязателен", http.StatusBadRequest)
		return
	}

	originalURL, ok := h.service.Get(shortID)
	if !ok {
		http.Error(w, "URL не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// APIShortenHandler обрабатывает POST запросы JSON API для создания короткой ссылки
func (h *ShortenerHandler) APIShortenHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenerRequest

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL не может быть пустым", http.StatusBadRequest)
		return
	}

	shortID, err := h.service.Save(r.Context(), req.URL)
	if err != nil {
		middleware.Log.Printf("Ошибка сохранения URL: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	shortURL, err := url.JoinPath(h.baseURL, shortID)
	if err != nil {
		http.Error(w, "Не удалось построить короткий URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := ShortenerResponse{Result: shortURL}
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		return
	}
}

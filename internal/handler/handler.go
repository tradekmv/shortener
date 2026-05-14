package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog"
	"github.com/tradekmv/shortener.git/internal/auth"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
)

type ShortenerRequest struct {
	URL string `json:"url"`
}

type ShortenerResponse struct {
	Result string `json:"result"`
}

// BatchRequestItem represents a single URL in batch request
type BatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponseItem represents a single result in batch response
type BatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// ShortenerHandler обрабатывает HTTP-запросы для сокращения URL
type ShortenerHandler struct {
	service *service.Service
	baseURL string
	log     *zerolog.Logger
}

func New(service *service.Service, baseURL string, store storage.Storage, log *zerolog.Logger) *ShortenerHandler {
	return &ShortenerHandler{
		service: service,
		baseURL: baseURL,
		log:     log,
	}
}

// PingHandler проверяет соединение с хранилищем
func (h *ShortenerHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(); err != nil {
		h.log.Printf("Ошибка проверки соединения: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// PostHandler обрабатывает POST запросы для создания короткой ссылки
func (h *ShortenerHandler) PostHandler(w http.ResponseWriter, r *http.Request) {
	// Создаём или получаем userID из cookie
	userID, err := auth.CreateUserIDIfNeeded(w, r)
	if err != nil {
		h.log.Printf("Ошибка создания userID: %v", err)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело запроса", http.StatusBadRequest)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			h.log.Printf("Ошибка закрытия тела запроса: %v", err)
		}
	}(r.Body)

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "URL не может быть пустым", http.StatusBadRequest)
		return
	}

	shortID, err := h.service.SaveWithUserID(r.Context(), originalURL, userID)
	if err != nil {
		if errors.Is(err, service.ErrURLAlreadyExists) {
			// URL уже существует — возвращаем 409 Conflict с коротким URL
			shortURL, _ := url.JoinPath(h.baseURL, shortID)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
		h.log.Printf("Ошибка сохранения URL: %v", err)
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

	originalURL, err := h.service.Get(r.Context(), shortID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "URL не найден", http.StatusNotFound)
			return
		}
		h.log.Printf("Ошибка получения URL: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// APIShortenHandler обрабатывает POST запросы JSON API для создания короткой ссылки
func (h *ShortenerHandler) APIShortenHandler(w http.ResponseWriter, r *http.Request) {
	// Создаём или получаем userID из cookie
	userID, err := auth.CreateUserIDIfNeeded(w, r)
	if err != nil {
		h.log.Printf("Ошибка создания userID: %v", err)
		// Продолжаем без userID, кука уже должна быть установлена при CreateUserIDIfNeeded
		// но если была ошибка, используем пустой userID
		userID = ""
	}

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

	shortID, err := h.service.SaveWithUserID(r.Context(), req.URL, userID)
	if err != nil {
		if errors.Is(err, service.ErrURLAlreadyExists) {
			// URL уже существует — возвращаем 409 Conflict с коротким URL в JSON формате
			shortURL, _ := url.JoinPath(h.baseURL, shortID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ShortenerResponse{Result: shortURL})
			return
		}
		h.log.Printf("Ошибка сохранения URL: %v", err)
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

// APIBatchShortenHandler handles POST /api/shorten/batch requests for batch URL shortening
func (h *ShortenerHandler) APIBatchShortenHandler(w http.ResponseWriter, r *http.Request) {
	var req []BatchRequestItem

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "Пустой батч", http.StatusBadRequest)
		return
	}

	// Convert request to URLRecords for service
	records := make([]storage.URLRecord, 0, len(req))
	for _, item := range req {
		if item.OriginalURL == "" {
			http.Error(w, "URL не может быть пустым", http.StatusBadRequest)
			return
		}
		records = append(records, storage.URLRecord{
			OriginalURL: item.OriginalURL,
		})
	}

	// Generate short IDs
	results, err := h.service.SaveBatch(r.Context(), records)
	if err != nil {
		h.log.Printf("Ошибка batch сохранения: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Build response
	response := make([]BatchResponseItem, 0, len(results))
	for i, rec := range results {
		shortURL, err := url.JoinPath(h.baseURL, rec.ShortURL)
		if err != nil {
			h.log.Printf("Ошибка построения URL: %v", err)
			http.Error(w, "Не удалось построить короткий URL", http.StatusInternalServerError)
			return
		}
		response = append(response, BatchResponseItem{
			CorrelationID: req[i].CorrelationID,
			ShortURL:      shortURL,
		})
	}

	// Устанавливаем Content-Type до записи
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(response); err != nil {
		h.log.Printf("Ошибка кодирования JSON: %v", err)
		return
	}
}

// GetUserURLsHandler обрабатывает GET /api/user/urls запросы
func (h *ShortenerHandler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	// Пытаемся получить userID из существующей куки или создаём новую
	userID, err := auth.CreateUserIDIfNeeded(w, r)
	if err != nil {
		h.log.Printf("Ошибка создания userID: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if userID == "" {
		// Кука содержит пустой user_id
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем URLs пользователя
	urls, err := h.service.GetUserURLs(r.Context(), userID)
	if err != nil {
		h.log.Printf("Ошибка получения URLs пользователя: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		// Нет URLs для пользователя
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Формируем ответ
	response := make([]UserURLResponse, 0, len(urls))
	for _, rec := range urls {
		shortURL, err := url.JoinPath(h.baseURL, rec.ShortURL)
		if err != nil {
			h.log.Printf("Ошибка построения URL: %v", err)
			http.Error(w, "Не удалось построить короткий URL", http.StatusInternalServerError)
			return
		}
		response = append(response, UserURLResponse{
			ShortURL:    shortURL,
			OriginalURL: rec.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	if err := enc.Encode(response); err != nil {
		h.log.Printf("Ошибка кодирования JSON: %v", err)
		return
	}
}

// UserURLResponse структура ответа для /api/user/urls
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

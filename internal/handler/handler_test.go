package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/tradekmv/shortener.git/internal/repository/mock"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
	"go.uber.org/mock/gomock"
)

var testLogger = zerolog.New(os.Stdout).With().Timestamp().Logger()

var _ storage.Storage = (*mock_storage.MockStorage)(nil)

func TestPostHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Save(gomock.Any(), gomock.Any(), "https://example.com").Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "http://localhost:8080/") {
		t.Errorf("ожидался URL с префиксом 'http://localhost:8080/', получен '%s'", body)
	}
}

func TestPostHandler_EmptyBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get(gomock.Any(), "abc123").Return("https://example.com", nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("ожидался статус %d, получен %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("ожидалось Location 'https://example.com', получено '%s'", location)
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get(gomock.Any(), "nonexistent").Return("", service.ErrNotFound)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ожидался статус %d, получен %d", http.StatusNotFound, w.Code)
	}
}

func TestAPIShortenHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Save(gomock.Any(), gomock.Any(), "https://example.com").Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIShortenHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("ожидался Content-Type 'application/json', получен '%s'", contentType)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, `"result":"`) {
		t.Errorf("ожидался ответ с полем 'result', получен '%s'", respBody)
	}
}

func TestAPIShortenHandler_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIShortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIShortenHandler_EmptyURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `{"url":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIShortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

func TestPingHandler_NoDatabase(t *testing.T) {
	svc := service.NewService(nil)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.PingHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d при отсутствии хранилища, получен %d", http.StatusInternalServerError, w.Code)
	}
}

func TestPingHandler_WithMockStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping().Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", mockStorage, &testLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.PingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
	}
}

func TestPingHandler_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping().Return(errors.New("connection refused"))

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", mockStorage, &testLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.PingHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d при ошибке хранилища, получен %d", http.StatusInternalServerError, w.Code)
	}
}

func TestAPIBatchShortenHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	// Mock SaveBatch for batch save operation
	mockStorage.EXPECT().SaveBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, urls []storage.URLRecord) ([]storage.URLRecord, error) {
			result := make([]storage.URLRecord, len(urls))
			for i, url := range urls {
				result[i] = storage.URLRecord{
					ShortURL:    "testID",
					OriginalURL: url.OriginalURL,
				}
			}
			return result, nil
		},
	).Times(1)

	body := `[
		{"correlation_id": "abc123", "original_url": "https://example.com/1"},
		{"correlation_id": "def456", "original_url": "https://example.com/2"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIBatchShortenHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("ожидался Content-Type 'application/json', получен '%s'", contentType)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, `"short_url":"`) {
		t.Errorf("ожидался ответ с полем 'short_url', получен '%s'", respBody)
	}
	if !strings.Contains(respBody, `"correlation_id":"`) {
		t.Errorf("ожидался ответ с полем 'correlation_id', получен '%s'", respBody)
	}
}

func TestAPIBatchShortenHandler_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIBatchShortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIBatchShortenHandler_EmptyBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `[]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIBatchShortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIBatchShortenHandler_EmptyURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock_storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil, &testLogger)

	body := `[{"correlation_id": "abc123", "original_url": ""}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.APIBatchShortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
	}
}

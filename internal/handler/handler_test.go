package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tradekmv/shortener.git/internal/repository/mock"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
	"go.uber.org/mock/gomock"
)

// Compile-time проверка
var _ storage.Storage = (*mock.MockStorage)(nil)

func TestPostHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Save(gomock.Any(), "https://example.com").Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get("abc123").Return("https://example.com", true)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get("nonexistent").Return("", false)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Save(gomock.Any(), "https://example.com").Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", nil)

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
	h := New(svc, "http://localhost:8080", nil)

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

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping().Return(nil)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", mockStorage)

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

	mockStorage := mock.NewMockStorage(ctrl)
	mockStorage.EXPECT().Ping().Return(errors.New("connection refused"))

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080", mockStorage)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.PingHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ожидался статус %d при ошибке хранилища, получен %d", http.StatusInternalServerError, w.Code)
	}
}

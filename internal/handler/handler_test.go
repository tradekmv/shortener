package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
	"go.uber.org/mock/gomock"
)

func TestPostHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Exists(gomock.Any()).Return(false)
	mockStorage.EXPECT().Save(gomock.Any(), "https://example.com")

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080")

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

	mockStorage := storage.NewMockStorage(ctrl)
	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080")

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

	mockStorage := storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get("abc123").Return("https://example.com", true)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080")

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

	mockStorage := storage.NewMockStorage(ctrl)
	mockStorage.EXPECT().Get("nonexistent").Return("", false)

	svc := service.NewService(mockStorage)
	h := New(svc, "http://localhost:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ожидался статус %d, получен %d", http.StatusNotFound, w.Code)
	}
}

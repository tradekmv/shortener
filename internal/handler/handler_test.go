package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockStorage struct {
	urls map[string]string
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		urls: make(map[string]string),
	}
}

func (m *MockStorage) Save(originalURL string) string {
	shortID := "test123"
	m.urls[shortID] = originalURL
	return shortID
}

func (m *MockStorage) Get(shortID string) (string, bool) {
	url, exists := m.urls[shortID]
	return url, exists
}

func TestPostHandler_Success(t *testing.T) {
	mock := NewMockStorage()
	h := New(mock, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "http://localhost:8080/") {
		t.Errorf("expected URL to contain base URL, got %s", body)
	}
}

func TestPostHandler_EmptyBody(t *testing.T) {
	mock := NewMockStorage()
	h := New(mock, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostHandler_WrongMethod(t *testing.T) {
	mock := NewMockStorage()
	h := New(mock, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.PostHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestGetHandler_Success(t *testing.T) {
	mock := NewMockStorage()
	mock.urls["abc123"] = "https://example.com"
	h := New(mock, "http://localhost:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected Location 'https://example.com', got '%s'", location)
	}
}

func TestGetHandler_NotFound(t *testing.T) {
	mock := NewMockStorage()
	h := New(mock, "http://localhost:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetHandler_WrongMethod(t *testing.T) {
	mock := NewMockStorage()
	h := New(mock, "http://localhost:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodPost, "/abc123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

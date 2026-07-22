package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
)

func init() {
	// AUTH_SECRET_KEY нужен для работы handler'ов, устанавливающих куки.
	if len(os.Getenv("AUTH_SECRET_KEY")) < 32 {
		os.Setenv("AUTH_SECRET_KEY", "this-is-a-very-long-test-secret-key-1234567890")
	}
}

func newExampleHandler() *handler.ShortenerHandler {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	store := storage.NewMemory()
	svc := service.NewService(store)
	return handler.New(svc, "http://localhost:8080", store, &log, nil)
}

// ExampleShortenerHandler_PostHandler демонстрирует сокращение URL через POST /.
func ExampleShortenerHandler_PostHandler() {
	h := newExampleHandler()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/page"))
	w := httptest.NewRecorder()
	h.PostHandler(w, req)

	fmt.Println(w.Code)
	fmt.Println(strings.HasPrefix(w.Body.String(), "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

// ExampleShortenerHandler_GetHandler демонстрирует разрешение короткой ссылки.
func ExampleShortenerHandler_GetHandler() {
	store := storage.NewMemory()
	if err := store.SaveWithUserID(context.Background(), "abc12345", "https://example.com/page", ""); err != nil {
		fmt.Println("setup error:", err)
		return
	}

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	svc := service.NewService(store)
	h := handler.New(svc, "http://localhost:8080", store, &log, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.GetHandler)

	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Location"))
	// Output:
	// 307
	// https://example.com/page
}

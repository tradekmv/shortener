package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/tradekmv/shortener.git/internal/config"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/middleware"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Ошибка парсинга флагов: %v", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	store := storage.New()
	svc := service.NewService(store)
	h := handler.New(svc, cfg.BaseURL)

	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Get("/{id}", h.GetHandler)

	addr := cfg.ServerAddress
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Printf("Ошибка сервера: %v", err)
	}
}

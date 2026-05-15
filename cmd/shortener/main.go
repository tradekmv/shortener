package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/tradekmv/shortener.git/internal/config"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/middleware"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
	"github.com/tradekmv/shortener.git/pkg/logger"
	"github.com/tradekmv/shortener.git/pkg/registry"
)

func main() {
	log := logger.New()

	// Registry для управления ресурсами
	reg := registry.New()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Ошибка парсинга флагов: %v", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return middleware.LoggingMiddleware(next, log)
	})
	r.Use(middleware.GzipMiddleware)

	// Выбираем хранилище в зависимости от конфигурации
	store := initStorage(cfg, log)
	reg.Register(store)

	svc := service.NewService(store)
	h := handler.New(svc, cfg.BaseURL, store, log)

	r.Get("/ping", h.PingHandler)
	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIBatchShortenHandler)
	r.Get("/api/user/urls", h.GetUserURLsHandler)
	r.Delete("/api/user/urls", h.DeleteUserURLsHandler)
	r.Get("/{id}", h.GetHandler)

	addr := cfg.ServerAddress

	// Graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Запуск сервера в горутине
	go func() {
		log.Printf("Сервер запущен на %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка сервера: %v", err)
			os.Exit(1)
		}
	}()

	// Ожидание сигнала для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы сервера...")

	// Закрываем все ресурсы через registry
	if err := reg.CloseAll(); err != nil {
		log.Printf("Ошибка закрытия ресурсов: %v", err)
	}

	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Printf("Ошибка при завершении: %v", err)
	}
	log.Println("Сервер остановлен")
}

// initStorage выбирает хранилище с учётом конфигурации и fallback-логикой
func initStorage(cfg *config.Config, log *logger.Logger) storage.Storage {
	// 1. PostgreSQL
	if cfg.DatabaseDSN != "" {
		store, err := storage.NewPostgres(cfg.DatabaseDSN)
		if err == nil {
			log.Println("Используем PostgreSQL хранилище")
			return store
		}
		log.Printf("Ошибка подключения к PostgreSQL: %v", err)
	}

	// 2. Файловое хранилище
	if cfg.FileStoragePath != "" {
		store, err := storage.New(cfg.FileStoragePath)
		if err == nil {
			log.Println("Используем файловый storage")
			return store
		}
		log.Printf("Ошибка инициализации файлового хранилища: %v", err)
	}

	// 3. Память (fallback)
	log.Println("Используем хранилище в памяти")
	return storage.NewMemory()
}

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/tradekmv/shortener.git/internal/audit"
	"github.com/tradekmv/shortener.git/internal/config"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/middleware"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
	"github.com/tradekmv/shortener.git/pkg/logger"
	"github.com/tradekmv/shortener.git/pkg/registry"
)

// main — точка входа в приложение.
// Выполняет graceful shutdown всех ресурсов перед завершением.
func main() {
	log := logger.New()

	reg := registry.New()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Ошибка парсинга флагов: %v", err)
		reg.CloseAll()
		return
	}

	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return middleware.LoggingMiddleware(next, log)
	})
	r.Use(middleware.GzipMiddleware)

	store := initStorage(cfg, log)
	reg.Register(store)

	auditPub := initAudit(cfg, log)
	if auditPub != nil {
		reg.Register(auditPub)
	}

	svc := service.NewService(store)
	h := handler.New(svc, cfg.BaseURL, store, log, auditPub)

	r.Get("/ping", h.PingHandler)
	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Post("/api/shorten/batch", h.APIBatchShortenHandler)
	r.Get("/api/user/urls", h.GetUserURLsHandler)
	r.Delete("/api/user/urls", h.DeleteUserURLsHandler)
	r.Get("/{id}", h.GetHandler)

	addr := cfg.ServerAddress

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("Сервер запущен на %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Получен сигнал завершения")
	case err := <-serverErr:
		log.Printf("Ошибка сервера: %v", err)
	}

	log.Println("Завершение работы сервера...")

	h.Close()

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
	if cfg.DatabaseDSN != "" {
		store, err := storage.NewPostgres(cfg.DatabaseDSN)
		if err == nil {
			log.Println("Используем PostgreSQL хранилище")
			return store
		}
		log.Printf("Ошибка подключения к PostgreSQL: %v", err)
	}

	if cfg.FileStoragePath != "" {
		store, err := storage.New(cfg.FileStoragePath)
		if err == nil {
			log.Println("Используем файловый storage")
			return store
		}
		log.Printf("Ошибка инициализации файлового хранилища: %v", err)
	}

	log.Println("Используем хранилище в памяти")
	return storage.NewMemory()
}

// initAudit инициализирует систему аудита с наблюдателями (паттерн Observer)
func initAudit(cfg *config.Config, log *logger.Logger) *audit.Publisher {
	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		log.Println("Аудит отключён")
		return nil
	}

	publisher := audit.NewPublisher(log)

	if cfg.AuditFile != "" {
		fileObs, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			log.Printf("Ошибка создания файлового наблюдателя аудита: %v", err)
		} else {
			publisher.Subscribe(fileObs)
			log.Printf("Аудит в файл включён: %s", cfg.AuditFile)
		}
	}

	if cfg.AuditURL != "" {
		remoteObs := audit.NewRemoteObserver(cfg.AuditURL)
		publisher.Subscribe(remoteObs)
		log.Printf("Аудит на удалённый сервер включён: %s", cfg.AuditURL)
	}

	return publisher
}

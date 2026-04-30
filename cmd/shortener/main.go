package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/tradekmv/shortener.git/internal/config"
	"github.com/tradekmv/shortener.git/internal/db"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/middleware"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		middleware.Log.Printf("Ошибка парсинга флагов: %v", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	// Выбираем хранилище в зависимости от конфигурации
	var store storage.Storage
	var dbPinger db.Pinger

	// 1. Пытаемся использовать PostgreSQL
	if cfg.DatabaseDSN != "" {
		postgresStore, err := storage.NewPostgres(cfg.DatabaseDSN)
		if err != nil {
			middleware.Log.Printf("Ошибка подключения к PostgreSQL: %v", err)
			// Fallback к файловому хранилищу
			if cfg.FileStoragePath != "" {
				middleware.Log.Println("Используем файловый fallback")
				store, err = storage.New(cfg.FileStoragePath)
				if err != nil {
					middleware.Log.Printf("Ошибка инициализации файлового хранилища: %v", err)
					os.Exit(1)
				}
			} else {
				// Fallback к памяти
				middleware.Log.Println("Используем хранилище в памяти")
				store = storage.NewMemory()
			}
		} else {
			store = postgresStore
			dbPinger = postgresStore
			middleware.Log.Println("Используем PostgreSQL хранилище")
		}
	} else {
		// 2. Пытаемся использовать файловое хранилище
		if cfg.FileStoragePath != "" {
			store, err = storage.New(cfg.FileStoragePath)
			if err != nil {
				middleware.Log.Printf("Ошибка инициализации файлового хранилища: %v", err)
				// Fallback к памяти
				middleware.Log.Println("Используем хранилище в памяти")
				store = storage.NewMemory()
			} else {
				middleware.Log.Println("Используем файловый storage")
			}
		} else {
			// 3. Используем память
			store = storage.NewMemory()
			middleware.Log.Println("Используем хранилище в памяти")
		}
	}

	svc := service.NewService(store)
	h := handler.New(svc, cfg.BaseURL, dbPinger)

	r.Get("/ping", h.PingHandler)
	r.Post("/", h.PostHandler)
	r.Post("/api/shorten", h.APIShortenHandler)
	r.Get("/{id}", h.GetHandler)

	addr := cfg.ServerAddress

	// Graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Запуск сервера в горутине
	go func() {
		middleware.Log.Printf("Сервер запущен на %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			middleware.Log.Printf("Ошибка сервера: %v", err)
			os.Exit(1)
		}
	}()

	// Ожидание сигнала для завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	middleware.Log.Println("Завершение работы сервера...")

	// Закрываем хранилище
	if store != nil {
		if err := store.Close(); err != nil {
			middleware.Log.Printf("Ошибка закрытия хранилища: %v", err)
		}
	}

	if err := srv.Shutdown(context.TODO()); err != nil {
		middleware.Log.Printf("Ошибка при завершении: %v", err)
	}
	middleware.Log.Println("Сервер остановлен")
}

// Compile-time проверка, что db.Database реализует db.Pinger
var _ db.Pinger = (*db.Database)(nil)

package main

import (
	"log"
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
		log.Printf("Ошибка парсинга флагов: %v", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	store, err := storage.New(cfg.FileStoragePath)
	if err != nil {
		log.Printf("Ошибка инициализации хранилища: %v", err)
		os.Exit(1)
	}

	svc := service.NewService(store)

	// Инициализация подключения к БД
	var database *db.Database
	if cfg.DatabaseDSN != "" {
		database, err = db.New(cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Ошибка подключения к БД: %v", err)
			os.Exit(1)
		}
		defer database.Close()

		// Инициализация схемы БД
		if err := database.InitSchema(); err != nil {
			log.Printf("Ошибка инициализации схемы БД: %v", err)
			os.Exit(1)
		}
	}

	h := handler.New(svc, cfg.BaseURL, database)

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
	if err := srv.Shutdown(nil); err != nil {
		log.Printf("Ошибка при завершении: %v", err)
	}
	log.Println("Сервер остановлен")
}

package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/tradekmv/shortener.git/internal/repository/storage"
)

const (
	charset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length      = 8
	maxAttempts = 10
)

var (
	ErrMaxRetriesExceeded = errors.New("не удалось сгенерировать уникальный ID после максимального количества попыток")
	ErrURLAlreadyExists   = errors.New("URL уже существует")
	ErrAlreadyExists      = errors.New("короткий ID уже существует")
	ErrNotFound           = errors.New("короткая ссылка не найдена")
)

// PostgresStorageGetter интерфейс для хранилищ с поддержкой поиска по original_url
type PostgresStorageGetter interface {
	GetByOriginalURL(originalURL string) (string, bool)
}

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) *Service {
	return &Service{storage: storage}
}

// Ping проверяет соединение с хранилищем
func (s *Service) Ping() error {
	if s.storage == nil {
		return errors.New("storage is nil")
	}
	return s.storage.Ping()
}

// GetStore возвращает хранилище для доступа к интерфейсу Storage
func (s *Service) GetStore() storage.Storage {
	return s.storage
}

func (s *Service) Save(ctx context.Context, originalURL string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := generateID(length)
		if err != nil {
			return "", err
		}

		err = s.storage.Save(ctx, id, originalURL)
		if err == nil {
			return id, nil
		}

		lastErr = err

		// Если URL уже существует — возвращаем сразу (не retry)
		if errors.Is(err, storage.ErrURLAlreadyExists) {
			// Пытаемся получить существующий shortURL по originalURL
			if getter, ok := s.storage.(PostgresStorageGetter); ok {
				if existingID, found := getter.GetByOriginalURL(originalURL); found {
					return existingID, ErrURLAlreadyExists
				}
			}
			return "", ErrURLAlreadyExists
		}

		// Если коллизия short ID — пробуем следующую итерацию
		if errors.Is(err, storage.ErrAlreadyExists) {
			continue
		}

		// Любая другая ошибка — завершаем сразу
		return "", fmt.Errorf("ошибка сохранения ссылки: %w", err)
	}

	// Если исчерпаны все попытки
	return "", fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

func (s *Service) Get(ctx context.Context, shortID string) (string, error) {
	return s.storage.Get(ctx, shortID)
}

// SaveBatch saves multiple URLs in one operation
func (s *Service) SaveBatch(ctx context.Context, urls []storage.URLRecord) ([]storage.URLRecord, error) {
	// Generate short IDs for all URLs first (to maintain correlation with correlation_id)
	records := make([]storage.URLRecord, 0, len(urls))
	for _, rec := range urls {
		shortID, err := generateID(length)
		if err != nil {
			return nil, err
		}
		records = append(records, storage.URLRecord{
			ShortURL:    shortID,
			OriginalURL: rec.OriginalURL,
		})
	}

	// Save all URLs in batch
	return s.storage.SaveBatch(ctx, records)
}

func generateID(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

func IsURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

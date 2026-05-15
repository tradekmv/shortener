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

// Ошибки сервиса
var (
	ErrMaxRetriesExceeded = errors.New("не удалось сгенерировать уникальный ID после максимального количества попыток")
	ErrURLAlreadyExists   = errors.New("URL уже существует")
	ErrAlreadyExists      = errors.New("короткий ID уже существует")
	ErrNotFound           = errors.New("короткая ссылка не найдена")
	ErrDeletedGone        = errors.New("URL удалён")
)

// PostgresStorageGetter интерфейс для хранилищ с поддержкой поиска по original_url
type PostgresStorageGetter interface {
	GetByOriginalURL(originalURL string) (string, bool)
}

// StorageWithUserID интерфейс для хранилищ с поддержкой user_id
type StorageWithUserID interface {
	SaveWithUserID(ctx context.Context, shortID, originalURL, userID string) error
}

// Service предоставляет бизнес-логику для работы с URL-ссылками
type Service struct {
	storage storage.Storage
}

// NewService создает новый экземпляр Service
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

// Get возвращает оригинальный URL по shortID
func (s *Service) Get(ctx context.Context, shortID string) (string, error) {
	url, err := s.storage.Get(ctx, shortID)
	if err != nil {
		if errors.Is(err, storage.ErrDeletedGone) {
			return "", ErrDeletedGone
		}
		return "", err
	}
	return url, nil
}

// Save saves the original URL and returns the short ID
func (s *Service) Save(ctx context.Context, originalURL string) (string, error) {
	return s.SaveWithUserID(ctx, originalURL, "")
}

// SaveWithUserID saves the original URL with user ID and returns the short ID
func (s *Service) SaveWithUserID(ctx context.Context, originalURL, userID string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := generateID(length)
		if err != nil {
			return "", err
		}

		// Проверяем, поддерживает ли storage сохранение с userID
		if saver, ok := s.storage.(StorageWithUserID); ok {
			err = saver.SaveWithUserID(ctx, id, originalURL, userID)
		} else {
			err = s.storage.Save(ctx, id, originalURL)
		}

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

		// Если коллизия short id — пробуем следующую итерацию
		if errors.Is(err, storage.ErrAlreadyExists) {
			continue
		}

		// Любая другая ошибка — завершаем сразу
		return "", fmt.Errorf("ошибка сохранения ссылки: %w", err)
	}

	// Если исчерпаны все попытки
	return "", fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// GetURLByUser retrieves URL by short ID
func (s *Service) GetURLByUser(ctx context.Context, shortID string) (string, error) {
	return s.storage.Get(ctx, shortID)
}

// GetUserURLs returns all URLs for the given user ID
func (s *Service) GetUserURLs(ctx context.Context, userID string) ([]storage.URLRecord, error) {
	return s.storage.GetUserURLs(ctx, userID)
}

// DeleteUserURLs marks URLs as deleted for the given user (async operation)
func (s *Service) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	return s.storage.DeleteUserURLs(ctx, userID, shortIDs)
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

// IsURL проверяет, является ли строка валидным URL (http:// или https://)
func IsURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

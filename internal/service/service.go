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

var ErrMaxRetriesExceeded = errors.New("не удалось сгенерировать уникальный ID после максимального количества попыток")

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) Save(ctx context.Context, originalURL string) (string, error) {
	for i := 0; i < maxAttempts; i++ {
		id, err := generateID(length)
		if err != nil {
			return "", err
		}

		err = s.storage.Save(id, originalURL)
		if err == nil {
			return id, nil
		}

		if !errors.Is(err, storage.ErrAlreadyExists) {
			return "", fmt.Errorf("ошибка сохранения ссылки: %w", err)
		}
	}
	return "", ErrMaxRetriesExceeded
}

func (s *Service) Get(shortID string) (string, bool) {
	return s.storage.Get(shortID)
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
	return s.storage.SaveBatch(records)
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

// Package service реализует бизнес-логику сервиса сокращения URL:
// генерацию коротких идентификаторов, валидацию URL и работу с хранилищем.
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

// Ошибки сервиса.
var (
	// ErrMaxRetriesExceeded возвращается, если не удалось сгенерировать
	// уникальный ID за максимальное количество попыток.
	ErrMaxRetriesExceeded = errors.New("не удалось сгенерировать уникальный ID после максимального количества попыток")
	// ErrURLAlreadyExists возвращается, когда оригинальный URL уже сокращён.
	ErrURLAlreadyExists = errors.New("URL уже существует")
	// ErrAlreadyExists возвращается при коллизии сгенерированного короткого ID.
	ErrAlreadyExists = errors.New("короткий ID уже существует")
	// ErrNotFound возвращается, если короткая ссылка не найдена.
	ErrNotFound = errors.New("короткая ссылка не найдена")
	// ErrDeletedGone возвращается, если URL был удалён пользователем.
	ErrDeletedGone = errors.New("URL удалён")
)

// Service предоставляет бизнес-логику для работы с URL-ссылками.
// Инкапсулирует хранилище и применяет правила генерации идентификаторов.
type Service struct {
	storage storage.Storage
}

// NewService создаёт новый экземпляр Service поверх переданного хранилища.
// storage — реализация интерфейса storage.Storage (memory, file, postgres).
func NewService(storage storage.Storage) *Service {
	return &Service{storage: storage}
}

// Ping проверяет соединение с хранилищем.
// Возвращает ошибку, если хранилище недоступно или не инициализировано.
func (s *Service) Ping() error {
	if s.storage == nil {
		return errors.New("storage is nil")
	}
	return s.storage.Ping()
}

// GetStore возвращает хранилище для прямого доступа к интерфейсу Storage.
// Используется в обработчиках, которым нужны методы, отсутствующие в Service.
func (s *Service) GetStore() storage.Storage {
	return s.storage
}

// Get возвращает оригинальный URL по его короткому идентификатору.
// Возвращает ErrDeletedGone, если URL был удалён пользователем.
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

// Save сокращает originalURL и возвращает сгенерированный short ID.
// Используется для анонимного сохранения без привязки к пользователю.
func (s *Service) Save(ctx context.Context, originalURL string) (string, error) {
	return s.SaveWithUserID(ctx, originalURL, "")
}

// SaveWithUserID сокращает originalURL и привязывает его к userID.
// При коллизии ID выполняется до maxAttempts повторных попыток генерации.
// Если originalURL уже существует, возвращает ErrURLAlreadyExists и существующий short ID.
func (s *Service) SaveWithUserID(ctx context.Context, originalURL, userID string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := generateID(length)
		if err != nil {
			return "", err
		}

		err = s.storage.SaveWithUserID(ctx, id, originalURL, userID)

		if err == nil {
			return id, nil
		}

		lastErr = err

		// Если URL уже существует — возвращаем сразу (не retry)
		if errors.Is(err, storage.ErrURLAlreadyExists) {
			// Пытаемся получить существующий shortURL по originalURL
			if existingID, found := s.storage.GetByOriginalURL(originalURL); found {
				return existingID, ErrURLAlreadyExists
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

// GetURLByUser возвращает URL по его короткому ID.
// Используется при чтении ссылки владельцем.
func (s *Service) GetURLByUser(ctx context.Context, shortID string) (string, error) {
	return s.storage.Get(ctx, shortID)
}

// GetUserURLs возвращает все сокращённые URL пользователя.
// Возвращает пустой слайс, если у пользователя нет ссылок.
func (s *Service) GetUserURLs(ctx context.Context, userID string) ([]storage.URLRecord, error) {
	return s.storage.GetUserURLs(ctx, userID)
}

// DeleteUserURLs помечает URL пользователя как удалённые.
// Операция выполняется асинхронно батчем в хранилище.
func (s *Service) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	return s.storage.DeleteUserURLs(ctx, userID, shortIDs)
}

// SaveBatch сокращает несколько URL за один вызов.
// Для каждого URL генерируется уникальный short ID.
// Возвращает слайс записей с заполненными ShortURL и OriginalURL.
func (s *Service) SaveBatch(ctx context.Context, urls []storage.URLRecord) ([]storage.URLRecord, error) {
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
	return s.storage.SaveBatch(ctx, records)
}

// fastCharsetIndex быстро маппит случайный байт в индекс charset
// через таблицу предвычислений, чтобы избежать деления
var fastCharsetIndex [256]byte

func init() {
	for i := range fastCharsetIndex {
		fastCharsetIndex[i] = byte(charset[i%len(charset)])
	}
}

func generateID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = fastCharsetIndex[b[i]]
	}
	return string(b), nil
}

// IsURL проверяет, является ли строка валидным URL (http:// или https://)
func IsURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

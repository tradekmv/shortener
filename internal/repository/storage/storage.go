//go:generate mockgen -source=storage.go -destination=mock/mock.go

// Пакет storage предоставляет интерфейсы и реализации хранилища для URL-ссылок
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

// Ошибки хранилища
var (
	ErrAlreadyExists    = errors.New("короткая ссылка уже существует")
	ErrURLAlreadyExists = errors.New("URL уже существует")
	ErrNotFound         = errors.New("короткая ссылка не найдена")
	ErrDeletedGone      = errors.New("URL удалён")
)

// URLRecord представляет запись сокращённой ссылки
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

// Storage интерфейс хранилища
type Storage interface {
	Save(ctx context.Context, shortID, originalURL string) error
	Get(ctx context.Context, shortID string) (string, error)
	// SaveBatch saves multiple URLs in one operation.
	SaveBatch(ctx context.Context, urls []URLRecord) ([]URLRecord, error)
	// GetUserURLs возвращает все URLs для указанного userID
	GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error)
	// DeleteUserURLs помечает URL как удалённые (только для владельца)
	DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error
	Close() error
	Ping() error
}

// Pinger интерфейс для проверки соединения
type Pinger interface {
	Ping() error
}

// Shortener реализует файловый storage для хранения URL
type Shortener struct {
	mu       sync.RWMutex
	storage  map[string]string
	filePath string
}

// New создает новый экземпляр Shortener с файловым хранилищем
func New(filePath string) (*Shortener, error) {
	s := &Shortener{
		storage:  make(map[string]string),
		filePath: filePath,
	}
	if err := s.loadFromFile(); err != nil {
		log.Error().Err(err).Msg("Не удалось загрузить данные из файла")
	}
	return s, nil
}

func (s *Shortener) loadFromFile() error {
	if s.filePath == "" {
		return nil
	}
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		log.Error().Err(err).Msg("Ошибка чтения файла")
		return err
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		log.Error().Err(err).Msg("Ошибка парсинга JSON")
		return err
	}

	for _, rec := range records {
		s.storage[rec.ShortURL] = rec.OriginalURL
	}
	return nil
}

func (s *Shortener) saveToFile() error {
	if s.filePath == "" {
		return nil
	}
	var records []URLRecord
	for shortURL, originalURL := range s.storage {
		records = append(records, URLRecord{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	data, err := json.Marshal(records)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		log.Error().Err(err).Msg("Ошибка записи в файл")
		return err
	}
	return nil
}

// Save сохраняет короткий ID и оригинальный URL
func (s *Shortener) Save(ctx context.Context, shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storage[shortID]; ok {
		return ErrAlreadyExists
	}

	s.storage[shortID] = originalURL
	if err := s.saveToFile(); err != nil {
		return fmt.Errorf("ошибка сохранения в файл: %w", err)
	}
	return nil
}

// Get возвращает оригинальный URL по shortID
func (s *Shortener) Get(ctx context.Context, shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.storage[shortID]
	if !ok {
		return "", ErrNotFound
	}
	return originalURL, nil
}

// SaveBatch saves multiple URLs in one operation for file storage
func (s *Shortener) SaveBatch(ctx context.Context, urls []URLRecord) ([]URLRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]URLRecord, 0, len(urls))
	for _, rec := range urls {
		if existingURL, ok := s.storage[rec.ShortURL]; ok {
			if existingURL == rec.OriginalURL {
				result = append(result, URLRecord{
					ShortURL:    rec.ShortURL,
					OriginalURL: rec.OriginalURL,
				})
			}
			continue
		}
		s.storage[rec.ShortURL] = rec.OriginalURL
		result = append(result, URLRecord{
			ShortURL:    rec.ShortURL,
			OriginalURL: rec.OriginalURL,
		})
	}

	if err := s.saveToFile(); err != nil {
		return nil, fmt.Errorf("ошибка сохранения в файл: %w", err)
	}
	return result, nil
}

// Close закрывает хранилище (пустая реализация для файлового хранилища)
func (s *Shortener) Close() error {
	return nil
}

// Ping проверяет доступность хранилища
func (s *Shortener) Ping() error {
	return nil
}

// GetUserURLs возвращает все URLs для указанного userID (файловое хранилище не поддерживает множественных пользователей)
func (s *Shortener) GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error) {
	return nil, nil
}

// DeleteUserURLs помечает URLs как удалённые (файловое хранилище не поддерживает userID, просто помечает)
func (s *Shortener) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Для файлового хранилища без userID просто удаляем записи
	// В реальной реализации здесь была бы логика проверки owner
	for _, id := range shortIDs {
		delete(s.storage, id)
	}
	return s.saveToFile()
}

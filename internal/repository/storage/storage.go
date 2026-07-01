// Package storage предоставляет интерфейсы и реализации хранилища для URL-ссылок.
//
// Поддерживаются три реализации:
//   - MemoryStorage — данные в оперативной памяти (для тестов и быстрого прототипирования).
//   - Shortener — файловое хранилище, перезаписывает JSON-файл при каждом изменении.
//   - PostgresStorage — реляционное хранилище с уникальными индексами.
//go:generate mockgen -source=storage.go -destination=mock/mock.go

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

// Ошибки хранилища.
var (
	// ErrAlreadyExists возвращается, когда короткий ID уже занят.
	ErrAlreadyExists = errors.New("короткая ссылка уже существует")
	// ErrURLAlreadyExists возвращается, когда оригинальный URL уже сокращён.
	ErrURLAlreadyExists = errors.New("URL уже существует")
	// ErrNotFound возвращается, если короткая ссылка не найдена.
	ErrNotFound = errors.New("короткая ссылка не найдена")
	// ErrDeletedGone возвращается, если URL был удалён пользователем.
	ErrDeletedGone = errors.New("URL удалён")
)

// URLRecord представляет одну запись сокращённой ссылки.
// Используется для чтения/записи между слоями сервиса и хранилищем.
type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

// Storage — интерфейс хранилища URL-ссылок.
// Реализации должны быть безопасны для конкурентного использования.
type Storage interface {
	// Save сохраняет пару (shortID, originalURL).
	Save(ctx context.Context, shortID, originalURL string) error
	// SaveWithUserID сохраняет URL с привязкой к userID.
	SaveWithUserID(ctx context.Context, shortID, originalURL, userID string) error
	// Get возвращает originalURL по shortID.
	Get(ctx context.Context, shortID string) (string, error)
	// GetByOriginalURL возвращает shortURL по originalURL.
	// Используется для идемпотентного сохранения.
	GetByOriginalURL(originalURL string) (string, bool)
	// SaveBatch сокращает несколько URL за один вызов.
	SaveBatch(ctx context.Context, urls []URLRecord) ([]URLRecord, error)
	// GetUserURLs возвращает все URL пользователя.
	GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error)
	// DeleteUserURLs помечает URL пользователя как удалённые.
	DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error
	// Close освобождает ресурсы хранилища.
	Close() error
	// Ping проверяет доступность хранилища.
	Ping() error
}

// Pinger — интерфейс проверки соединения.
// Любая реализация может быть проверена на доступность через Ping().
type Pinger interface {
	Ping() error
}

// Shortener — файловая реализация хранилища Storage.
// Хранит данные в JSON-файле, перезаписывая его при каждом изменении.
// Подходит для небольших нагрузок и однопроцессных развёртываний.
type Shortener struct {
	mu       sync.RWMutex
	storage  map[string]string
	filePath string
}

// New создаёт новый экземпляр Shortener.
// filePath — путь к JSON-файлу с данными. Если пустая строка, файл не используется.
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

// Save сохраняет короткий ID и оригинальный URL.
// При коллизии shortID возвращает ErrAlreadyExists.
// После каждого вызова перезаписывает файл хранилища.
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

// SaveWithUserID сохраняет URL с привязкой к userID.
// В файловой реализации userID игнорируется.
func (s *Shortener) SaveWithUserID(ctx context.Context, shortID, originalURL, userID string) error {
	return s.Save(ctx, shortID, originalURL)
}

// Get возвращает оригинальный URL по shortID.
// Возвращает ErrNotFound, если запись отсутствует.
func (s *Shortener) Get(ctx context.Context, shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.storage[shortID]
	if !ok {
		return "", ErrNotFound
	}
	return originalURL, nil
}

// GetByOriginalURL возвращает shortURL по originalURL.
// Реализация линейно сканирует map (O(n)).
func (s *Shortener) GetByOriginalURL(originalURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for shortURL, origURL := range s.storage {
		if origURL == originalURL {
			return shortURL, true
		}
	}
	return "", false
}

// SaveBatch сохраняет несколько URL за один вызов.
// Дубликаты с совпадающим originalURL пропускаются.
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

// Close освобождает ресурсы (для файлового хранилища — no-op).
func (s *Shortener) Close() error {
	return nil
}

// Ping всегда возвращает nil (файловое хранилище всегда доступно).
func (s *Shortener) Ping() error {
	return nil
}

// GetUserURLs не поддерживается файловым хранилищем.
// Всегда возвращает (nil, nil).
func (s *Shortener) GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error) {
	return nil, nil
}

// DeleteUserURLs удаляет записи по shortIDs.
// Файловое хранилище не различает владельцев, поэтому удаляет без проверки userID.
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

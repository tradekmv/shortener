//go:generate mockgen -source=storage.go -destination=mock/mock.go

package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

var ErrAlreadyExists = errors.New("короткая ссылка уже существует")

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Storage интерфейс хранилища
type Storage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
	Close() error
	Ping() error
}

// Pinger интерфейс для проверки соединения
type Pinger interface {
	Ping() error
}

type Shortener struct {
	mu       sync.RWMutex
	storage  map[string]string
	filePath string
}

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

func (s *Shortener) Save(shortID, originalURL string) error {
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

func (s *Shortener) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.storage[shortID]
	return originalURL, ok
}

// Close закрывает хранилище (пустая реализация для файлового хранилища)
func (s *Shortener) Close() error {
	return nil
}

// Ping проверяет доступность хранилища
func (s *Shortener) Ping() error {
	return nil
}

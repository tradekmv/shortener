package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

var ErrAlreadyExists = errors.New("короткая ссылка уже существует")

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
}

type Shortener struct {
	mu       sync.RWMutex
	storage  map[string]string
	filePath string
}

func New(filePath string) *Shortener {
	s := &Shortener{
		storage:  make(map[string]string),
		filePath: filePath,
	}
	s.loadFromFile()
	return s
}

func (s *Shortener) loadFromFile() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}

	for _, rec := range records {
		s.storage[rec.ShortURL] = rec.OriginalURL
	}
}

func (s *Shortener) saveToFile() {
	var records []URLRecord
	for shortURL, originalURL := range s.storage {
		records = append(records, URLRecord{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	data, err := json.Marshal(records)
	if err != nil {
		return
	}

	os.WriteFile(s.filePath, data, 0644)
}

func (s *Shortener) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storage[shortID]; ok {
		return ErrAlreadyExists
	}

	s.storage[shortID] = originalURL
	s.saveToFile()
	return nil
}

func (s *Shortener) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.storage[shortID]
	return originalURL, ok
}

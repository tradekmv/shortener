package storage

import (
	"errors"
	"sync"
)

var ErrAlreadyExists = errors.New("короткая ссылка уже существует")

type Storage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
}

type Shortener struct {
	mu      sync.RWMutex
	storage map[string]string
}

func New() *Shortener {
	return &Shortener{
		storage: make(map[string]string),
	}
}

func (s *Shortener) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storage[shortID]; ok {
		return ErrAlreadyExists
	}

	s.storage[shortID] = originalURL
	return nil
}

func (s *Shortener) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.storage[shortID]
	return originalURL, ok
}

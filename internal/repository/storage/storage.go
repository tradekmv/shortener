package storage

import (
	"sync"
)

type Storage interface {
	Save(shortID, originalURL string)
	Get(shortID string) (string, bool)
	Exists(shortID string) bool
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

func (s *Shortener) Save(shortID, originalURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storage[shortID] = originalURL
}

func (s *Shortener) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, exists := s.storage[shortID]
	return originalURL, exists
}

func (s *Shortener) Exists(shortID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.storage[shortID]
	return exists
}

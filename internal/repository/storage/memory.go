package storage

import (
	"sync"
)

// MemoryStorage хранит данные в памяти
type MemoryStorage struct {
	mu      sync.RWMutex
	storage map[string]string
}

// NewMemory создаёт новое хранилище в памяти
func NewMemory() *MemoryStorage {
	return &MemoryStorage{
		storage: make(map[string]string),
	}
}

// Save сохраняет пару shortURL → originalURL
func (s *MemoryStorage) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storage[shortID]; ok {
		return ErrAlreadyExists
	}
	s.storage[shortID] = originalURL
	return nil
}

// Get возвращает originalURL по shortID
func (s *MemoryStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.storage[shortID]
	return url, ok
}

// Len возвращает количество записей
func (s *MemoryStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.storage)
}

// Close закрывает хранилище (пустая реализация для памяти)
func (s *MemoryStorage) Close() error {
	return nil
}

// Ping проверяет доступность хранилища
func (s *MemoryStorage) Ping() error {
	return nil
}

package storage

import (
	"sync"
)

// Storage определяет интерфейс для хранилища URL
type Storage interface {
	Save(originalURL string) string
	Get(shortID string) (string, bool)
}

// Shortener реализует интерфейс Storage с использованием in-memory хранилища
type Shortener struct {
	mu      sync.RWMutex
	counter int64
	storage map[string]string
}

func New() *Shortener {
	return &Shortener{
		storage: make(map[string]string),
		counter: 1000000,
	}
}

// Save сохраняет URL и возвращает короткий ID
func (s *Shortener) Save(originalURL string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++

	shortID := s.encodeBase62(s.counter)

	s.storage[shortID] = originalURL

	return shortID
}

// Get возвращает оригинальный URL по короткому ID
func (s *Shortener) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, exists := s.storage[shortID]
	return originalURL, exists
}

// encodeBase62 преобразует число в строку Base62
func (s *Shortener) encodeBase62(num int64) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const base = 62

	if num == 0 {
		return string(chars[0])
	}

	var result []byte
	for num > 0 {
		remainder := num % base
		result = append([]byte{chars[remainder]}, result...)
		num = num / base
	}
	return string(result)
}

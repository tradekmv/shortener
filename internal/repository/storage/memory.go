package storage

import (
	"context"
	"sync"
)

// MemoryStorage хранит данные в памяти процесса.
// Все данные теряются при перезапуске.
// Безопасен для конкурентного использования.
type MemoryStorage struct {
	mu      sync.RWMutex
	storage map[string]string
}

// NewMemory создаёт новое хранилище в памяти.
// Используется в тестах и при прототипировании.
func NewMemory() *MemoryStorage {
	return &MemoryStorage{
		storage: make(map[string]string),
	}
}

// Save сохраняет пару (shortURL, originalURL) в памяти.
// При коллизии возвращает ErrAlreadyExists.
func (s *MemoryStorage) Save(ctx context.Context, shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.storage[shortID]; ok {
		return ErrAlreadyExists
	}
	s.storage[shortID] = originalURL
	return nil
}

// SaveWithUserID сохраняет URL с привязкой к userID.
// В MemoryStorage userID игнорируется (нет multi-user поддержки).
func (s *MemoryStorage) SaveWithUserID(ctx context.Context, shortID, originalURL, userID string) error {
	return s.Save(ctx, shortID, originalURL)
}

// Get возвращает originalURL по shortID.
// Возвращает ErrNotFound, если запись отсутствует.
func (s *MemoryStorage) Get(ctx context.Context, shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.storage[shortID]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

// GetByOriginalURL возвращает shortURL по originalURL.
// Реализация линейно сканирует map (O(n)).
func (s *MemoryStorage) GetByOriginalURL(originalURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortURL, url := range s.storage {
		if url == originalURL {
			return shortURL, true
		}
	}
	return "", false
}

// SaveBatch сохраняет несколько URL за один вызов (с удержанием одного мьютекса).
// Дубликаты по (shortURL, originalURL) пропускаются.
// Заполняет переданный слайс in-place и возвращает его обрезанную часть.
func (s *MemoryStorage) SaveBatch(ctx context.Context, urls []URLRecord) ([]URLRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := 0
	for _, rec := range urls {
		if existingURL, ok := s.storage[rec.ShortURL]; ok {
			if existingURL == rec.OriginalURL {
				urls[n] = rec
				n++
			}
			continue
		}
		s.storage[rec.ShortURL] = rec.OriginalURL
		urls[n] = rec
		n++
	}
	return urls[:n], nil
}

// Len возвращает количество записей в хранилище.
func (s *MemoryStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.storage)
}

// Close освобождает ресурсы (для MemoryStorage — no-op).
func (s *MemoryStorage) Close() error {
	return nil
}

// Ping всегда возвращает nil (память всегда доступна).
func (s *MemoryStorage) Ping() error {
	return nil
}

// GetUserURLs не поддерживается MemoryStorage.
// Всегда возвращает (nil, nil).
func (s *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error) {
	return nil, nil
}

// DeleteUserURLs удаляет записи по shortIDs.
// MemoryStorage не различает владельцев.
func (s *MemoryStorage) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range shortIDs {
		delete(s.storage, id)
	}
	return nil
}

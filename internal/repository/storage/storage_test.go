package storage

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
)

func TestShortener_Save(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	err = s.Save(context.Background(), "abc123", "https://example.com")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}

	url, err := s.Get(context.Background(), "abc123")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestShortener_Get_Found(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	s.Save(context.Background(), "abc123", "https://example.com")

	url, err := s.Get(context.Background(), "abc123")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestShortener_Get_NotFound(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	_, err = s.Get(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ожидалась ошибка ErrNotFound, получена: %v", err)
	}
}

func TestShortener_Save_Success(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	err = s.Save(context.Background(), "abc123", "https://example.com")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
}

func TestShortener_Concurrent(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.Save(context.Background(), string(rune('a'+id%26))+string(rune('0'+id%10)), "https://example.com")
		}(i)
	}
	wg.Wait()

	if len(s.storage) != 100 {
		t.Errorf("ожидалось 100 записей, получено %d", len(s.storage))
	}
}

func TestShortener_GetByOriginalURL(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	s.Save(context.Background(), "abc123", "https://example.com")

	shortURL, found := s.GetByOriginalURL("https://example.com")
	if !found {
		t.Error("ожидалось найти shortURL")
	}
	if shortURL != "abc123" {
		t.Errorf("ожидался shortURL 'abc123', получен '%s'", shortURL)
	}
}

func TestShortener_GetByOriginalURL_NotFound(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	_, found := s.GetByOriginalURL("https://notfound.com")
	if found {
		t.Error("не ожидалось находить shortURL")
	}
}

func TestShortener_DeleteUserURLs(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}
	s.Save(context.Background(), "abc123", "https://example.com")
	s.Save(context.Background(), "def456", "https://test.com")

	err = s.DeleteUserURLs(context.Background(), "user1", []string{"abc123"})
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}

	_, err = s.Get(context.Background(), "abc123")
	if err == nil {
		t.Error("ожидалась ошибка ErrNotFound после удаления")
	}

	_, err = s.Get(context.Background(), "def456")
	if err != nil {
		t.Errorf("неожиданная ошибка для def456: %v", err)
	}
}

func TestShortener_FileStorage(t *testing.T) {
	tmpFile := t.TempDir() + "/test_storage.json"
	defer os.Remove(tmpFile)

	s, err := New(tmpFile)
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	err = s.Save(context.Background(), "short1", "https://example1.com")
	if err != nil {
		t.Fatalf("ошибка сохранения: %v", err)
	}
	err = s.Save(context.Background(), "short2", "https://example2.com")
	if err != nil {
		t.Fatalf("ошибка сохранения: %v", err)
	}

	s2, err := New(tmpFile)
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	url, err := s2.Get(context.Background(), "short1")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if url != "https://example1.com" {
		t.Errorf("ожидался URL 'https://example1.com', получен '%s'", url)
	}

	url, err = s2.Get(context.Background(), "short2")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if url != "https://example2.com" {
		t.Errorf("ожидался URL 'https://example2.com', получен '%s'", url)
	}
}

func TestShortener_SaveBatch(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	urls := []URLRecord{
		{ShortURL: "s1", OriginalURL: "https://a.com"},
		{ShortURL: "s2", OriginalURL: "https://b.com"},
		{ShortURL: "s3", OriginalURL: "https://c.com"},
	}

	result, err := s.SaveBatch(context.Background(), urls)
	if err != nil {
		t.Fatalf("ошибка SaveBatch: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("ожидалось 3 записи, получено %d", len(result))
	}
}

func TestShortener_Save_AlreadyExists(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	err = s.Save(context.Background(), "id1", "https://example.com")
	if err != nil {
		t.Fatalf("первое сохранение: %v", err)
	}

	err = s.Save(context.Background(), "id1", "https://other.com")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("ожидалась ErrAlreadyExists, получена: %v", err)
	}
}

func TestShortener_SaveWithUserID(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	err = s.SaveWithUserID(context.Background(), "id1", "https://example.com", "user-1")
	if err != nil {
		t.Fatalf("SaveWithUserID: %v", err)
	}

	url, err := s.Get(context.Background(), "id1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestShortener_Ping(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	if err := s.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestShortener_Close(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestShortener_GetUserURLs(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("ошибка создания хранилища: %v", err)
	}

	urls, err := s.GetUserURLs(context.Background(), "user1")
	if err != nil {
		t.Errorf("GetUserURLs: %v", err)
	}
	if urls != nil {
		t.Errorf("ожидался nil, получено: %v", urls)
	}
}

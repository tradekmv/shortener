package storage

import (
	"context"
	"errors"
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

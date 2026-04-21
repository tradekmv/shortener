package storage

import (
	"sync"
	"testing"
)

func TestShortener_Save(t *testing.T) {
	s := New("")
	err := s.Save("abc123", "https://example.com")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}

	url, ok := s.Get("abc123")
	if !ok {
		t.Errorf("ожидалось, что ключ 'abc123' существует")
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestShortener_Get_Found(t *testing.T) {
	s := New("")
	s.Save("abc123", "https://example.com")

	url, ok := s.Get("abc123")
	if !ok {
		t.Errorf("ожидалось, что ключ 'abc123' существует")
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestShortener_Get_NotFound(t *testing.T) {
	s := New("")

	_, ok := s.Get("nonexistent")
	if ok {
		t.Errorf("ожидалось, что ключ 'nonexistent' не существует")
	}
}

func TestShortener_Save_Success(t *testing.T) {
	s := New("")
	err := s.Save("abc123", "https://example.com")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
}

func TestShortener_Concurrent(t *testing.T) {
	s := New("")
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.Save(string(rune('a'+id%26))+string(rune('0'+id%10)), "https://example.com")
		}(i)
	}
	wg.Wait()

	if len(s.storage) != 100 {
		t.Errorf("ожидалось 100 записей, получено %d", len(s.storage))
	}
}

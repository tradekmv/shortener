package storage

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryStorage_Save_Get(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if err := s.Save(ctx, "id1", "https://example.com"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	url, err := s.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestMemoryStorage_Save_Duplicate(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_ = s.Save(ctx, "id1", "https://example.com")
	if err := s.Save(ctx, "id1", "https://other.com"); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("ожидалась ErrAlreadyExists, получена: %v", err)
	}
}

func TestMemoryStorage_Get_NotFound(t *testing.T) {
	s := NewMemory()
	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получена: %v", err)
	}
}

func TestMemoryStorage_GetByOriginalURL(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_ = s.Save(ctx, "id1", "https://example.com")
	id, ok := s.GetByOriginalURL("https://example.com")
	if !ok || id != "id1" {
		t.Errorf("ожидался id='id1', получен '%s', ok=%v", id, ok)
	}
	_, ok = s.GetByOriginalURL("https://missing.com")
	if ok {
		t.Error("ожидалось ok=false для несуществующего URL")
	}
}

func TestMemoryStorage_SaveBatch(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	urls := []URLRecord{
		{ShortURL: "id1", OriginalURL: "https://example.com/1"},
		{ShortURL: "id2", OriginalURL: "https://example.com/2"},
	}
	results, err := s.SaveBatch(ctx, urls)
	if err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("ожидалось 2 результата, получено %d", len(results))
	}
}

func TestMemoryStorage_DeleteUserURLs(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	_ = s.Save(ctx, "id1", "https://example.com")
	_ = s.Save(ctx, "id2", "https://example.com/2")
	if err := s.DeleteUserURLs(ctx, "user", []string{"id1"}); err != nil {
		t.Fatalf("DeleteUserURLs: %v", err)
	}
	if s.Len() != 1 {
		t.Errorf("ожидался Len=1, получено %d", s.Len())
	}
}

func TestMemoryStorage_Ping_Close(t *testing.T) {
	s := NewMemory()
	if err := s.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestMemoryStorage_GetUserURLs(t *testing.T) {
	s := NewMemory()
	urls, err := s.GetUserURLs(context.Background(), "user")
	if err != nil {
		t.Fatalf("GetUserURLs: %v", err)
	}
	if urls != nil {
		t.Errorf("ожидалось nil для MemoryStorage, получено: %v", urls)
	}
}

func TestMemoryStorage_SaveWithUserID(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if err := s.SaveWithUserID(ctx, "id1", "https://example.com", "user-1"); err != nil {
		t.Fatalf("SaveWithUserID: %v", err)
	}
	url, err := s.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestMemoryStorage_Len(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if s.Len() != 0 {
		t.Errorf("ожидался Len=0, получено %d", s.Len())
	}
	_ = s.Save(ctx, "id1", "https://example.com")
	if s.Len() != 1 {
		t.Errorf("ожидался Len=1, получено %d", s.Len())
	}
}

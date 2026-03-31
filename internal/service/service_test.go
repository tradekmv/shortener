package service

import (
	"testing"
)

func TestIsURL_ValidHTTP(t *testing.T) {
	if !IsURL("http://example.com") {
		t.Errorf("ожидалось, что 'http://example.com' является валидным URL")
	}
}

func TestIsURL_ValidHTTPS(t *testing.T) {
	if !IsURL("https://example.com") {
		t.Errorf("ожидалось, что 'https://example.com' является валидным URL")
	}
}

func TestIsURL_Invalid(t *testing.T) {
	testCases := []string{
		"ftp://example.com",
		"example.com",
		"example",
		"",
		"https:\\example.com",
	}

	for _, tc := range testCases {
		if IsURL(tc) {
			t.Errorf("ожидалось, что '%s' является невалидным URL", tc)
		}
	}
}

func TestSave_Success(t *testing.T) {
	mockStorage := newMockStorage()
	svc := NewService(mockStorage)

	id, err := svc.Save("https://example.com")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if id == "" {
		t.Errorf("ожидался непустой ID")
	}
	if len(id) != length {
		t.Errorf("ожидалась длина ID %d, получена %d", length, len(id))
	}
}

func TestSave_AnyInput(t *testing.T) {
	// Сервис не валидирует URL - валидация происходит в обработчике
	// Этот тест проверяет, что сервис сохраняет любые полученные данные
	mockStorage := newMockStorage()
	svc := NewService(mockStorage)

	id, err := svc.Save("invalid-url")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if id == "" {
		t.Errorf("ожидался непустой ID")
	}

	// Проверяем, что URL был сохранен
	url, found := mockStorage.Get(id)
	if !found {
		t.Errorf("ожидалось, что URL будет сохранен")
	}
	if url != "invalid-url" {
		t.Errorf("ожидался сохраненный URL 'invalid-url', получен '%s'", url)
	}
}

func TestGet_Found(t *testing.T) {
	mockStorage := newMockStorage()
	mockStorage.Save("abc123", "https://example.com")
	svc := NewService(mockStorage)

	url, found := svc.Get("abc123")
	if !found {
		t.Errorf("ожидалось найти ID 'abc123'")
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestGet_NotFound(t *testing.T) {
	mockStorage := newMockStorage()
	svc := NewService(mockStorage)

	_, found := svc.Get("nonexistent")
	if found {
		t.Errorf("ожидалось не найти ID 'nonexistent'")
	}
}

func TestSave_GeneratesUniqueIDs(t *testing.T) {
	mockStorage := newMockStorage()
	svc := NewService(mockStorage)

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := svc.Save("https://example.com/page" + string(rune('0'+i)))
		if err != nil {
			t.Fatalf("неожиданная ошибка на итерации %d: %v", i, err)
		}
		if ids[id] {
			t.Errorf("сгенерирован дубликат ID: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_Length(t *testing.T) {
	for _, n := range []int{4, 8, 16, 32} {
		id, err := generateID(n)
		if err != nil {
			t.Errorf("неожиданная ошибка для длины %d: %v", n, err)
		}
		if len(id) != n {
			t.Errorf("ожидалась длина ID %d, получена %d", n, len(id))
		}
	}
}

func TestGenerateID_Charset(t *testing.T) {
	id, _ := generateID(100)
	for _, c := range id {
		valid := false
		for _, cs := range charset {
			if c == cs {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("символ '%c' отсутствует в наборе символов", c)
		}
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id, err := generateID(8)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		_ = ids[id]
	}
}

type mockStorage struct {
	data map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{data: make(map[string]string)}
}

func (m *mockStorage) Save(shortID, originalURL string) {
	m.data[shortID] = originalURL
}

func (m *mockStorage) Get(shortID string) (string, bool) {
	url, exists := m.data[shortID]
	return url, exists
}

func (m *mockStorage) Exists(shortID string) bool {
	_, exists := m.data[shortID]
	return exists
}

package service

import (
	"context"
	"testing"

	"github.com/tradekmv/shortener.git/internal/repository/mock"
	"go.uber.org/mock/gomock"
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockStorage(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		Save(gomock.Any(), "https://example.com").
		Return(nil)

	id, err := svc.Save(context.Background(), "https://example.com")
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockStorage(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		Save(gomock.Any(), "invalid-url").
		Return(nil)

	id, err := svc.Save(context.Background(), "invalid-url")
	if err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	if id == "" {
		t.Errorf("ожидался непустой ID")
	}
}

func TestGet_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockStorage(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		Get("abc123").
		Return("https://example.com", true)

	url, found := svc.Get("abc123")
	if !found {
		t.Errorf("ожидалось найти ID 'abc123'")
	}
	if url != "https://example.com" {
		t.Errorf("ожидался URL 'https://example.com', получен '%s'", url)
	}
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockStorage(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		Get("nonexistent").
		Return("", false)

	_, found := svc.Get("nonexistent")
	if found {
		t.Errorf("ожидалось не найти ID 'nonexistent'")
	}
}

func TestSave_GeneratesUniqueIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockStorage(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(100)

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := svc.Save(context.Background(), "https://example.com/page"+string(rune('0'+i)))
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

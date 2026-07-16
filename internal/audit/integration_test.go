// Интеграционный тест для проверки аудита
package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestFileObserver_Integration проверяет запись событий в файл
func TestFileObserver_Integration(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_integration_*.json")
	if err != nil {
		t.Fatalf("ошибка создания временного файла: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	observer, err := NewFileObserver(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка создания FileObserver: %v", err)
	}

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user_test_123",
		URL:       "https://mylongdomain.com/my/long/path/to/shorten/",
	}

	if err := observer.Notify(event); err != nil {
		t.Errorf("ошибка записи события: %v", err)
	}

	observer.Close()

	// Проверяем содержимое файла
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка чтения файла: %v", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		t.Errorf("файл пустой после записи события")
	}

	var parsedEvent Event
	if err := json.Unmarshal([]byte(content), &parsedEvent); err != nil {
		t.Errorf("ошибка парсинга JSON: %v\nСодержимое: %s", err, content)
		return
	}

	if parsedEvent.Action != "shorten" {
		t.Errorf("ожидался action 'shorten', получен '%s'", parsedEvent.Action)
	}
	if parsedEvent.UserID != "user_test_123" {
		t.Errorf("ожидался user_id 'user_test_123', получен '%s'", parsedEvent.UserID)
	}
	if parsedEvent.URL != "https://mylongdomain.com/my/long/path/to/shorten/" {
		t.Errorf("ожидался другой URL, получен '%s'", parsedEvent.URL)
	}
}

// TestRemoteObserver_Integration проверяет отправку событий на удалённый сервер
func TestRemoteObserver_Integration(t *testing.T) {
	var receivedEvents []Event
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("ошибка декодирования: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "follow",
		UserID:    "user_remote_456",
		URL:       "https://example.com/follow",
	}

	if err := observer.Notify(event); err != nil {
		t.Errorf("ошибка отправки события: %v", err)
	}

	observer.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 1 {
		t.Errorf("ожидалось 1 событие, получено %d", len(receivedEvents))
		return
	}

	if receivedEvents[0].Action != "follow" {
		t.Errorf("ожидался action 'follow', получен '%s'", receivedEvents[0].Action)
	}
}

// TestPublisher_MultipleObservers проверяет работу с несколькими наблюдателями
func TestPublisher_MultipleObservers(t *testing.T) {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	pub := NewPublisher(&log)

	tmpFile, _ := os.CreateTemp("", "audit_multi_*.json")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	fileObs, err := NewFileObserver(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка создания FileObserver: %v", err)
	}
	defer fileObs.Close()

	var remoteReceived []Event
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		json.NewDecoder(r.Body).Decode(&event)
		mu.Lock()
		remoteReceived = append(remoteReceived, event)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	remoteObs := NewRemoteObserver(server.URL)
	defer remoteObs.Close()

	pub.Subscribe(fileObs)
	pub.Subscribe(remoteObs)

	events := []Event{
		{Timestamp: 1234567890, Action: "shorten", UserID: "u1", URL: "https://a.com"},
		{Timestamp: 1234567891, Action: "follow", UserID: "u2", URL: "https://b.com"},
	}

	for _, event := range events {
		pub.Publish(event)
	}

	// Publish асинхронный: Close дожидается обработки всех событий из очереди.
	if err := pub.Close(); err != nil {
		t.Fatalf("ошибка закрытия Publisher: %v", err)
	}

	// Проверяем файл
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка чтения файла: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"action":"shorten"`) {
		t.Errorf("файл не содержит событие shorten")
	}
	if !strings.Contains(content, `"action":"follow"`) {
		t.Errorf("файл не содержит событие follow")
	}

	// Проверяем удалённый сервер
	mu.Lock()
	rcvd := len(remoteReceived)
	mu.Unlock()
	if rcvd != 2 {
		t.Errorf("ожидалось 2 события на удалённом сервере, получено %d", rcvd)
	}
}

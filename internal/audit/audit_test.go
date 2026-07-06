// Package audit реализует паттерн Observer (Наблюдатель) для аудита запросов.
// Publisher рассылает события аудита всем подписанным наблюдателям (файл, удалённый сервер).
package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

var testLogger = zerolog.New(os.Stdout).With().Timestamp().Logger()

// MockObserver мок-наблюдатель для тестов
type MockObserver struct {
	mu        sync.Mutex
	events    []Event
	notifyErr error
}

func (m *MockObserver) Notify(event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return m.notifyErr
}

func (m *MockObserver) Close() error {
	return nil
}

func (m *MockObserver) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.events...)
}

func TestPublisher_PublishToAllObservers(t *testing.T) {
	log := testLogger
	pub := NewPublisher(&log)
	defer pub.Close()

	observer1 := &MockObserver{}
	observer2 := &MockObserver{}

	pub.Subscribe(observer1)
	pub.Subscribe(observer2)

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user123",
		URL:       "https://example.com",
	}

	pub.Publish(event)

	// Publish асинхронный: Close дожидается обработки всех событий из очереди.
	if err := pub.Close(); err != nil {
		t.Fatalf("ошибка закрытия Publisher: %v", err)
	}

	if len(observer1.Events()) != 1 {
		t.Errorf("ожидалось 1 событие у observer1, получено %d", len(observer1.Events()))
	}
	if len(observer2.Events()) != 1 {
		t.Errorf("ожидалось 1 событие у observer2, получено %d", len(observer2.Events()))
	}

	if observer1.Events()[0].Action != "shorten" {
		t.Errorf("ожидалось Action 'shorten', получено '%s'", observer1.Events()[0].Action)
	}
}

func TestPublisher_NoObservers(t *testing.T) {
	log := testLogger
	pub := NewPublisher(&log)

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		URL:       "https://example.com",
	}

	// Не должно паниковать
	pub.Publish(event)
}

func TestFileObserver_WriteAndClose(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_test_*.json")
	if err != nil {
		t.Fatalf("ошибка создания временного файла: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	observer, err := NewFileObserver(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка создания FileObserver: %v", err)
	}
	defer observer.Close()

	events := []Event{
		{Timestamp: 1234567890, Action: "shorten", UserID: "user1", URL: "https://example.com/1"},
		{Timestamp: 1234567891, Action: "follow", UserID: "user2", URL: "https://example.com/2"},
	}

	for _, event := range events {
		if err := observer.Notify(event); err != nil {
			t.Errorf("ошибка записи события: %v", err)
		}
	}

	// Читаем содержимое файла
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ошибка чтения файла: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Errorf("файл пустой")
	}

	// Проверяем, что каждое событие записано на отдельной строке
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != len(events) {
		t.Errorf("ожидалось %d строк, получено %d", len(events), len(lines))
	}

	// Проверяем валидность JSON
	for i, line := range lines {
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("строка %d не валидный JSON: %v", i+1, err)
		}
	}
}

func TestRemoteObserver_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("ожидался метод POST, получен %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("ожидался Content-Type 'application/json', получен '%s'", ct)
		}

		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("ошибка декодирования JSON: %v", err)
		}
		if event.Action != "shorten" {
			t.Errorf("ожидался Action 'shorten', получен '%s'", event.Action)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)
	defer observer.Close()

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user1",
		URL:       "https://example.com",
	}

	if err := observer.Notify(event); err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
}

func TestRemoteObserver_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)
	defer observer.Close()

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		URL:       "https://example.com",
	}

	err := observer.Notify(event)
	if err == nil {
		t.Errorf("ожидалась ошибка при статусе 500")
	}
}

func TestRemoteObserver_InvalidURL(t *testing.T) {
	observer := NewRemoteObserver("http://invalid-url-that-does-not-exist:99999")
	defer observer.Close()

	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		URL:       "https://example.com",
	}

	err := observer.Notify(event)
	if err == nil {
		t.Errorf("ожидалась ошибка при невалидном URL")
	}
}

func TestPublisher_CloseAll(t *testing.T) {
	log := testLogger
	pub := NewPublisher(&log)

	observer1 := &MockObserver{}
	observer2 := &MockObserver{}

	pub.Subscribe(observer1)
	pub.Subscribe(observer2)

	if err := pub.Close(); err != nil {
		t.Errorf("неожиданная ошибка при закрытии: %v", err)
	}
}

// TestPublisher_DrainQueueOnClose проверяет, что события,
// отправленные непосредственно перед Close, не теряются.
func TestPublisher_DrainQueueOnClose(t *testing.T) {
	log := testLogger
	pub := NewPublisher(&log)

	const count = 100
	observer := &MockObserver{}
	pub.Subscribe(observer)

	for i := 0; i < count; i++ {
		pub.Publish(Event{Action: "shorten", URL: "https://example.com"})
	}

	if err := pub.Close(); err != nil {
		t.Fatalf("ошибка Close: %v", err)
	}

	if got := len(observer.Events()); got != count {
		t.Errorf("после Close ожидалось %d событий, получено %d", count, got)
	}
}

// TestPublisher_PublishFromMultipleGoroutines проверяет,
// что Publish безопасен для конкурентного использования.
func TestPublisher_PublishFromMultipleGoroutines(t *testing.T) {
	log := testLogger
	pub := NewPublisher(&log)

	const (
		writers         = 10
		eventsPerWriter = 50
	)

	observer := &MockObserver{}
	pub.Subscribe(observer)

	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerWriter; j++ {
				pub.Publish(Event{
					Action: "shorten",
					URL:    "https://example.com",
				})
			}
		}(i)
	}
	wg.Wait()

	if err := pub.Close(); err != nil {
		t.Fatalf("ошибка Close: %v", err)
	}

	want := writers * eventsPerWriter
	if got := len(observer.Events()); got != want {
		t.Errorf("ожидалось %d событий, получено %d", want, got)
	}
}

// TestRemoteObserver_RetriesOnServerError проверяет ретраи при 5xx ответах.
func TestRemoteObserver_RetriesOnServerError(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)
	defer observer.Close()

	event := Event{Action: "shorten", URL: "https://example.com"}
	err := observer.Notify(event)

	if err == nil {
		t.Errorf("ожидалась ошибка после ретраев")
	}
	got := atomic.LoadInt32(&attempts)
	if got != int32(remoteMaxRetries) {
		t.Errorf("ожидалось %d попыток, получено %d", remoteMaxRetries, got)
	}
}

// TestRemoteObserver_NoRetryOnClientError проверяет отсутствие ретраев при 4xx.
func TestRemoteObserver_NoRetryOnClientError(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)
	defer observer.Close()

	event := Event{Action: "shorten", URL: "https://example.com"}
	err := observer.Notify(event)

	if err == nil {
		t.Errorf("ожидалась ошибка")
	}
	got := atomic.LoadInt32(&attempts)
	if got != 1 {
		t.Errorf("ожидалась 1 попытка (без ретраев), получено %d", got)
	}
}

// TestRemoteObserver_RetryThenSuccess проверяет успех после нескольких 5xx.
func TestRemoteObserver_RetryThenSuccess(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)
	defer observer.Close()

	event := Event{Action: "shorten", URL: "https://example.com"}
	if err := observer.Notify(event); err != nil {
		t.Errorf("неожиданная ошибка: %v", err)
	}
	got := atomic.LoadInt32(&attempts)
	if got != 2 {
		t.Errorf("ожидалось 2 попытки, получено %d", got)
	}
}

// Компилируем проверку, что MockObserver реализует интерфейс Observer
var _ Observer = (*MockObserver)(nil)
var _ Observer = (*FileObserver)(nil)
var _ Observer = (*RemoteObserver)(nil)

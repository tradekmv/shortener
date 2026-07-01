// Пакет audit реализует паттерн Observer (Наблюдатель) для аудита запросов.
// Publisher рассылает события аудита всем подписанным наблюдателям (файл, удалённый сервер).
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Event представляет одну запись события аудита.
// Сериализуется в JSON для передачи наблюдателям.
type Event struct {
	Timestamp int64  `json:"ts"`      // Unix-время в миллисекундах
	Action    string `json:"action"`  // Действие (например, "shorten", "delete")
	UserID    string `json:"user_id"` // Идентификатор пользователя
	URL       string `json:"url"`     // Затронутый URL
}

// Observer — интерфейс наблюдателя для получения событий аудита.
// Реализации: FileObserver (запись в файл), RemoteObserver (HTTP POST).
type Observer interface {
	Notify(event Event) error
	Close() error
}

// Publisher — издатель событий аудита (паттерн Observer).
// Безопасен для конкурентного использования.
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
	log       *zerolog.Logger
}

// NewPublisher создаёт новый издатель событий аудита.
// log — zerolog логгер для сообщений об ошибках наблюдателей.
func NewPublisher(log *zerolog.Logger) *Publisher {
	return &Publisher{
		observers: make([]Observer, 0),
		log:       log,
	}
}

// Subscribe добавляет наблюдателя к издателю.
// Один наблюдатель может быть подписан несколько раз и получит события несколько раз.
func (p *Publisher) Subscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// Publish отправляет событие всем подписанным наблюдателям.
// Ошибки отдельных наблюдателей логируются, но не прерывают рассылку.
func (p *Publisher) Publish(event Event) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, observer := range p.observers {
		if err := observer.Notify(event); err != nil {
			p.log.Error().Err(err).Msg("Ошибка отправки события аудита")
		}
	}
}

// Close закрывает всех наблюдателей.
// Возвращает агрегированную ошибку, если несколько наблюдателей вернули ошибки.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for _, observer := range p.observers {
		if err := observer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("ошибки закрытия наблюдателей: %v", errs)
	}
	return nil
}

// FileObserver — наблюдатель, записывающий события в JSONL-файл.
// Каждое событие — отдельная строка, синхронизируется с диском через fsync.
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создаёт наблюдатель, записывающий события в указанный файл.
// Файл открывается в режиме append+create. После каждой записи вызывается fsync.
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}

	return &FileObserver{file: file}, nil
}

// Notify записывает событие в файл в формате JSON + перевод строки.
// Вызывает fsync после каждой записи для надёжности.
func (f *FileObserver) Notify(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга события: %w", err)
	}

	// Добавляем новую строку
	data = append(data, '\n')

	if _, err := f.file.Write(data); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

	// Принудительная синхронизация с диском
	if err := f.file.Sync(); err != nil {
		return fmt.Errorf("ошибка синхронизации файла: %w", err)
	}

	return nil
}

// Close закрывает файл наблюдателя.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Close()
}

// RemoteObserver — наблюдатель, отправляющий события на удалённый сервер по HTTP POST.
// Использует http.Client с таймаутом 5 секунд.
type RemoteObserver struct {
	url    string
	client *http.Client
}

// NewRemoteObserver создаёт наблюдатель, отправляющий события на удалённый сервер.
// url — адрес принимающего API в формате http://host/path.
func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие на удалённый сервер по HTTP POST.
// Возвращает ошибку при сетевых проблемах или HTTP-статусе >= 400.
func (r *RemoteObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга события: %w", err)
	}

	resp, err := r.client.Post(r.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("ошибка отправки на удалённый сервер: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("удалённый сервер вернул статус %d", resp.StatusCode)
	}

	return nil
}

// Close закрывает HTTP-клиент (закрывает idle-соединения).
func (r *RemoteObserver) Close() error {
	r.client.CloseIdleConnections()
	return nil
}

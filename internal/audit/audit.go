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

// Event представляет событие аудита
type Event struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}

// Observer интерфейс наблюдателя для получения событий аудита
type Observer interface {
	Notify(event Event) error
	Close() error
}

// Publisher издатель событий аудита (паттерн Observer)
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
	log       *zerolog.Logger
}

// NewPublisher создаёт новый издатель событий аудита
func NewPublisher(log *zerolog.Logger) *Publisher {
	return &Publisher{
		observers: make([]Observer, 0),
		log:       log,
	}
}

// Subscribe добавляет наблюдателя
func (p *Publisher) Subscribe(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// Publish отправляет событие всем наблюдателям
func (p *Publisher) Publish(event Event) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, observer := range p.observers {
		if err := observer.Notify(event); err != nil {
			p.log.Error().Err(err).Msg("Ошибка отправки события аудита")
		}
	}
}

// Close закрывает все наблюдатели
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

// FileObserver наблюдатель для записи событий в файл
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создаёт наблюдатель для записи в файл
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}

	return &FileObserver{file: file}, nil
}

// Notify записывает событие в файл
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

// Close закрывает файл
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Close()
}

// RemoteObserver наблюдатель для отправки событий на удалённый сервер
type RemoteObserver struct {
	url    string
	client *http.Client
}

// NewRemoteObserver создаёт наблюдатель для отправки на удалённый сервер
func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие на удалённый сервер
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

// Close закрывает HTTP клиент
func (r *RemoteObserver) Close() error {
	r.client.CloseIdleConnections()
	return nil
}

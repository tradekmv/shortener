// Package audit реализует паттерн Observer (Наблюдатель) для аудита запросов.
// Publisher рассылает события аудита всем подписанным наблюдателям (файл, удалённый сервер).
package audit

import (
	"bytes"
	"encoding/json"
	"errors"
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
	Timestamp int64  `json:"ts"`      // Unix-время в секундах (time.Now().Unix())
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
//
// Publish не блокирует вызывающую сторону: события кладутся в
// буферизованную очередь и обрабатываются пулом воркеров в фоне.
// При переполнении очереди обработка события делегируется
// отдельной горутине (overflow), отслеживаемой через отдельный
// WaitGroup — Close дожидается завершения всех таких горутин,
// чтобы не потерять события.
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
	log       *zerolog.Logger

	queue    chan Event
	stop     chan struct{}
	wg       sync.WaitGroup // воркеры очереди
	overflow sync.WaitGroup // overflow-горутины
	closing  sync.Once
}

// Конфигурация пула обработки событий аудита.
const (
	// auditQueueSize — размер буфера очереди событий.
	// При переполнении Publish делегирует обработку отдельной overflow-горутине.
	auditQueueSize = 1024
	// auditWorkers — количество воркеров, обрабатывающих очередь.
	auditWorkers = 4
)

// NewPublisher создаёт новый издатель событий аудита.
// log — zerolog логгер для сообщений об ошибках наблюдателей.
// Запускает пул воркеров для асинхронной рассылки событий наблюдателям.
func NewPublisher(log *zerolog.Logger) *Publisher {
	p := &Publisher{
		observers: make([]Observer, 0),
		log:       log,
		queue:     make(chan Event, auditQueueSize),
		stop:      make(chan struct{}),
	}
	for i := 0; i < auditWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

// worker обрабатывает события из очереди последовательно.
// При получении сигнала остановки сначала дочитывает уже принятые в
// очередь события, и только после этого завершается — иначе часть
// событий была бы потеряна при shutdown.
func (p *Publisher) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			// Дочитываем очередь до конца перед выходом.
			for {
				select {
				case event := <-p.queue:
					p.dispatch(event)
				default:
					return
				}
			}
		case event := <-p.queue:
			p.dispatch(event)
		}
	}
}

// dispatch рассылает событие всем подписанным наблюдателям синхронно.
// Вызывается только из воркера; ошибки отдельных наблюдателей логируются.
func (p *Publisher) dispatch(event Event) {
	p.mu.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()

	for _, observer := range observers {
		if err := observer.Notify(event); err != nil {
			p.log.Error().Err(err).Msg("Ошибка отправки события аудита")
		}
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
// Метод не блокирует вызывающую сторону: событие помещается в очередь,
// которую обрабатывают воркеры. Если очередь переполнена, обработка
// события делегируется отдельной горутине, отслеживаемой через
// Publisher.overflow — Close дожидается её завершения.
func (p *Publisher) Publish(event Event) {
	select {
	case p.queue <- event:
		// событие принято в очередь
	default:
		// очередь переполнена — обрабатываем в отдельной горутине,
		// которую пересчитает Close через overflow WaitGroup.
		p.overflow.Add(1)
		go func() {
			defer p.overflow.Done()
			p.dispatch(event)
		}()
	}
}

// Close закрывает издателя и всех наблюдателей.
// Останавливает воркеров, дожидается завершения overflow-горутин,
// затем вызывает Close у каждого наблюдателя. Возвращает агрегированную
// ошибку, если несколько наблюдателей вернули ошибки.
func (p *Publisher) Close() error {
	p.closing.Do(func() {
		close(p.stop)
	})
	p.wg.Wait()
	// Дожидаемся завершения всех overflow-горутин, чтобы не потерять
	// события, отправленные через Publish в обход очереди.
	p.overflow.Wait()

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
// Сериализация в JSON выполняется под мьютексом файла (через
// json.Encoder), чтобы маршалинг и запись были атомарны относительно
// параллельных вызовов Notify; содержимое критической секции при
// этом остаётся минимальным — без отдельного json.Marshal.
func (f *FileObserver) Notify(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	enc := json.NewEncoder(f.file)
	if err := enc.Encode(event); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

	// Принудительная синхронизация с диском.
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

// Конфигурация ретраев для RemoteObserver.
const (
	// remoteMaxRetries — максимальное количество попыток отправки (включая первую).
	remoteMaxRetries = 3
	// remoteRetryBaseDelay — базовая задержка между попытками (экспоненциальный backoff).
	remoteRetryBaseDelay = 100 * time.Millisecond
)

// RemoteObserver — наблюдатель, отправляющий события на удалённый сервер по HTTP POST.
// Использует http.Client с таймаутом 5 секунд и автоматическими ретраями
// при сетевых ошибках или HTTP-статусах 5xx (экспоненциальный backoff).
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
// При сетевых ошибках или HTTP-статусе 5xx выполняется до remoteMaxRetries
// попыток с экспоненциальным backoff. HTTP-статусы 4xx (кроме 429) не ретраятся.
func (r *RemoteObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга события: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < remoteMaxRetries; attempt++ {
		err := r.doPost(data)
		if err == nil {
			return nil
		}
		lastErr = err

		// Не ретраим клиентские ошибки 4xx (кроме 429 Too Many Requests).
		if isClientError(err) && !isTooManyRequests(err) {
			return err
		}

		// Последняя попытка — не спим.
		if attempt == remoteMaxRetries-1 {
			break
		}
		time.Sleep(remoteRetryBaseDelay << attempt)
	}
	return lastErr
}

// doPost выполняет одну попытку HTTP POST.
// Возвращает ошибку для сетевых сбоев, 5xx и 4xx (кроме 429).
func (r *RemoteObserver) doPost(data []byte) error {
	resp, err := r.client.Post(r.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("ошибка отправки на удалённый сервер: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500, resp.StatusCode == http.StatusTooManyRequests:
		return &httpStatusError{status: resp.StatusCode}
	case resp.StatusCode >= 400:
		return &httpStatusError{status: resp.StatusCode}
	default:
		return nil
	}
}

// httpStatusError — ошибка HTTP-ответа с ненормальным статусом.
// Используется для принятия решения о ретраях.
type httpStatusError struct {
	status int
}

// Error возвращает текстовое описание HTTP-статуса.
func (e *httpStatusError) Error() string {
	return fmt.Sprintf("удалённый сервер вернул статус %d", e.status)
}

// isClientError сообщает, является ли ошибка HTTP-статусом 4xx.
func isClientError(err error) bool {
	var httpErr *httpStatusError
	if errors.As(err, &httpErr) {
		return httpErr.status >= 400 && httpErr.status < 500
	}
	return false
}

// isTooManyRequests сообщает, является ли ошибка HTTP 429.
func isTooManyRequests(err error) bool {
	var httpErr *httpStatusError
	if errors.As(err, &httpErr) {
		return httpErr.status == http.StatusTooManyRequests
	}
	return false
}

// Close закрывает HTTP-клиент (закрывает idle-соединения).
func (r *RemoteObserver) Close() error {
	r.client.CloseIdleConnections()
	return nil
}

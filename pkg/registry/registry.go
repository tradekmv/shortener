package registry

import (
	"errors"
	"sync"
)

// ErrAlreadyClosed ошибка при повторном закрытии
var ErrAlreadyClosed = errors.New("registry already closed")

// Closeable интерфейс для ресурсов с методом Close
type Closeable interface {
	Close() error
}

// Registry управляет жизненным циклом ресурсов
type Registry struct {
	mu    sync.Mutex
	items []Closeable
}

// New создаёт новый Registry
func New() *Registry {
	return &Registry{}
}

// Register добавляет ресурс для управления
func (r *Registry) Register(c Closeable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, c)
}

// CloseAll закрывает все зарегистрированные ресурсы
func (r *Registry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for _, item := range r.items {
		if err := item.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

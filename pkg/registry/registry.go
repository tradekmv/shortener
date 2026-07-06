// Package registry provides a lightweight container for managing the
// lifecycle of resources that implement an [io.Closer]-like Close method.
//
// The package is useful when an application owns a set of resources
// (database connections, files, network listeners, etc.) that need to
// be released deterministically on shutdown. Callers register resources
// with a [Registry] and later invoke [Registry.CloseAll] to release them
// all in a single, thread-safe step.
package registry

import (
	"errors"
	"sync"
)

// ErrAlreadyClosed возвращается при попытке повторного закрытия ресурса.
//
// The error is currently reserved for future use; the current
// implementation of [Registry.CloseAll] does not return it.
var ErrAlreadyClosed = errors.New("registry already closed")

// Closeable описывает ресурс, который можно закрыть.
//
// Any type that owns external resources (database handles, files,
// network connections, etc.) and exposes a Close method that returns
// an error satisfies this interface and can be tracked by [Registry].
//
// Implementations of Close should be idempotent: calling Close
// multiple times must not corrupt internal state and should return
// an error (typically [ErrAlreadyClosed]) on subsequent calls.
type Closeable interface {
	Close() error
}

// Registry управляет жизненным циклом набора ресурсов типа [Closeable].
//
// A Registry is safe for concurrent use by multiple goroutines. The
// zero value is not usable; create an instance with [New].
//
// Registry stores resources in the order they were registered and
// closes them in the same order during [Registry.CloseAll], which is
// typically the reverse of the desired shutdown order. Register
// resources accordingly (close last → register first).
type Registry struct {
	mu    sync.Mutex
	items []Closeable
}

// New создаёт новый пустой Registry.
//
// The returned registry contains no resources; use [Registry.Register]
// to add them. New is safe to call from any goroutine.
//
// Example:
//
//	reg := registry.New()
//	reg.Register(db)
//	reg.Register(logFile)
//	defer reg.CloseAll()
func New() *Registry {
	return &Registry{}
}

// Register добавляет ресурс c под управление Registry.
//
// The resource is appended to the internal slice and will be closed
// by [Registry.CloseAll] in the order it was registered. Register is
// safe to call from multiple goroutines thanks to an internal mutex.
//
// A nil c is silently ignored to make registration idempotent in
// optional-resource patterns.
func (r *Registry) Register(c Closeable) {
	if c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, c)
}

// CloseAll закрывает все ранее зарегистрированные ресурсы.
//
// Each registered [Closeable] is closed exactly once. If any Close
// call returns an error, it is collected; only the first encountered
// error is returned to the caller, but every resource is still
// attempted to be closed. This "best-effort" semantics ensures that a
// single faulty resource does not prevent the remaining ones from
// being released.
//
// CloseAll is safe to call from multiple goroutines, but typically it
// is invoked once during process shutdown.
//
// Example:
//
//	reg := registry.New()
//	reg.Register(db)
//	if err := reg.CloseAll(); err != nil {
//	    log.Printf("shutdown error: %v", err)
//	}
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

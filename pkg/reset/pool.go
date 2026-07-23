// Package reset provides a generic Pool with Reset() support using sync.Pool.
package reset

import (
	"reflect"
	"sync"
)

// Resetter — интерфейс для типов с методом Reset()
type Resetter interface {
	Reset()
}

// Pool — пул объектов типа T с поддержкой сброса состояния.
// Реализован на основе sync.Pool: объекты кооперируются с GC,
// а поле New автоматически создаёт объект при пустом пуле.
// Тип T ограничен интерфейсом Resetter.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New creates and returns a Pool.
// The size parameter is accepted for API compatibility but has no effect:
// sync.Pool manages its own internal size adaptively.
func New[T Resetter](size int) *Pool[T] {
	var zero T

	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				// Handle pointer types: new(T) returns **Type for pointer T
				// We need to allocate the underlying type directly
				if reflect.TypeOf(zero).Kind() == reflect.Ptr {
					v := reflect.New(reflect.TypeOf(zero).Elem())
					return v.Interface()
				}
				// Value types: new(T) works correctly
				t := new(T)
				return t
			},
		},
	}
}

// Get returns an object from the pool.
// If the pool is empty, New creates a new object automatically.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put places an object into the pool.
// Reset() is called before putting to clear the state.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}

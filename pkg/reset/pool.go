// Package reset provides a generic Pool with Reset() support using sync.Pool.
package reset

import "sync"

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

// New создаёт и возвращает указатель на Pool с заданным размером.
// Размер влияет на начальную ёмкость внутреннего пула.
func New[T Resetter](size int) *Pool[T] {
	var zero T

	p := &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return new(T)
			},
		},
	}

	// Pre-populate the pool to reach initial size
	for i := 0; i < size; i++ {
		p.Put(zero)
	}

	return p
}

// Get возвращает объект из пула.
// Если пул пуст, New создаёт новый объект автоматически.
func (p *Pool[T]) Get() T {
	obj := p.pool.Get().(T)
	return obj
}

// Put помещает объект в пул.
// Перед помещением вызывается Reset() для сброса состояния.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}

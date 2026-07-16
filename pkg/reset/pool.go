// Package reset provides a generic Pool with Reset() support.
package reset

// Resetter — интерфейс для типов с методом Reset()
type Resetter interface {
	Reset()
}

// Pool — пул объектов типа T с поддержкой сброса состояния.
// Тип T ограничен интерфейсом Resetter.
type Pool[T Resetter] struct {
	pool chan T
}

// New создаёт и возвращает указатель на Pool.
func New[T Resetter](size int) *Pool[T] {
	return &Pool[T]{
		pool: make(chan T, size),
	}
}

// Get возвращает объект из пула.
// Если пул пуст, возвращает zero-значение типа T.
func (p *Pool[T]) Get() T {
	select {
	case obj := <-p.pool:
		return obj
	default:
		var zero T
		return zero
	}
}

// Put помещает объект в пул.
// Перед помещением вызывает Reset() для сброса состояния.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()

	select {
	case p.pool <- obj:
	default:
		// Пул полон, объект отбрасывается
	}
}

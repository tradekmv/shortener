package reset

import "testing"

// TestableStruct — тестовая структура для проверки Pool
type TestableStruct struct {
	value int
	data  string
}

// Reset сбрасывает состояние структуры
func (s *TestableStruct) Reset() {
	s.value = 0
	s.data = ""
}

func TestPool_GetPut(t *testing.T) {
	pool := New[*TestableStruct](2)

	// Get из пустого пула
	obj1 := pool.Get()
	if obj1 != nil {
		t.Errorf("expected nil from empty pool, got %v", obj1)
	}

	// Put в пул
	obj2 := &TestableStruct{value: 42, data: "test"}
	pool.Put(obj2)

	// Get обратно
	obj3 := pool.Get()
	if obj3 == nil {
		t.Fatal("expected object from pool, got nil")
	}
	if obj3.value != 0 || obj3.data != "" {
		t.Errorf("expected Reset() called, got value=%d, data=%s", obj3.value, obj3.data)
	}
}

func TestPool_PutReset(t *testing.T) {
	pool := New[*TestableStruct](1)

	obj := &TestableStruct{value: 100, data: "hello"}
	pool.Put(obj)

	retrieved := pool.Get()
	if retrieved.value != 0 {
		t.Errorf("expected value 0 after Reset, got %d", retrieved.value)
	}
	if retrieved.data != "" {
		t.Errorf("expected empty data after Reset, got %s", retrieved.data)
	}
}

func TestPool_FullPool(t *testing.T) {
	pool := New[*TestableStruct](1)

	// Put two objects - второй должен отброситься (пул полон)
	pool.Put(&TestableStruct{value: 1})
	pool.Put(&TestableStruct{value: 2}) // Отброшен

	// Должен вернуть первый объект (уже сброшенный)
	obj := pool.Get()
	if obj.value != 0 {
		t.Errorf("expected value 0 from pool (reset), got %d", obj.value)
	}
}

func TestPool_PointerShared(t *testing.T) {
	// Тест показывает, что Pool возвращает тот же указатель
	// и Reset() уже вызван (поведение sync.Pool)
	pool := New[*TestableStruct](1)

	obj := &TestableStruct{value: 99}
	pool.Put(obj)

	// При Put Reset() уже вызван
	if obj.value != 0 {
		t.Errorf("expected Reset called on Put, got value=%d", obj.value)
	}
}

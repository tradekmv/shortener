package reset

import (
	"testing"
)

// TestableStruct — тестовая структура для проверки Pool
type TestableStruct struct {
	value int
	data  string
}

func (s *TestableStruct) Reset() {
	s.value = 0
	s.data = ""
}

// TestPool_GetNeverNil — sync.Pool.Get не возвращает nil
func TestPool_GetNeverNil(t *testing.T) {
	pool := New[*TestableStruct](1)

	obj := pool.Get()
	if obj == nil {
		t.Error("sync.Pool.Get() should never return nil")
	}
}

// TestPool_PutReset — Put вызывает Reset перед помещением в пул
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

// TestPool_Reuse — объект переиспользуется после Put/Get
func TestPool_Reuse(t *testing.T) {
	pool := New[*TestableStruct](1)

	obj := pool.Get()
	obj.value = 42
	obj.data = "test"

	pool.Put(obj)

	retrieved := pool.Get()
	if retrieved.value != 0 || retrieved.data != "" {
		t.Errorf("expected Reset applied, got value=%d, data=%s", retrieved.value, retrieved.data)
	}
}

// TestPool_ValueFromNew — при пустом пуле New создаёт объект через sync.Pool.New
func TestPool_ValueFromNew(t *testing.T) {
	pool := New[*TestableStruct](0)

	// Первый Get из пустого пула должен создать объект через New
	obj := pool.Get()
	if obj == nil {
		t.Fatal("Get should never return nil from sync.Pool with New")
	}

	// Изменяем и кладём обратно
	obj.value = 999
	pool.Put(obj)

	// Следующий Get должен вернуть объект сброшенным
	retrieved := pool.Get()
	if retrieved.value != 0 {
		t.Errorf("expected Reset called, got value=%d", retrieved.value)
	}
}

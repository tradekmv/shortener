package storage

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

const benchDSN = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"

func newBenchPostgres(b *testing.B) *PostgresStorage {
	s, err := NewPostgres(benchDSN)
	if err != nil {
		b.Skipf("postgres недоступен: %v", err)
	}
	return s
}

// benchRunID возвращает уникальный ID для запуска бенчмарка (UnixNano)
// чтобы избежать конфликтов уникального индекса original_url между запусками
var benchRunID = time.Now().UnixNano()
var benchRunMu sync.Mutex
var benchRunCounter int

// truncatePostgres очищает таблицу urls перед бенчмарком
func truncatePostgres(b *testing.B, s *PostgresStorage) {
	if _, err := s.db.Exec("TRUNCATE TABLE urls"); err != nil {
		b.Logf("не удалось очистить таблицу (возможно, миграция ещё не прошла): %v", err)
	}
}

func benchURL(i int) string {
	return fmt.Sprintf("https://example.com/pg/%d/%d", benchRunID, i)
}

func benchNextID() string {
	benchRunMu.Lock()
	defer benchRunMu.Unlock()
	benchRunCounter++
	return "bench" + strconv.FormatInt(benchRunID, 10) + strconv.Itoa(benchRunCounter)
}

func BenchmarkPostgresStorage_Save(b *testing.B) {
	s := newBenchPostgres(b)
	defer s.Close()
	truncatePostgres(b, s)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, benchNextID(), benchURL(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresStorage_Get(b *testing.B) {
	s := newBenchPostgres(b)
	defer s.Close()
	truncatePostgres(b, s)
	ctx := context.Background()
	const n = 1000
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id := benchNextID()
		_ = s.Save(ctx, id, benchURL(i))
		ids[i] = id
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Get(ctx, ids[i%n])
	}
}

func BenchmarkMemoryStorage_Save(b *testing.B) {
	s := NewMemory()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, benchNextID(), benchURL(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorage_Get(b *testing.B) {
	s := NewMemory()
	ctx := context.Background()
	const n = 1000
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id := benchNextID()
		_ = s.Save(ctx, id, benchURL(i))
		ids[i] = id
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Get(ctx, ids[i%n])
	}
}

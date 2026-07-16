package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/tradekmv/shortener.git/internal/repository/storage"
)

func BenchmarkGenerateID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, err := generateID(length)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSave(b *testing.B) {
	svc := NewService(storage.NewMemory())
	ctx := context.Background()
	i := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := svc.Save(ctx, "https://example.com/"+strconv.Itoa(i))
		if err != nil {
			b.Fatal(err)
		}
		i++
	}
}

func BenchmarkSaveBatch(b *testing.B) {
	svc := NewService(storage.NewMemory())
	ctx := context.Background()
	urls := make([]storage.URLRecord, 100)
	for i := range urls {
		urls[i] = storage.URLRecord{OriginalURL: "https://example.com/" + strconv.Itoa(i)}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := svc.SaveBatch(ctx, urls)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	svc := NewService(storage.NewMemory())
	ctx := context.Background()
	const n = 1000
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id, _ := svc.Save(ctx, "https://example.com/"+strconv.Itoa(i))
		ids[i] = id
	}
	i := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.Get(ctx, ids[i%n])
		i++
	}
}

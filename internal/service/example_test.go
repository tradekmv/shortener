package service_test

import (
	"context"
	"fmt"

	"github.com/tradekmv/shortener.git/internal/repository/storage"
	"github.com/tradekmv/shortener.git/internal/service"
)

// ExampleIsURL демонстрирует валидацию URL.
func ExampleIsURL() {
	fmt.Println(service.IsURL("https://example.com"))
	fmt.Println(service.IsURL("http://example.com"))
	fmt.Println(service.IsURL("ftp://example.com"))
	fmt.Println(service.IsURL("not-a-url"))
	// Output:
	// true
	// true
	// false
	// false
}

// ExampleService_Save демонстрирует сокращение URL.
func ExampleService_Save() {
	store := storage.NewMemory()
	svc := service.NewService(store)
	ctx := context.Background()

	id, err := svc.Save(ctx, "https://example.com/page")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(id) > 0)
	// Output:
	// true
}

// ExampleService_Get демонстрирует получение оригинального URL.
func ExampleService_Get() {
	store := storage.NewMemory()
	svc := service.NewService(store)
	ctx := context.Background()

	id, _ := svc.Save(ctx, "https://example.com/page")
	got, err := svc.Get(ctx, id)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(got)
	// Output:
	// https://example.com/page
}

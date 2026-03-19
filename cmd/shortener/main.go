package main

import (
	"net/http"

	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
)

func main() {
	store := storage.New()
	h := handler.New(store, "http://localhost:8080")
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, h.PostHandler)
	mux.HandleFunc(`/{id}`, h.GetHandler)

	err := http.ListenAndServe(`localhost:8080`, mux)
	if err != nil {
		panic(err)
	}
}

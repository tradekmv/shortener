package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tradekmv/shortener.git/internal/handler"
	"github.com/tradekmv/shortener.git/internal/repository/storage"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	store := storage.New()
	h := handler.New(store, "http://localhost:8080")

	r.Post("/", h.PostHandler)
	r.Get("/{id}", h.GetHandler)

	err := http.ListenAndServe("localhost:8080", r)
	if err != nil {
		panic(err)
	}
}

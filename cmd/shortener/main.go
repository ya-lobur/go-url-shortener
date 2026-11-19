package main

import (
	"log"
	"net/http"

	"github.com/ya-lobur/go-url-shortener/internal/handler"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	storage := repository.NewMemoryStorage()
	return http.ListenAndServe("localhost:8080", handler.New(storage))
}

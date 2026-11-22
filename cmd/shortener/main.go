package main

import (
	"log"
	"net/http"

	"github.com/ya-lobur/go-url-shortener/internal/config"
	"github.com/ya-lobur/go-url-shortener/internal/handler"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
)

func main() {
	cfg := config.ParseFlags()
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config.Config) error {
	storage := repository.NewMemoryStorage()
	return http.ListenAndServe(cfg.RunAddr, handler.New(storage, cfg.BaseURL))
}

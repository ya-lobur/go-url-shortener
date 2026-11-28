package main

import (
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/ya-lobur/go-url-shortener/internal/config"
	"github.com/ya-lobur/go-url-shortener/internal/handler"
	"github.com/ya-lobur/go-url-shortener/internal/logger"
	"github.com/ya-lobur/go-url-shortener/internal/middleware"
	"github.com/ya-lobur/go-url-shortener/internal/repository"
	"github.com/ya-lobur/go-url-shortener/internal/service"
)

func main() {
	cfg := config.Parse()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	log, err := logger.New()
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer log.Sync()

	storage := repository.NewMemoryStorage()
	router := handler.New(service.New(storage), cfg.Handler, log)
	log.Info("Starting server", zap.String("address", cfg.HTTP.RunAddr))
	return http.ListenAndServe(cfg.HTTP.RunAddr, middleware.RequestLogger(log)(router))
}

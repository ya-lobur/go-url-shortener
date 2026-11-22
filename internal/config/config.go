package config

import "flag"

// Config contains the HTTP server settings.
type Config struct {
	RunAddr string
	BaseURL string
}

// ParseFlags reads the application settings from command-line flags.
func ParseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.RunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.Parse()
	return cfg
}

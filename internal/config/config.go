package config

import (
	"flag"
	"os"
)

// Config contains the application settings.
type Config struct {
	HTTP    HTTPConfig
	Handler HandlerConfig
}

type HTTPConfig struct{ RunAddr string }
type HandlerConfig struct{ BaseURL string }

// Parse reads flags, then overrides them with non-empty environment values.
func Parse() Config {
	var cfg Config
	flag.StringVar(&cfg.HTTP.RunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.Handler.BaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.Parse()

	if runAddr := os.Getenv("SERVER_ADDRESS"); runAddr != "" {
		cfg.HTTP.RunAddr = runAddr
	}
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		cfg.Handler.BaseURL = baseURL
	}

	return cfg
}

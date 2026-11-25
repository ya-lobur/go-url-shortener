package config

import "flag"

// Config contains the application settings.
type Config struct {
	HTTP    HTTPConfig
	Handler HandlerConfig
}

type HTTPConfig struct{ RunAddr string }
type HandlerConfig struct{ BaseURL string }

// ParseFlags reads the application settings from command-line flags.
func ParseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.HTTP.RunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.Handler.BaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.Parse()
	return cfg
}

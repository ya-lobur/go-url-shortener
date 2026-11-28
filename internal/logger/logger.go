package logger

import "go.uber.org/zap"

// New creates an Info-level JSON logger without sampling.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	// Every request must be logged, including repeated messages under load.
	cfg.Sampling = nil
	return cfg.Build()
}

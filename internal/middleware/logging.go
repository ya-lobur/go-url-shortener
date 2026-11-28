package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}

	n, err := w.ResponseWriter.Write(body)
	w.size += n
	return n, err
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.ResponseWriter.WriteHeader(status)
	// Informational responses may precede the final response.
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		return
	}
	w.status = status
	w.wroteHeader = true
}

// RequestLogger records each completed request and its response at Info level.
func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			uri, method := r.RequestURI, r.Method
			writer := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(writer, r)

			logger.Info("HTTP request",
				zap.String("uri", uri),
				zap.String("method", method),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", writer.status),
				zap.Int("size", writer.size),
			)
		})
	}
}

package apiserver

import (
	"log/slog"
	"net/http"
	"time"
)

type responseStatusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseStatusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newAccessLogHandler(log *slog.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			recorder := &responseStatusRecorder{ResponseWriter: w}

			start := time.Now()
			h.ServeHTTP(recorder, req)
			log.Info("api access",
				"status", recorder.status,
				"method", req.Method,
				"path", req.URL.Path,
				"duration", time.Since(start),
			)
		})
	}
}

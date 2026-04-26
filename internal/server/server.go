package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/mizanproxy/mizan/internal/api"
	"github.com/mizanproxy/mizan/internal/observe"
	"github.com/mizanproxy/mizan/internal/store"
)

type Config struct {
	Bind string
}

func New(cfg Config, st *store.Store, log *slog.Logger) *http.Server {
	if cfg.Bind == "" {
		cfg.Bind = "127.0.0.1:7890"
	}
	mux := http.NewServeMux()
	api.Register(mux, st)
	mux.Handle("/", embeddedUI())

	return &http.Server{
		Addr:              cfg.Bind,
		Handler:           recoverer(logger(log, mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusRecorder) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusRecorder) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func logger(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.statusCode()
		observe.RecordHTTPRequest(r.Method, routePattern(r), status)
		log.Info("http_request", "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(start).Milliseconds())
	})
}

func routePattern(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

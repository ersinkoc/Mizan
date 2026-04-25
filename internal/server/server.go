package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/mizanproxy/mizan/internal/api"
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

func logger(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
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

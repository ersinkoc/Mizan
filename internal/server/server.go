package server

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mizanproxy/mizan/internal/api"
	"github.com/mizanproxy/mizan/internal/observe"
	"github.com/mizanproxy/mizan/internal/store"
)

type Config struct {
	Bind string
	Auth AuthConfig
}

type AuthConfig struct {
	Token         string
	BasicUser     string
	BasicPassword string
}

func (cfg AuthConfig) Enabled() bool {
	return cfg.Token != "" || (cfg.BasicUser != "" && cfg.BasicPassword != "")
}

func ParseBasicCredential(value string) (string, string, error) {
	user, password, ok := strings.Cut(value, ":")
	if !ok || user == "" || password == "" {
		return "", "", fmt.Errorf("basic auth credential must use user:password")
	}
	return user, password, nil
}

func RequiresAuth(bind string) bool {
	host, _, err := net.SplitHostPort(bind)
	if err != nil {
		host = bind
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}
	return !ip.IsLoopback()
}

func New(cfg Config, st *store.Store, log *slog.Logger) *http.Server {
	if cfg.Bind == "" {
		cfg.Bind = "127.0.0.1:7890"
	}
	mux := http.NewServeMux()
	api.Register(mux, st)
	mux.Handle("/", embeddedUI())

	handler := http.Handler(mux)
	if cfg.Auth.Enabled() {
		handler = authenticator(cfg.Auth, handler)
	}
	return &http.Server{
		Addr:              cfg.Bind,
		Handler:           recoverer(logger(log, handler)),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func authenticator(cfg AuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authPublicPath(r.URL.Path) || cfg.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.BasicUser != "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Mizan"`)
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func authPublicPath(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

func (cfg AuthConfig) authorized(r *http.Request) bool {
	if cfg.Token != "" {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if ok && constantTimeEqual(token, cfg.Token) {
			return true
		}
	}
	if cfg.BasicUser != "" && cfg.BasicPassword != "" {
		user, password, ok := r.BasicAuth()
		return ok && constantTimeEqual(user, cfg.BasicUser) && constantTimeEqual(password, cfg.BasicPassword)
	}
	return false
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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

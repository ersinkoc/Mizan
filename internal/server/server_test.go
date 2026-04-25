package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mizanproxy/mizan/internal/store"
)

func TestServerRoutesAndSPA(t *testing.T) {
	st := store.New(t.TempDir())
	srv := New(Config{}, st, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	srv.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("health status=%d", res.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/some/spa/path", nil)
	res = httptest.NewRecorder()
	srv.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("spa status=%d", res.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/nope", nil)
	res = httptest.NewRecorder()
	srv.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("api fallback status=%d", res.Code)
	}
}

func TestRecoverer(t *testing.T) {
	handler := recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", res.Code)
	}
}

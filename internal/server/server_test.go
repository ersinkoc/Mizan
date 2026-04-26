package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res = httptest.NewRecorder()
	srv.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("metrics status=%d", res.Code)
	}
	if body := res.Body.String(); !strings.Contains(body, `mizan_http_requests_total{method="GET",route="GET /healthz",status="200"} 1`) {
		t.Fatalf("metrics missing request count:\n%s", body)
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

func TestStatusRecorder(t *testing.T) {
	blank := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	if blank.statusCode() != http.StatusOK {
		t.Fatalf("blank status=%d", blank.statusCode())
	}
	blank.Flush()
	if blank.statusCode() != http.StatusOK {
		t.Fatalf("flushed blank status=%d", blank.statusCode())
	}

	res := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: res}
	rec.WriteHeader(http.StatusAccepted)
	rec.WriteHeader(http.StatusTeapot)
	if rec.statusCode() != http.StatusAccepted || res.Code != http.StatusAccepted {
		t.Fatalf("status=%d recorder=%d", rec.statusCode(), res.Code)
	}
	if rec.Unwrap() != res {
		t.Fatal("unexpected wrapped response writer")
	}
	rec.Flush()
	if !res.Flushed {
		t.Fatal("response was not flushed")
	}
}

func TestStatusRecorderWriteAndRoutePattern(t *testing.T) {
	res := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: res}
	if _, err := rec.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if rec.statusCode() != http.StatusOK {
		t.Fatalf("status=%d", rec.statusCode())
	}
	req := httptest.NewRequest(http.MethodGet, "/unmatched", nil)
	if got := routePattern(req); got != "unmatched" {
		t.Fatalf("route=%q", got)
	}
	req.Pattern = "GET /known"
	if got := routePattern(req); got != "GET /known" {
		t.Fatalf("route=%q", got)
	}
}

func TestEmbeddedUIRootAndDefaultLogger(t *testing.T) {
	handler := logger(nil, embeddedUI())
	for _, path := range []string{"/", ""} {
		req := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("root path %q status=%d", path, res.Code)
		}
		_, _ = io.Copy(io.Discard, res.Body)
	}
}

func TestEmbeddedUIMissingDist(t *testing.T) {
	handler := embeddedUIFromSub(nil, errEmbeddedUIMissing)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", res.Code)
	}
}

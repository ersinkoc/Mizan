package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

const nginxStubStatus = `Active connections: 291
server accepts handled requests
 16630948 16630948 31070465
Reading: 6 Writing: 179 Waiting: 106
`

func TestParseNginxStubStatus(t *testing.T) {
	stats, err := ParseNginxStubStatus("\n" + nginxStubStatus + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Active != 291 || stats.Accepts != 16630948 || stats.Requests != 31070465 || stats.Writing != 179 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestParseNginxStubStatusErrors(t *testing.T) {
	for _, payload := range []string{
		"",
		"Active conns: 1\nserver accepts handled requests\n1 1 1\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: x\nserver accepts handled requests\n1 1 1\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 1\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\nx 1 1\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 x 1\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 1 x\nReading: 0 Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 1 1\nReading: x Writing: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 1 1\nReading: 0 Waiting: 1\n",
		"Active connections: 1\nserver accepts handled requests\n1 1 1\nReading: 0 Writing: 0\n",
	} {
		if _, err := ParseNginxStubStatus(payload); err == nil {
			t.Fatalf("expected parse error for %q", payload)
		}
	}
}

func TestSummarizeNginx(t *testing.T) {
	status, message := summarizeNginx(NginxStubStatus{Active: 2, Accepts: 10, Handled: 10, Requests: 20, Reading: 1, Writing: 1})
	if status != "healthy" || !strings.Contains(message, "active=2") {
		t.Fatalf("status=%q message=%q", status, message)
	}
	status, message = summarizeNginx(NginxStubStatus{Accepts: 10, Handled: 8})
	if status != "warning" || !strings.Contains(message, "2 accepted") {
		t.Fatalf("status=%q message=%q", status, message)
	}
}

func TestCollectNginx(t *testing.T) {
	target := store.Target{ID: "t_1", Name: "edge", Host: "host", Engine: ir.EngineNginx}
	snapshot, err := CollectNginx(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "unknown" || snapshot.Message == "" {
		t.Fatalf("snapshot=%+v", snapshot)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(nginxStubStatus))
	}))
	defer server.Close()
	target.MonitorEndpoint = server.URL
	snapshot, err = CollectNginx(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "healthy" || !strings.Contains(snapshot.Message, "requests=31070465") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestCollectNginxFailures(t *testing.T) {
	target := store.Target{ID: "t_1", Name: "edge", Host: "host", Engine: ir.EngineNginx, MonitorEndpoint: "http://example.invalid"}
	oldFetchURL := fetchURL
	t.Cleanup(func() { fetchURL = oldFetchURL })
	fetchURL = func(context.Context, string) ([]byte, error) {
		return nil, errors.New("fetch failed")
	}
	snapshot, err := CollectNginx(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "failed" || snapshot.Message != "fetch failed" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	fetchURL = func(context.Context, string) ([]byte, error) {
		return []byte("bad"), nil
	}
	snapshot, err = CollectNginx(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "failed" || !strings.Contains(snapshot.Message, "incomplete") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

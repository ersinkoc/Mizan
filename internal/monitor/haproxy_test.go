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

const haproxyUpCSV = `# pxname,svname,scur,status
be_app,app1,3,UP
be_app,app2,4,UP
be_app,BACKEND,7,UP
fe_web,FRONTEND,7,OPEN
`

func TestParseHAProxyStats(t *testing.T) {
	stats, err := ParseHAProxyStats(haproxyUpCSV)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Servers) != 2 || stats.Servers[0].ProxyName != "be_app" || stats.Servers[0].Current != "3" {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestParseHAProxyStatsErrors(t *testing.T) {
	if _, err := ParseHAProxyStats("be_app,app1,UP\n"); err == nil {
		t.Fatal("expected missing header error")
	}
	if _, err := ParseHAProxyStats(""); err == nil {
		t.Fatal("expected empty stats error")
	}
	if _, err := ParseHAProxyStats("# pxname,svname,status\n\"bad"); err == nil {
		t.Fatal("expected csv error")
	}
}

func TestSummarizeHAProxy(t *testing.T) {
	for _, tc := range []struct {
		stats  HAProxyStats
		status string
		text   string
	}{
		{HAProxyStats{}, "unknown", "did not include"},
		{HAProxyStats{Servers: []HAProxyServerStat{{Status: "UP"}}}, "healthy", "1 HAProxy servers are up"},
		{HAProxyStats{Servers: []HAProxyServerStat{{Status: "MAINT"}}}, "warning", "need attention"},
		{HAProxyStats{Servers: []HAProxyServerStat{{Status: "DOWN"}}}, "failed", "down"},
	} {
		status, message := summarizeHAProxy(tc.stats)
		if status != tc.status || !strings.Contains(message, tc.text) {
			t.Fatalf("status=%q message=%q", status, message)
		}
	}
}

func TestCollectHAProxy(t *testing.T) {
	target := store.Target{ID: "t_1", Name: "edge", Host: "host", Engine: ir.EngineHAProxy}
	snapshot, err := CollectHAProxy(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "unknown" || snapshot.Message == "" {
		t.Fatalf("snapshot=%+v", snapshot)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(haproxyUpCSV))
	}))
	defer server.Close()
	target.MonitorEndpoint = server.URL
	snapshot, err = CollectHAProxy(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "healthy" || !strings.Contains(snapshot.Message, "2 HAProxy servers are up") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestCollectHAProxyFailures(t *testing.T) {
	target := store.Target{ID: "t_1", Name: "edge", Host: "host", Engine: ir.EngineHAProxy}
	oldFetchURL := fetchURL
	t.Cleanup(func() { fetchURL = oldFetchURL })
	fetchURL = func(context.Context, string) ([]byte, error) {
		return nil, errors.New("fetch failed")
	}
	target.MonitorEndpoint = "http://example.invalid"
	snapshot, err := CollectHAProxy(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "failed" || snapshot.Message != "fetch failed" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	fetchURL = func(context.Context, string) ([]byte, error) {
		return []byte("bad,row\n"), nil
	}
	snapshot, err = CollectHAProxy(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != "failed" || !strings.Contains(snapshot.Message, "header") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestFetchURLBranches(t *testing.T) {
	if _, err := fetchURL(t.Context(), "://bad"); err == nil {
		t.Fatal("expected request error")
	}
	if _, err := fetchURL(t.Context(), "http://127.0.0.1:1"); err == nil {
		t.Fatal("expected connection error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	if _, err := fetchURL(t.Context(), server.URL+"/fail"); err == nil {
		t.Fatal("expected status error")
	}
	data, err := fetchURL(t.Context(), server.URL+"/ok")
	if err != nil || string(data) != "ok" {
		t.Fatalf("data=%q err=%v", string(data), err)
	}
}

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

func TestProjectLifecycleEndpoints(t *testing.T) {
	st := store.New(t.TempDir())
	mux := http.NewServeMux()
	Register(mux, st)

	res := doJSON(mux, http.MethodGet, "/healthz", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("health status=%d", res.Code)
	}
	res = doJSON(mux, http.MethodGet, "/version", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("version status=%d", res.Code)
	}
	res = doJSON(mux, http.MethodPost, "/api/v1/projects", map[string]any{"name": "edge", "engines": []string{"haproxy", "nginx"}})
	if res.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
		IR      map[string]any `json:"ir"`
		Version string         `json:"version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.Project.ID
	version := created.Version

	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/v1/projects", http.StatusOK},
		{http.MethodGet, "/api/v1/projects/" + id, http.StatusOK},
		{http.MethodGet, "/api/v1/projects/" + id + "/ir", http.StatusOK},
		{http.MethodGet, "/api/v1/projects/" + id + "/ir/snapshots", http.StatusOK},
		{http.MethodGet, "/api/v1/projects/" + id + "/ir/tags", http.StatusOK},
		{http.MethodGet, "/api/v1/projects/" + id + "/audit", http.StatusOK},
	} {
		res = doJSON(mux, tc.method, tc.path, nil)
		if res.Code != tc.status {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, res.Code, res.Body.String())
		}
	}

	model, _, err := st.GetIR(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	model.Frontends = append(model.Frontends, ir.Frontend{ID: "fe_web", Name: "web", Bind: ":80", Protocol: "http"})
	model.Backends = append(model.Backends, ir.Backend{ID: "be_app", Name: "app", Algorithm: "roundrobin", Servers: []string{}})
	res = doJSONWithHeader(mux, http.MethodPatch, "/api/v1/projects/"+id+"/ir", map[string]any{"ir": model}, map[string]string{"If-Match": version})
	if res.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", res.Code, res.Body.String())
	}
	var patched struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}

	res = doJSON(mux, http.MethodPost, "/api/v1/projects/"+id+"/generate", map[string]string{"target": "nginx"})
	if res.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", res.Code, res.Body.String())
	}
	res = doJSON(mux, http.MethodPost, "/api/v1/projects/"+id+"/validate", map[string]string{"target": "haproxy"})
	if res.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", res.Code, res.Body.String())
	}

	res = doJSON(mux, http.MethodGet, "/api/v1/projects/"+id+"/ir/snapshots", nil)
	var snapshots []string
	if err := json.Unmarshal(res.Body.Bytes(), &snapshots); err != nil {
		t.Fatal(err)
	}
	if len(snapshots) < 2 {
		t.Fatalf("snapshots=%v", snapshots)
	}
	res = doJSON(mux, http.MethodGet, "/api/v1/projects/"+id+"/ir/snapshots/"+snapshots[0], nil)
	if res.Code != http.StatusOK {
		t.Fatalf("get snapshot status=%d body=%s", res.Code, res.Body.String())
	}
	res = doJSON(mux, http.MethodPost, "/api/v1/projects/"+id+"/ir/tag", map[string]string{"snapshot_ref": snapshots[0], "label": "latest"})
	if res.Code != http.StatusCreated {
		t.Fatalf("tag status=%d body=%s", res.Code, res.Body.String())
	}
	res = doJSON(mux, http.MethodPost, "/api/v1/projects/"+id+"/ir/diff", map[string]string{"from_hash": snapshots[len(snapshots)-1], "to_hash": snapshots[0]})
	if res.Code != http.StatusOK {
		t.Fatalf("diff status=%d body=%s", res.Code, res.Body.String())
	}
	res = doJSONWithHeader(mux, http.MethodPost, "/api/v1/projects/"+id+"/ir/revert", map[string]string{"snapshot_ref": "latest"}, map[string]string{"If-Match": patched.Version})
	if res.Code != http.StatusOK {
		t.Fatalf("revert status=%d body=%s", res.Code, res.Body.String())
	}
	res = doJSON(mux, http.MethodDelete, "/api/v1/projects/"+id, nil)
	if res.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestImportAndGenerate(t *testing.T) {
	st := store.New(t.TempDir())
	mux := http.NewServeMux()
	Register(mux, st)
	body := map[string]string{
		"name":     "imported",
		"filename": "haproxy.cfg",
		"config": `
frontend web
  bind :80
  default_backend be_app
backend be_app
  server app1 127.0.0.1:8080
`,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/import", bytes.NewReader(raw))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	genBody := []byte(`{"target":"haproxy"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+created.Project.ID+"/generate", bytes.NewReader(genBody))
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("backend be_app")) {
		t.Fatalf("generated config missing backend: %s", res.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+created.Project.ID+"/audit", nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("project.import")) || !bytes.Contains(res.Body.Bytes(), []byte("config.generate")) {
		t.Fatalf("audit missing expected actions: %s", res.Body.String())
	}
}

func TestAPIErrorBranches(t *testing.T) {
	st := store.New(t.TempDir())
	mux := http.NewServeMux()
	Register(mux, st)

	for _, tc := range []struct {
		method string
		path   string
		body   any
		status int
	}{
		{http.MethodPost, "/api/v1/projects", "{bad", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/projects", map[string]any{"name": ""}, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/projects/import", map[string]any{"filename": "x.cfg"}, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/projects/import", map[string]any{"filename": "x.txt", "config": "not config"}, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/projects/missing", nil, http.StatusNotFound},
		{http.MethodGet, "/api/v1/projects/missing/ir", nil, http.StatusNotFound},
		{http.MethodGet, "/api/v1/projects/missing/ir/snapshots", nil, http.StatusOK},
		{http.MethodGet, "/api/v1/projects/missing/ir/snapshots/nope", nil, http.StatusNotFound},
		{http.MethodPost, "/api/v1/projects/missing/generate", map[string]string{"target": "bad"}, http.StatusNotFound},
		{http.MethodPost, "/api/v1/projects/missing/validate", map[string]string{"target": "bad"}, http.StatusNotFound},
	} {
		res := doPossiblyRawJSON(mux, tc.method, tc.path, tc.body, nil)
		if res.Code != tc.status {
			t.Fatalf("%s %s status=%d want=%d body=%s", tc.method, tc.path, res.Code, tc.status, res.Body.String())
		}
	}

	res := doJSON(mux, http.MethodPost, "/api/v1/projects", map[string]any{"name": "edge", "engines": []string{"haproxy"}})
	if res.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.Project.ID
	for _, tc := range []struct {
		path   string
		body   any
		status int
	}{
		{"/api/v1/projects/" + id + "/ir", "{bad", http.StatusBadRequest},
		{"/api/v1/projects/" + id + "/ir", map[string]any{}, http.StatusBadRequest},
		{"/api/v1/projects/" + id + "/ir/revert", map[string]any{}, http.StatusBadRequest},
		{"/api/v1/projects/" + id + "/ir/diff", map[string]any{"from_hash": "missing", "to_hash": "missing"}, http.StatusNotFound},
		{"/api/v1/projects/" + id + "/ir/tag", map[string]any{"snapshot_ref": "missing", "label": "bad"}, http.StatusBadRequest},
		{"/api/v1/projects/" + id + "/generate", map[string]string{"target": "bad"}, http.StatusBadRequest},
		{"/api/v1/projects/" + id + "/validate", map[string]string{"target": "bad"}, http.StatusBadRequest},
	} {
		res := doPossiblyRawJSON(mux, http.MethodPost, tc.path, tc.body, nil)
		if strings.HasSuffix(tc.path, "/ir") {
			res = doPossiblyRawJSON(mux, http.MethodPatch, tc.path, tc.body, map[string]string{"If-Match": created.Version})
		}
		if res.Code != tc.status {
			t.Fatalf("%s status=%d want=%d body=%s", tc.path, res.Code, tc.status, res.Body.String())
		}
	}
}

func TestAPIHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if actorFromRequest(req) != "local" {
		t.Fatal("expected local actor")
	}
	req.Header.Set("X-Mizan-Actor", "alice")
	if actorFromRequest(req) != "alice" {
		t.Fatal("expected header actor")
	}
	if truncate("abc", 10) != "abc" || truncate("abcdef", 3) != "abc" {
		t.Fatal("truncate unexpected")
	}
	if hasErrors([]ir.Issue{{Severity: ir.SeverityWarning}}) {
		t.Fatal("warnings should not be errors")
	}
}

func doJSON(mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	return doJSONWithHeader(mux, method, path, body, nil)
}

func doJSONWithHeader(mux http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	return doPossiblyRawJSON(mux, method, path, body, headers)
}

func doPossiblyRawJSON(mux http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else if raw, ok := body.(string); ok {
		reader = bytes.NewReader([]byte(raw))
	} else {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	return res
}

package ir

import "testing"

func TestLintCatchesDanglingBackendAndMissingTLS(t *testing.T) {
	model := EmptyModel("p_1", "test", "", []Engine{EngineHAProxy})
	model.Frontends = append(model.Frontends, Frontend{
		ID:             "fe_web",
		Name:           "web",
		Bind:           ":443",
		Protocol:       "http",
		DefaultBackend: "missing",
	})
	issues := Lint(model)
	if len(issues) < 2 {
		t.Fatalf("expected at least two issues, got %+v", issues)
	}
	var tls, backend bool
	for _, issue := range issues {
		if issue.Field == "tls_id" {
			tls = true
		}
		if issue.Field == "default_backend" {
			backend = true
		}
	}
	if !tls || !backend {
		t.Fatalf("missing expected issues: %+v", issues)
	}
}

func TestHashStable(t *testing.T) {
	model := EmptyModel("p_1", "test", "", []Engine{EngineHAProxy})
	a, err := Hash(model)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Hash(model)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("hash not stable: %s != %s", a, b)
	}
}

func TestLintBroaderBranches(t *testing.T) {
	model := EmptyModel("p_1", "edge", "", []Engine{EngineHAProxy, EngineNginx})
	model.Frontends = []Frontend{
		{ID: "fe1", Name: "one", Bind: "127.0.0.1:443", TLSID: "missing", DefaultBackend: "be_missing", Rules: []string{"missing_rule"}},
		{ID: "fe2", Name: "two", Bind: "127.0.0.1:443"},
	}
	model.Backends = []Backend{
		{ID: "be1", Name: "dup", HealthCheckID: "hc_missing", Servers: []string{"missing_server"}},
		{ID: "be2", Name: "dup", Servers: []string{}},
	}
	model.Servers = []Server{{ID: "srv_bad", Address: "", Port: 70000}}
	model.Rules = []Rule{{ID: "rule_bad", Action: RuleAction{Type: "use_backend", BackendID: "missing"}}}
	model.TLSProfiles = []TLSProfile{{ID: "tls_bad"}}
	model.Caches = []CachePolicy{{ID: "cache"}}
	issues := Lint(model)
	fields := map[string]bool{}
	for _, issue := range issues {
		fields[issue.Field] = true
	}
	for _, field := range []string{"bind", "tls_id", "default_backend", "rules", "health_check_id", "servers", "address", "port", "cert_path", "action.backend_id", "name"} {
		if !fields[field] {
			t.Fatalf("expected field %s in issues: %+v", field, issues)
		}
	}
}

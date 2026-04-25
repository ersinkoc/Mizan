package ir

import (
	"encoding/json"
	"testing"
)

func TestApplyEntityMutations(t *testing.T) {
	model := EmptyModel("p_1", "edge", "", []Engine{EngineHAProxy})
	frontend := Frontend{ID: "fe_web", Name: "web", Bind: ":80", Protocol: "http"}
	backend := Backend{ID: "be_app", Name: "app", Algorithm: "roundrobin", Servers: []string{}}
	server := Server{ID: "s1", Address: "127.0.0.1", Port: 8080, Weight: 100}
	rule := Rule{ID: "r1", Predicate: Predicate{Type: "path_prefix", Value: "/api/"}, Action: RuleAction{Type: "use_backend", BackendID: "be_app"}}

	var err error
	model, err = Apply(model, mutation("frontend.create", "", "", "", frontend))
	if err != nil {
		t.Fatal(err)
	}
	model, err = Apply(model, mutation("backend.create", "", "", "", backend))
	if err != nil {
		t.Fatal(err)
	}
	model, err = Apply(model, mutation("server.create", "", "be_app", "", server))
	if err != nil {
		t.Fatal(err)
	}
	model, err = Apply(model, mutation("rule.create", "", "", "fe_web", rule))
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Frontends) != 1 || len(model.Backends) != 1 || len(model.Servers) != 1 || len(model.Rules) != 1 {
		t.Fatalf("unexpected model counts: %+v", model)
	}
	if model.Backends[0].Servers[0] != "s1" || model.Frontends[0].Rules[0] != "r1" {
		t.Fatalf("references were not attached: %+v", model)
	}

	model, err = Apply(model, mutation("frontend.update", "fe_web", "", "", map[string]any{"bind": ":8080"}))
	if err != nil {
		t.Fatal(err)
	}
	if model.Frontends[0].Bind != ":8080" {
		t.Fatalf("frontend not updated: %+v", model.Frontends[0])
	}

	model, err = Apply(model, Mutation{Type: "view.move", EntityID: "be_app", X: 50, Y: 75})
	if err != nil {
		t.Fatal(err)
	}
	if model.Backends[0].View.X != 50 || model.Backends[0].View.Y != 75 {
		t.Fatalf("backend view not moved: %+v", model.Backends[0].View)
	}
	model, err = Apply(model, Mutation{Type: "view.move", EntityID: "fe_web", X: 15, Y: 25})
	if err != nil {
		t.Fatal(err)
	}
	model, err = Apply(model, Mutation{Type: "view.move", EntityID: "r1", X: 35, Y: 45})
	if err != nil {
		t.Fatal(err)
	}
	if model.Frontends[0].View.X != 15 || model.Rules[0].View.X != 35 {
		t.Fatalf("frontend/rule view not moved: %+v %+v", model.Frontends[0].View, model.Rules[0].View)
	}

	model, err = Apply(model, Mutation{Type: "view.zoom", Zoom: 1.5})
	if err != nil {
		t.Fatal(err)
	}
	if model.View.Zoom != 1.5 {
		t.Fatalf("zoom not updated: %+v", model.View)
	}
}

func TestApplyDeletesAndErrors(t *testing.T) {
	model := EmptyModel("p_1", "edge", "", []Engine{EngineHAProxy})
	model.Frontends = []Frontend{{ID: "fe_web", Rules: []string{"r1"}}}
	model.Backends = []Backend{{ID: "be_app", Servers: []string{"s1"}}}
	model.Servers = []Server{{ID: "s1", Address: "127.0.0.1", Port: 8080}}
	model.Rules = []Rule{{ID: "r1"}}

	next, err := Apply(model, Mutation{Type: "server.delete", ID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Servers) != 0 || len(next.Backends[0].Servers) != 0 {
		t.Fatalf("server delete did not detach references: %+v", next)
	}
	next, err = Apply(next, Mutation{Type: "rule.delete", ID: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Rules) != 0 || len(next.Frontends[0].Rules) != 0 {
		t.Fatalf("rule delete did not detach references: %+v", next)
	}
	if _, err := Apply(model, Mutation{Type: "backend.update", ID: "missing", Data: json.RawMessage(`{"name":"x"}`)}); err == nil {
		t.Fatal("expected missing entity error")
	}
	if _, err := Apply(model, Mutation{Type: "unknown"}); err == nil {
		t.Fatal("expected unsupported mutation error")
	}
	if _, err := Apply(model, Mutation{Type: "frontend.create"}); err == nil {
		t.Fatal("expected missing data error")
	}
	if _, err := Apply(model, Mutation{Type: "frontend.update", ID: "fe_web", Data: json.RawMessage(`{bad`)}); err == nil {
		t.Fatal("expected invalid patch json error")
	}
}

func mutation(kind, id, backendID, entityID string, data any) Mutation {
	raw, _ := json.Marshal(data)
	return Mutation{Type: kind, ID: id, BackendID: backendID, EntityID: entityID, Data: raw}
}

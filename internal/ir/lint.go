package ir

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	EntityID string   `json:"entity_id,omitempty"`
	Field    string   `json:"field,omitempty"`
	Message  string   `json:"message"`
}

func Lint(m *Model) []Issue {
	var issues []Issue
	backendIDs := map[string]bool{}
	serverIDs := map[string]bool{}
	ruleIDs := map[string]bool{}
	tlsIDs := map[string]bool{}
	healthIDs := map[string]bool{}

	backendNames := map[string]string{}
	for _, be := range m.Backends {
		if be.ID == "" {
			issues = append(issues, errIssue(be.ID, "id", "backend id is required"))
		}
		if prior := backendNames[strings.ToLower(be.Name)]; prior != "" && be.Name != "" {
			issues = append(issues, errIssue(be.ID, "name", fmt.Sprintf("duplicate backend name also used by %s", prior)))
		}
		backendNames[strings.ToLower(be.Name)] = be.ID
		backendIDs[be.ID] = true
		if len(be.Servers) == 0 {
			issues = append(issues, Issue{Severity: SeverityWarning, EntityID: be.ID, Field: "servers", Message: "backend has no servers"})
		}
		if be.HealthCheckID != "" {
			healthIDs[be.HealthCheckID] = false
		}
	}
	for _, srv := range m.Servers {
		serverIDs[srv.ID] = true
		if srv.Address == "" {
			issues = append(issues, errIssue(srv.ID, "address", "server address is required"))
		}
		if srv.Port <= 0 || srv.Port > 65535 {
			issues = append(issues, errIssue(srv.ID, "port", "server port must be between 1 and 65535"))
		}
	}
	for _, rule := range m.Rules {
		ruleIDs[rule.ID] = true
		if rule.Action.Type == "use_backend" && !backendIDs[rule.Action.BackendID] {
			issues = append(issues, errIssue(rule.ID, "action.backend_id", "rule points to a missing backend"))
		}
	}
	for _, tls := range m.TLSProfiles {
		tlsIDs[tls.ID] = true
		if tls.CertPath == "" {
			issues = append(issues, errIssue(tls.ID, "cert_path", "TLS certificate path is required"))
		}
	}
	for _, hc := range m.HealthChecks {
		healthIDs[hc.ID] = true
	}

	frontendBinds := map[string]string{}
	for _, fe := range m.Frontends {
		if fe.Bind == "" {
			issues = append(issues, errIssue(fe.ID, "bind", "frontend bind is required"))
		}
		if prior := frontendBinds[fe.Bind]; prior != "" {
			issues = append(issues, errIssue(fe.ID, "bind", fmt.Sprintf("bind collides with frontend %s", prior)))
		}
		frontendBinds[fe.Bind] = fe.ID
		if isTLSBind(fe.Bind) && fe.TLSID == "" {
			issues = append(issues, errIssue(fe.ID, "tls_id", "frontend binding on :443 requires a TLS profile"))
		}
		if fe.TLSID != "" && !tlsIDs[fe.TLSID] {
			issues = append(issues, errIssue(fe.ID, "tls_id", "frontend points to a missing TLS profile"))
		}
		if fe.DefaultBackend != "" && !backendIDs[fe.DefaultBackend] {
			issues = append(issues, errIssue(fe.ID, "default_backend", "frontend points to a missing default backend"))
		}
		for _, ruleID := range fe.Rules {
			if !ruleIDs[ruleID] {
				issues = append(issues, errIssue(fe.ID, "rules", fmt.Sprintf("frontend points to missing rule %s", ruleID)))
			}
		}
	}

	for _, be := range m.Backends {
		for _, serverID := range be.Servers {
			if !serverIDs[serverID] {
				issues = append(issues, errIssue(be.ID, "servers", fmt.Sprintf("backend points to missing server %s", serverID)))
			}
		}
		if be.HealthCheckID != "" && !healthIDs[be.HealthCheckID] {
			issues = append(issues, errIssue(be.ID, "health_check_id", "backend points to a missing health check"))
		}
	}

	for _, cache := range m.Caches {
		if hasEngine(m, EngineHAProxy) {
			issues = append(issues, Issue{Severity: SeverityWarning, EntityID: cache.ID, Message: "cache policies are Nginx-only and will be ignored for HAProxy"})
		}
	}

	return issues
}

func errIssue(entityID, field, msg string) Issue {
	return Issue{Severity: SeverityError, EntityID: entityID, Field: field, Message: msg}
}

func isTLSBind(bind string) bool {
	if strings.HasSuffix(bind, ":443") || bind == "443" {
		return true
	}
	_, port, err := net.SplitHostPort(bind)
	if err == nil {
		p, _ := strconv.Atoi(port)
		return p == 443
	}
	return false
}

func hasEngine(m *Model, e Engine) bool {
	for _, engine := range m.Engines {
		if engine == e {
			return true
		}
	}
	return false
}

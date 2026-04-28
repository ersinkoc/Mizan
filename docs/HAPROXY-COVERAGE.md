# HAProxy Coverage

Implemented in v0:

- `global` and `defaults` boilerplate
- `frontend` with `bind`, `mode`, ACLs, `use_backend`, and `default_backend`
- TLS bind with certificate path and ALPN
- `backend` with `balance`, `server`, `weight`, `maxconn`, and `check`
- HTTP health checks via `option httpchk`, plus server `check inter`, `rise`, and `fall`
- Bracketed IPv6 bind/server endpoints
- Quoted directive values, including certificate paths and ACL values that contain spaces or `#`
- Full-line and whitespace-prefixed inline comments in imported configs
- Opaque frontend/backend lines for unsupported directives such as `option forwardfor`, `http-request`, `redirect`, `option redispatch`, `http-reuse`, and `stick-table`
- Core parser/generator/parser round-trip coverage for frontends, backends, TLS certs, routing, servers, weights, max connections, health check timing, and opaque frontend/backend directives

Not yet complete:

- Fully modeled advanced directives beyond opaque preservation
- First-class stick table and advanced rate limiting editing
- Source-map driven native error mapping

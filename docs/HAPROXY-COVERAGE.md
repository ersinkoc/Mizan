# HAProxy Coverage

Implemented in v0:

- `global` and `defaults` boilerplate
- `frontend` with `bind`, `mode`, ACLs, `use_backend`, and `default_backend`
- TLS bind with certificate path and ALPN
- `backend` with `balance`, `server`, `weight`, `maxconn`, and `check`
- HTTP health checks via `option httpchk`
- Opaque backend lines when present in IR

Not yet complete:

- Full parser round-trip
- Stick tables and advanced rate limiting
- Source-map driven native error mapping


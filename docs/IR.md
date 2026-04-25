# Mizan IR

The IR is the single source of truth for the wizard, topology, generators, validators, and storage snapshots.

The v0 schema includes:

- `frontends`: listening endpoints with bind, protocol, optional TLS, rules, and default backend
- `backends`: upstream pools with algorithm, health check, and ordered server references
- `servers`: upstream address, port, weight, and connection limits
- `rules`: predicates and actions, currently focused on `use_backend`
- `tls_profiles`: certificate paths and TLS metadata
- `health_checks`: reusable HTTP/TCP health policies
- `view`: topology positions and canvas state

Snapshot versions are SHA-256 hashes of deterministic JSON output.


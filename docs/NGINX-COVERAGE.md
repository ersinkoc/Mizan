# Nginx Coverage

Implemented in v0:

- `events` and `http` contexts
- `upstream` blocks with weighted servers
- `server` blocks with `listen`, TLS certificate directives, and basic HTTP/2
- `location` blocks with `proxy_pass`
- `proxy_cache_path` for cache policies

Not yet complete:

- Full parser round-trip
- `map`, advanced `limit_req`, and include resolution
- Source-map driven native error mapping


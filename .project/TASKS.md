# Mizan — TASKS

> Phased delivery plan. ~12 phases, ~210 tasks, ~1,800 hours total (≈ 12 calendar months at ~38 h/week).

---

## Estimation Notes

- Hours are senior-engineer focused-work hours, not calendar time.
- Estimates assume a single primary developer plus AI-assisted execution (Ersin's typical workflow).
- Each phase ends with a release candidate tag and a working demo — the project is shippable, if narrower in scope, after every phase.
- Tasks marked **(BE)** are backend Go, **(FE)** are frontend React/TS, **(INF)** are infrastructure / CI / docs.

---

## Phase Summary

| Phase | Theme | Tasks | Hours |
|-------|-------|-------|-------|
| 1 | Foundation | 14 | 70 |
| 2 | Universal IR | 16 | 110 |
| 3 | HAProxy Translator + Parser | 22 | 170 |
| 4 | Nginx Translator + Parser | 22 | 160 |
| 5 | Storage & Snapshots | 14 | 90 |
| 6 | REST API + WebUI Skeleton | 18 | 140 |
| 7 | Wizard UI | 24 | 230 |
| 8 | React Flow Topology + Sync | 22 | 220 |
| 9 | Validation Pipeline | 14 | 110 |
| 10 | SSH Deployment & Clusters | 18 | 180 |
| 11 | Live Monitoring | 18 | 200 |
| 12 | Audit, Versioning, Polish | 16 | 130 |
| | **Total** | **218** | **1,810** |

---

## Phase 1 — Foundation (Week 1–2)

**Goal**: Working repo, CI green, base Go module, base React app, can run `make dev` locally.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 1.1 | Create GitHub org `mizanproxy`, repo `mizan`, set MIT license | INF | 1 |
| 1.2 | Initialize Go module `github.com/mizanproxy/mizan`; commit `cmd/mizan/main.go` skeleton | BE | 2 |
| 1.3 | Initialize Vite React 19 + TypeScript app under `webui/` | FE | 3 |
| 1.4 | Configure Tailwind v4.1 with `@theme` block from BRANDING.md palette | FE | 4 |
| 1.5 | Install shadcn/ui CLI; scaffold base components (button, input, dialog, sheet, tabs, toast) | FE | 4 |
| 1.6 | Add `next-themes` and verify dark/light theme switch on a placeholder page | FE | 2 |
| 1.7 | Write `Makefile` targets: `dev`, `ui`, `binary`, `release`, `test`, `lint` | INF | 4 |
| 1.8 | Create `internal/server/embed.go` skeleton with `//go:embed` and SPA fallback | BE | 4 |
| 1.9 | Wire `mizan serve` command to start HTTP server on `:7890` and serve embedded UI | BE | 5 |
| 1.10 | Write GitHub Actions: build matrix (linux/darwin/windows × amd64/arm64), test, lint, govulncheck | INF | 8 |
| 1.11 | Set up frontend ESLint + Prettier config; add to CI | INF | 3 |
| 1.12 | Set up `vitest`, `playwright`; smoke test in CI | INF | 6 |
| 1.13 | Write initial `README.md` (public-facing), `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` | INF | 4 |
| 1.14 | Tag `v0.0.1-alpha` release candidate (binary + tarball + checksums) via Actions | INF | 20 |

**Deliverable**: `mizan serve` shows a themed empty page on `localhost:7890`. CI is green.

---

## Phase 2 — Universal IR (Week 3–5)

**Goal**: Typed IR model in Go and TypeScript, generated from one source of truth, with structural lint and snapshot hashing.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 2.1 | Write IR schema source-of-truth in YAML at `internal/ir/schema.yaml` | INF | 4 |
| 2.2 | Build code-generator: YAML → Go structs (`internal/ir/types.go`) | BE | 10 |
| 2.3 | Build code-generator: YAML → Zod schemas + TS types (`webui/src/lib/ir-schema.ts`) | INF | 10 |
| 2.4 | Implement entity types: Frontend, Backend, Server, Rule, TLSProfile, HealthCheck, RateLimit, Cache, Logger | BE | 8 |
| 2.5 | Implement wrapper Model with id-keyed maps and ordered slices; helpers for CRUD | BE | 8 |
| 2.6 | Implement OpaqueBlock IR type for round-trip preservation | BE | 4 |
| 2.7 | Write canonicalizer: deterministic JSON output with sorted keys, stable array order | BE | 6 |
| 2.8 | Implement snapshot hash (SHA-256 of canonicalized bytes) | BE | 2 |
| 2.9 | Write IR mutation API (`Apply(model, mutation) (newModel, error)`) | BE | 12 |
| 2.10 | Implement structural lint: dangling refs, port collisions, duplicate names, missing TLS for `:443`, empty backends | BE | 14 |
| 2.11 | Write IR diff function (structural, not text) — produces add/remove/modify tree | BE | 10 |
| 2.12 | Implement view metadata (positions, zoom, pan) as part of IR | BE | 4 |
| 2.13 | Unit tests for canonicalize, hash, mutate (95%+ coverage) | BE | 12 |
| 2.14 | Unit tests for lint (every check has positive + negative cases) | BE | 8 |
| 2.15 | Mirror lint logic in TS (Zod refinements) so frontend can show issues without round-trip | FE | 6 |
| 2.16 | Document IR in `docs/IR.md` with diagrams and examples | INF | 6 |

**Deliverable**: `go test ./internal/ir/...` and `pnpm test ir` green. IR can be constructed, mutated, hashed, linted, diffed.

---

## Phase 3 — HAProxy Translator + Parser (Week 6–9)

**Goal**: IR ↔ haproxy.cfg round-trip for the supported feature set.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 3.1 | Study HAProxy 2.8 config reference; enumerate v1 directive coverage in `docs/HAPROXY-COVERAGE.md` | INF | 8 |
| 3.2 | Implement `global` and `defaults` block emission | BE | 6 |
| 3.3 | Implement `frontend` block emission (bind, mode, ACLs, use_backend, default_backend) | BE | 12 |
| 3.4 | Implement `backend` block emission (server lines, balance algorithm, options) | BE | 10 |
| 3.5 | Implement health check emission (`option httpchk`, `check inter`, `rise`, `fall`) | BE | 6 |
| 3.6 | Implement TLS emission (`bind :443 ssl crt`, `alpn`, `ciphers`) | BE | 8 |
| 3.7 | Implement stick-table emission with translation warnings when targeting Nginx-only | BE | 8 |
| 3.8 | Implement rate-limit via stick-table | BE | 8 |
| 3.9 | Implement source-map alongside emission | BE | 10 |
| 3.10 | Implement OpaqueBlock pass-through | BE | 6 |
| 3.11 | Build golden file test harness; first 20 fixture pairs | BE | 16 |
| 3.12 | Write parser tokenizer for HAProxy config (line-based, indent-aware sections) | BE | 14 |
| 3.13 | Implement parser: `global`, `defaults`, `frontend`, `backend`, `listen` | BE | 16 |
| 3.14 | Implement parser: ACL, use_backend, default_backend; produce IR Rules | BE | 10 |
| 3.15 | Implement parser: server lines, weights, checks | BE | 6 |
| 3.16 | Implement parser: TLS bind, certs | BE | 6 |
| 3.17 | Implement parser: stick-table, rate-limit | BE | 8 |
| 3.18 | Implement parser: OpaqueBlock for unknown directives | BE | 4 |
| 3.19 | Round-trip test: import 30 real-world configs from open-source projects, generate, diff, fail on non-trivial changes | BE | 16 |
| 3.20 | Write CLI: `mizan generate --target haproxy --out PATH` | BE | 4 |
| 3.21 | Write CLI: `mizan project import <haproxy.cfg>` | BE | 4 |
| 3.22 | Document supported directives + known limitations | INF | 6 |

**Deliverable**: Round-trip golden tests pass. `mizan generate --target haproxy` produces deployable output. `mizan import` consumes real HAProxy configs.

---

## Phase 4 — Nginx Translator + Parser (Week 10–13)

**Goal**: IR ↔ nginx.conf round-trip for the supported feature set.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 4.1 | Study Nginx 1.24+ config reference; enumerate coverage in `docs/NGINX-COVERAGE.md` | INF | 8 |
| 4.2 | Implement top-level emission: `events`, `http`, `stream` contexts | BE | 6 |
| 4.3 | Implement `upstream` block emission | BE | 8 |
| 4.4 | Implement `server` block emission (listen, server_name, SSL) | BE | 12 |
| 4.5 | Implement `location` block emission (proxy_pass, headers, rewrites) | BE | 10 |
| 4.6 | Implement `proxy_cache` config emission with HAProxy warnings | BE | 8 |
| 4.7 | Implement `limit_req_zone` + `limit_req` rate-limit emission | BE | 8 |
| 4.8 | Implement `map` block for header-driven routing | BE | 6 |
| 4.9 | Implement source-map alongside emission | BE | 10 |
| 4.10 | Implement OpaqueBlock pass-through | BE | 6 |
| 4.11 | Build golden file test harness; first 20 fixture pairs | BE | 14 |
| 4.12 | Write Nginx parser: brace-delimited block tokenizer | BE | 14 |
| 4.13 | Implement parser: `http`, `events`, `stream` | BE | 8 |
| 4.14 | Implement parser: `upstream`, `server`, `location` | BE | 14 |
| 4.15 | Implement parser: `proxy_pass`, headers, SSL, listen flags | BE | 10 |
| 4.16 | Implement parser: `proxy_cache`, `limit_req`, `map` | BE | 10 |
| 4.17 | Implement parser: include directives (with safe path resolution) | BE | 6 |
| 4.18 | Implement parser: OpaqueBlock for unknown directives | BE | 4 |
| 4.19 | Round-trip test: import 30 real-world configs | BE | 14 |
| 4.20 | CLI: `mizan generate --target nginx --out PATH` | BE | 2 |
| 4.21 | CLI: `mizan project import <nginx.conf>` | BE | 2 |
| 4.22 | Document supported directives + limitations | INF | 6 |

**Deliverable**: Round-trip golden tests pass for Nginx. CLI produces deployable nginx.conf.

---

## Phase 5 — Storage & Snapshots (Week 14–15)

**Goal**: Project files on disk, atomic writes, file locking, snapshot history.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 5.1 | Define filesystem layout (`~/.mizan/` tree) and create directory bootstrapper | BE | 4 |
| 5.2 | Implement atomic write helper (temp + fsync + rename) with cross-platform behavior | BE | 6 |
| 5.3 | Implement Unix `flock` lock with timeout and PID tracking | BE | 6 |
| 5.4 | Implement Windows `LockFileEx` lock | BE | 8 |
| 5.5 | Implement Project CRUD against filesystem | BE | 10 |
| 5.6 | Implement IR load/save with optimistic concurrency (version IDs) | BE | 8 |
| 5.7 | Implement snapshot directory and snapshot-on-mutate | BE | 8 |
| 5.8 | Implement snapshot retrieval and diff against historic snapshot | BE | 6 |
| 5.9 | Implement snapshot pruning (keep N most recent + tagged + within M days) | BE | 6 |
| 5.10 | Implement snapshot tagging | BE | 4 |
| 5.11 | Implement audit log appender (`audit.jsonl`, append-only) | BE | 4 |
| 5.12 | Implement state.json (last-opened project, UI prefs) | BE | 4 |
| 5.13 | Concurrency stress test (5 goroutines writing same project) | BE | 8 |
| 5.14 | Crash recovery test (simulate kill mid-write; verify last good snapshot is readable) | BE | 8 |

**Deliverable**: Projects persist correctly. Concurrent access fails safely. Crash recovery works.

---

## Phase 6 — REST API + WebUI Skeleton (Week 16–18)

**Goal**: HTTP layer, embedded SPA, basic navigation, theme polish, no actual editing yet.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 6.1 | HTTP server bootstrap with method-routed mux | BE | 4 |
| 6.2 | Middleware: request logging, panic recovery, CORS, basic rate limit | BE | 6 |
| 6.3 | API: `GET/POST/DELETE /api/v1/projects`, `GET/PATCH /api/v1/projects/{id}` | BE | 8 |
| 6.4 | API: `GET /api/v1/projects/{id}/ir`, `PATCH /api/v1/projects/{id}/ir` with If-Match | BE | 10 |
| 6.5 | API: snapshots list, get, revert, diff, tag | BE | 8 |
| 6.6 | API: error response envelope (RFC 7807 problem+json) | BE | 4 |
| 6.7 | TanStack Query setup; project list + detail hooks | FE | 6 |
| 6.8 | App shell: top bar, sidebar, project switcher (cmdk), theme toggle | FE | 12 |
| 6.9 | Project list page with search, "New Project" dialog | FE | 8 |
| 6.10 | Project home page (placeholder for IR editor; shows IR JSON for now) | FE | 6 |
| 6.11 | Empty states (no projects yet, no targets yet) per BRANDING.md | FE | 4 |
| 6.12 | Toast notifications via Sonner | FE | 3 |
| 6.13 | i18n setup with English + Turkish JSON, language switch in settings | FE | 8 |
| 6.14 | Settings page (theme, language, default project, telemetry opt-in/out) | FE | 6 |
| 6.15 | Health and version endpoints; About dialog reads version | BE+FE | 4 |
| 6.16 | OpenAPI spec generated from handlers (annotations + script) | INF | 12 |
| 6.17 | Integration test: spin up server, hit each endpoint with a known IR | BE | 12 |
| 6.18 | E2E smoke (Playwright): list → create → open → toggle theme → switch language | FE | 9 |

**Deliverable**: Working app shell. Can create and list projects. IR is stored and retrieved as JSON.

---

## Phase 7 — Wizard UI (Week 19–24)

**Goal**: Full step-based wizard binds to IR, every entity type editable.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 7.1 | Zustand IR slice with mutation dispatch and version tracking | FE | 8 |
| 7.2 | Optimistic update + 409 reconciliation flow | FE | 10 |
| 7.3 | Stepper component (linear nav with badge counts and validation status) | FE | 10 |
| 7.4 | Step: Project Basics (name, description, target engines) | FE | 4 |
| 7.5 | Step: Frontends list + sheet form | FE | 14 |
| 7.6 | Step: Backends list + sheet form (algorithm dropdown, health check picker) | FE | 14 |
| 7.7 | Step: Servers (nested under backend; weight, max_conn, attributes) | FE | 12 |
| 7.8 | Step: Rules / ACLs editor (predicate builder + action picker) | FE | 18 |
| 7.9 | Step: TLS Profiles (cert path, ciphers picker, ALPN multi-select) | FE | 12 |
| 7.10 | Step: Health Checks (HTTP/TCP type, path, expected status) | FE | 10 |
| 7.11 | Step: Rate Limits (key, period, request count) | FE | 8 |
| 7.12 | Step: Caches (Nginx-only; emits warning if HAProxy is targeted) | FE | 10 |
| 7.13 | Step: Loggers | FE | 6 |
| 7.14 | Step: Targets & Clusters (host, SSH user, port, sudo, reload command) | FE | 14 |
| 7.15 | Step: Review (read-only preview of all entities + warnings panel) | FE | 8 |
| 7.16 | Inline contextual help: each field has tooltip with HAProxy/Nginx directive name + docs link | FE | 14 |
| 7.17 | Bulk operations (multi-delete, bulk weight set) | FE | 8 |
| 7.18 | Keyboard shortcuts (cmd-K palette, j/k navigation between rows) | FE | 8 |
| 7.19 | Form-to-IR debounce + force-save behavior | FE | 6 |
| 7.20 | Display lint issues inline on offending fields | FE | 10 |
| 7.21 | Display target-capability warnings inline | FE | 8 |
| 7.22 | Test: every step has working create + edit + delete cases (Vitest + RTL) | FE | 16 |
| 7.23 | E2E (Playwright): wizard-only flow creates a deployable IR for both targets | FE | 12 |
| 7.24 | Empty state per step with "Add first ___" CTA | FE | 6 |

**Deliverable**: A user can build a complete HAProxy + Nginx-targeted project entirely through the wizard.

---

## Phase 8 — React Flow Topology + Two-Way Sync (Week 25–30)

**Goal**: Interactive canvas; edits in canvas reflect in wizard and vice versa.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 8.1 | Install `@xyflow/react`; integrate Zustand IR slice as graph source | FE | 6 |
| 8.2 | `buildGraph(model)` pure function: IR → ReactFlowNodes + Edges | FE | 12 |
| 8.3 | dagre layout for nodes lacking persisted positions | FE | 6 |
| 8.4 | Custom node: Frontend (port, TLS badge, protocol, status dot) | FE | 10 |
| 8.5 | Custom node: Backend (algorithm badge, server count, aggregate health) | FE | 10 |
| 8.6 | Custom node: Server (address, weight, status) | FE | 8 |
| 8.7 | Custom node: Rule (predicate text, target backend) | FE | 8 |
| 8.8 | Custom node: TLS, RateLimit, Cache, Logger as docked side-nodes | FE | 12 |
| 8.9 | Edge styling: default, conditional (rule), TLS-terminated | FE | 6 |
| 8.10 | Drag-to-create: pull from plus-handle to empty space → context menu | FE | 12 |
| 8.11 | Drag-to-connect: frontend → backend creates default_backend or attaches to selected rule | FE | 12 |
| 8.12 | Right-click context menu (edit, duplicate, delete, "view as wizard") | FE | 10 |
| 8.13 | "View as wizard" deep-link: jumps wizard to entity, opens form sheet | FE | 6 |
| 8.14 | Multi-select rubber-band; bulk delete | FE | 8 |
| 8.15 | Mini-map, zoom-to-fit, pan controls | FE | 6 |
| 8.16 | Search box: highlight matching nodes; auto-pan to first | FE | 8 |
| 8.17 | Persist node positions to IR `view` metadata via debounced patch | FE | 8 |
| 8.18 | Display lint issues as red badges on offending nodes | FE | 8 |
| 8.19 | Display target warnings as amber badges | FE | 6 |
| 8.20 | Hot-key: `g w` to topology, `g f` to wizard, `Esc` to deselect, `Del` to delete | FE | 6 |
| 8.21 | E2E: create entity in wizard, verify node appears; drag connection in canvas, verify wizard updates | FE | 12 |
| 8.22 | Performance: render 500-entity graph at 60fps; profile and memoize | FE | 30 |

**Deliverable**: Two-way sync works for every entity type. Canvas handles 500 entities at 60fps.

---

## Phase 9 — Validation Pipeline (Week 31–33)

**Goal**: Lint + native-binary-check + optional dry-run for both targets.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 9.1 | Validate service: `Validate(ctx, model, target) → ValidateResult` | BE | 6 |
| 9.2 | Wrap `haproxy -c -f <tmp>` invocation with timeout, capture stderr | BE | 6 |
| 9.3 | Wrap `nginx -t -c <tmp>` invocation | BE | 4 |
| 9.4 | Parse haproxy-c stderr for line numbers and severity | BE | 8 |
| 9.5 | Parse nginx-t stderr for line numbers and severity | BE | 6 |
| 9.6 | Map errors to IR entities via source map | BE | 8 |
| 9.7 | Cache validation results per IR snapshot hash | BE | 6 |
| 9.8 | Optional Docker dry-run runner (`haproxy:alpine`, `nginx:alpine`) | BE | 12 |
| 9.9 | API: `POST /api/v1/projects/{id}/validate` returning ValidateResult | BE | 4 |
| 9.10 | UI: Validation panel — pass/fail, line-numbered errors, "view in topology" jump links | FE | 14 |
| 9.11 | UI: Generated config viewer with CodeMirror 6 + syntax highlighting | FE | 14 |
| 9.12 | UI: Diff viewer (current vs last-deployed, current vs snapshot) | FE | 14 |
| 9.13 | CLI: `mizan validate [--target ...]` | BE | 4 |
| 9.14 | E2E: introduce an invalid IR, verify validation surfaces error on correct node | FE | 4 |

**Deliverable**: Validation works against real `haproxy` and `nginx` binaries. Errors surface visually on canvas and in wizard.

---

## Phase 10 — SSH Deployment & Clusters (Week 34–37)

**Goal**: Push generated config to remote hosts safely; cluster orchestration with rollback.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 10.1 | SSH client wrapper with agent + key-file + password fallback | BE | 10 |
| 10.2 | TOFU host key pinning per Target | BE | 6 |
| 10.3 | SFTP file upload with checksum verification | BE | 8 |
| 10.4 | Remote validate step: `ssh host '<target_binary> -c -f /tmp/...'` | BE | 6 |
| 10.5 | Backup current production config to timestamped path | BE | 4 |
| 10.6 | Atomic rename of temp → production path | BE | 4 |
| 10.7 | Reload command runner (configurable per target) | BE | 6 |
| 10.8 | Post-reload probe: HTTP GET or `<target_binary> -c -f <prod>` | BE | 8 |
| 10.9 | Rollback path: restore backup, re-issue reload | BE | 8 |
| 10.10 | Deploy state machine wired end-to-end with structured logs per state | BE | 14 |
| 10.11 | API: `POST /api/v1/projects/{id}/deploy` with progress events | BE | 6 |
| 10.12 | Cluster: parallel deploy with `parallelism` and `gate_on_failure` | BE | 14 |
| 10.13 | Encrypted secrets vault (Argon2id + AES-256-GCM) | BE | 16 |
| 10.14 | Passphrase prompt at server startup (terminal) and at deploy time (UI) | BE+FE | 10 |
| 10.15 | UI: Targets manager (CRUD targets, test connection, host-key approval) | FE | 16 |
| 10.16 | UI: Deploy panel — choose target/cluster, review diff, kick off deploy, watch progress | FE | 18 |
| 10.17 | UI: Audit log (read from `audit.jsonl`) with filters and entry detail drawer | FE | 14 |
| 10.18 | E2E: mock SSH server fixture; full happy-path + rollback path tests | BE+FE | 12 |

**Deliverable**: Real deploys to real HAProxy/Nginx hosts. Rollback works. Cluster orchestration works.

---

## Phase 11 — Live Monitoring (Week 38–42)

**Goal**: Stream runtime metrics from deployed targets; visualize on canvas + charts.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 11.1 | Collector interface and per-target goroutine lifecycle manager | BE | 10 |
| 11.2 | HAProxy Runtime API client: `show stat`, `show info` over UNIX/TCP socket | BE | 16 |
| 11.3 | HAProxy CSV parser → typed Snapshot | BE | 10 |
| 11.4 | Nginx OSS `stub_status` HTTP collector + parser | BE | 8 |
| 11.5 | Nginx Plus `/api` JSON collector | BE | 12 |
| 11.6 | Per-target/series ring buffer with two-segment iteration | BE | 12 |
| 11.7 | SSE broadcaster: pub/sub fan-out to subscribers per project | BE | 10 |
| 11.8 | API: `GET /api/v1/projects/{id}/monitor/snapshot` and `/stream` | BE | 8 |
| 11.9 | Reconnect with `Last-Event-ID` resume | BE | 6 |
| 11.10 | UI: monitor store (Zustand) with time-series ring synced from SSE | FE | 12 |
| 11.11 | UI: dashboard tab — per-target summary cards | FE | 12 |
| 11.12 | UI: per-backend chart panel (req/s, error rate, bytes, conns) — Recharts | FE | 18 |
| 11.13 | UI: per-server table with inline status, weight, current conns | FE | 10 |
| 11.14 | UI: topology canvas reflects live health (server color, backend aggregate) | FE | 12 |
| 11.15 | UI: time window selector (1h / 6h / 24h) with smooth re-render | FE | 8 |
| 11.16 | Optional Prometheus remote-write client for long-term retention | BE | 14 |
| 11.17 | CLI: `mizan monitor --target HOST` streaming console output | BE | 8 |
| 11.18 | E2E: mock LB process exposing fake stats; verify chart rendering and node colors | BE+FE | 14 |

**Deliverable**: Live charts and topology updates from real (and mock) HAProxy/Nginx targets.

---

## Phase 12 — Audit, Versioning, Diff Viewer, Polish (Week 43–45)

**Goal**: Polish the whole product — diff viewer, versioning UX, accessibility audit, performance pass, docs.

| # | Task | Type | Hrs |
|---|------|------|-----|
| 12.1 | Snapshot diff viewer: side-by-side wizard view of two snapshots | FE | 14 |
| 12.2 | Snapshot diff viewer: side-by-side topology view (added=green, removed=red, modified=amber) | FE | 14 |
| 12.3 | Snapshot diff viewer: structural diff tree | FE | 8 |
| 12.4 | Tag manager UI (apply, list, delete tags) | FE | 6 |
| 12.5 | Audit log: detailed event drawer with full metadata + IR snapshot link | FE | 8 |
| 12.6 | Audit log: export to CSV | FE | 4 |
| 12.7 | Empty state polish: every page has a clear next action | FE | 6 |
| 12.8 | Accessibility audit (axe-core); fix all WCAG 2.1 AA issues | FE | 14 |
| 12.9 | Keyboard navigation pass for topology canvas | FE | 8 |
| 12.10 | Performance pass: profile React renders, add memoization where needed | FE | 12 |
| 12.11 | Performance pass: backend pprof on heavy operations (validate, generate, monitor) | BE | 10 |
| 12.12 | Documentation: user guide with screenshots, video walkthrough | INF | 14 |
| 12.13 | Documentation: deployment guide for self-hosting on a VM | INF | 6 |
| 12.14 | Documentation: HAProxy and Nginx coverage docs final pass | INF | 4 |
| 12.15 | Release `v1.0.0`: signed binaries (cosign), Homebrew tap, GitHub Release with changelog | INF | 14 |
| 12.16 | Launch announcement: blog post, X thread, HN/Reddit submission | INF | 8 |

**Deliverable**: `v1.0.0` release. Public launch.

---

## Post-v1 Backlog (out of scope for the 12-month plan)

Captured here for reference; not estimated.

- Multi-user mode with RBAC (operator / reviewer / viewer roles)
- Caddy translator
- Traefik translator
- Envoy/xDS adapter
- Helm chart Mizan controller
- WASM plugin support for custom translators
- AI-assisted config diagnosis ("explain why this rule never matches")
- Live config hot-edit (preview changes against live traffic via shadow listener)

---

*End of TASKS.md*

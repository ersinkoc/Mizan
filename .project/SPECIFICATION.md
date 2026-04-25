# Mizan — SPECIFICATION

> Visual Config Architect for HAProxy & Nginx
> Version: 0.1.0 (planning)
> Owner: Ersin / ECOSTACK
> Status: Draft

---

## Table of Contents

1. [Overview](#1-overview)
2. [Goals & Non-Goals](#2-goals--non-goals)
3. [Personas](#3-personas)
4. [User Stories](#4-user-stories)
5. [Core Concepts](#5-core-concepts)
6. [Functional Requirements](#6-functional-requirements)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [REST API Surface](#8-rest-api-surface)
9. [Storage Schema](#9-storage-schema)
10. [Glossary](#10-glossary)

---

## 1. Overview

Mizan is a single-binary Go application with an embedded React 19 WebUI. It lets engineers design HAProxy and Nginx configurations through a **dual-mode editor** — a step-by-step wizard and an interactive React Flow topology canvas — that operate on the same Universal Intermediate Representation (IR). It then validates, deploys, and monitors those configurations on real servers.

The product covers the **full lifecycle**: design → validate → deploy → monitor → audit → roll back. All persistent state is stored as JSON files under the user's home directory (with file-level locking for concurrent safety), making it trivially Git-versionable.

---

## 2. Goals & Non-Goals

### 2.1 Goals

- Replace ad-hoc HAProxy/Nginx editing with a **visual, validated, version-controlled** workflow.
- Provide a **single source of truth** (the IR) that emits both HAProxy and Nginx output deterministically.
- Ship as **one self-contained binary**, no Docker, no Node runtime, no external database.
- Make **collaboration via Git** the default — JSON projects diff cleanly.
- Surface **runtime reality** alongside intended configuration: is the deployed config the one Mizan generated? Are backends actually healthy?
- Be **safe by construction**: every deploy is atomic, every change is logged, every reload can roll back.

### 2.2 Non-Goals

- **Not a load balancer**. Mizan does not handle traffic. HAProxy and Nginx do.
- **Not a service mesh control plane** (no Envoy/xDS, no Istio replacement).
- **Not a managed service**. It runs on the operator's machine or a private VM.
- **Not multi-tenant SaaS** in v1. Single-operator or small-team usage; auth is local-only.
- **Not a Kubernetes Ingress controller**. Mizan targets bare HAProxy and Nginx daemons.
- **Not an HAProxy/Nginx fork or wrapper**. It generates standard config files for unmodified upstream binaries.

---

## 3. Personas

**Pari — Senior SRE at a 200-engineer SaaS.** Maintains HAProxy in front of Kubernetes ingress and Nginx for static asset edges. Currently uses Ansible templates and dreads onboarding new teammates. Wants visual diffing and audit logs.

**Yusuf — Sole DevOps consultant.** Stands up edge stacks for clients across 20+ small environments. Reuses 80% of his configs. Wants project templates, fast bootstrapping, and a clean export workflow.

**Lina — Platform engineer at a fintech.** Compliance requires that every config change is reviewed and signed off. Needs deterministic generation (same IR → same bytes) and a clear audit trail.

**Tariq — Hobbyist self-hoster.** Runs a homelab with HAProxy fronting six services. Just wants a GUI that doesn't lie to him and doesn't require a Postgres install.

---

## 4. User Stories

The user stories below define the headline behaviors. Each is testable.

**US-01.** *As a new user*, I run `mizan serve` and reach a working WebUI on `localhost:7890` without any prior configuration step.

**US-02.** *As Pari*, I create a new project, choose "HAProxy", define one frontend on `:443` with TLS, attach two backends, and the canvas renders the topology automatically.

**US-03.** *As Yusuf*, I open an existing `haproxy.cfg` from disk and Mizan reverse-parses it into the IR, populating both wizard fields and topology nodes.

**US-04.** *As Lina*, I generate the config; Mizan invokes the local `haproxy -c` binary, surfaces validation errors with line numbers and node-level highlights on the canvas; nothing saves to disk until validation passes.

**US-05.** *As Pari*, I drag a backend node onto a frontend node in the canvas; the wizard panel for that frontend instantly shows the new ACL and `default_backend` association.

**US-06.** *As Tariq*, I deploy to my homelab via `mizan deploy --to homelab.local --reload`; if `haproxy -c` on the remote fails after upload, Mizan rolls back to the previous file before issuing the reload.

**US-07.** *As Lina*, I view the audit log and see every deploy with operator identity, timestamp, target host, IR snapshot ID, validation result, and reload outcome.

**US-08.** *As Pari*, I open the live monitoring tab; backend node colors on the topology turn red when health checks fail, and a charts panel shows current req/s and error rate per backend.

**US-09.** *As Yusuf*, I save a project as a "template", create a new project from it, change three fields, and ship it to a client environment in under five minutes.

**US-10.** *As any user*, I revert to the previous IR snapshot with one click; the wizard, topology, and generated output all update consistently.

---

## 5. Core Concepts

### 5.1 Project

A **Project** is a self-contained design unit, persisted as a directory under `~/.mizan/projects/<project-id>/`. It contains:

- The current IR (`config.json`)
- Snapshot history (`snapshots/<timestamp>-<id>.json`)
- Deployment targets (`targets.json`)
- Audit log (`audit.jsonl`)
- Project metadata (`project.json` — name, description, target engines, created/updated timestamps)

A user typically has one Project per environment (e.g., `edge-prod`, `edge-staging`, `homelab`).

### 5.2 Universal IR

The **Intermediate Representation** is a target-neutral, declarative model of a load-balancer configuration. It uses the following primary entities:

| Entity | Purpose |
|--------|---------|
| `Frontend` | A listening endpoint: bind address, port, TLS settings, protocol (HTTP / HTTP/2 / TCP) |
| `Backend` | A pool of upstream servers with a load-balancing algorithm and health-check policy |
| `Server` | A single upstream member of a Backend: address, port, weight, attributes |
| `Rule` (ACL/Route) | A predicate (`path == /api/*`, `host == api.example.com`) and an action (`use_backend`, `redirect`, `deny`) |
| `TLS` | A reusable TLS profile (cert paths, ciphers, ALPN, min version) |
| `HealthCheck` | A reusable health-check policy (interval, timeout, fall, rise, expected status) |
| `RateLimit` | A reusable rate-limit policy (requests per period per key) |
| `Cache` | A cache policy (Nginx-only — emits a warning if attached to a HAProxy-only project) |
| `Logger` | A log destination and format spec |
| `Cluster` | A group of `Target` hosts that should receive the same generated config |
| `Target` | A single deploy destination (host, SSH user, port, sudo, target binary path, reload command) |

Every IR mutation produces a **new immutable snapshot**, identified by a content hash (SHA-256 of canonicalized JSON), enabling deterministic diffing and reproducible deploys.

### 5.3 Translators

A **Translator** is a pure function `IR → cfg-bytes`:

- `HAProxyTranslator(ir) → []byte` (haproxy.cfg)
- `NginxTranslator(ir) → []byte` (nginx.conf)

Each translator runs **target capability checks** during translation and emits structured `Warning` and `Error` records when an IR feature has no equivalent in the target (e.g., HAProxy `stick-table` semantics have no direct Nginx mapping; Nginx `proxy_cache` does not exist in HAProxy).

A reverse path also exists: `Parser(cfg-bytes) → IR` for importing existing configurations. Reverse parsing is best-effort and surfaces unmapped directives as **opaque preserved blocks** that Mizan round-trips but does not visually represent.

### 5.4 Topology

The **Topology** is the visual graph rendered by React Flow. Each IR entity maps to a node type with deterministic positioning rules (top-down: Frontends → Rules → Backends → Servers; auxiliary nodes — TLS, RateLimit, Cache, Logger — appear as docked side-nodes attached to their owners).

Layout uses **dagre** for initial placement, after which user-positioned nodes persist their `(x, y)` in the IR's `view` metadata block. The IR is canonical; positioning is metadata.

---

## 6. Functional Requirements

### 6.1 Project Management (FR-PM)

- **FR-PM-01.** Create a new project with a name, description, and target engine set (HAProxy, Nginx, or both).
- **FR-PM-02.** List, open, rename, duplicate, archive, and delete projects.
- **FR-PM-03.** Mark a project as a "template"; instantiate a new project from a template (deep-copy IR, blank targets, reset audit).
- **FR-PM-04.** Import an existing `haproxy.cfg` or `nginx.conf` file into a new or existing project (reverse parser; opaque blocks preserved).
- **FR-PM-05.** Export the IR as `mizan-export.json` for offline inspection or transfer.

### 6.2 Universal IR Editing (FR-IR)

- **FR-IR-01.** All IR mutations route through a typed mutation API on the backend (`PATCH /api/projects/{id}/ir`) with optimistic concurrency via version IDs.
- **FR-IR-02.** Each mutation produces a snapshot, capped at the most recent 200 snapshots per project (configurable).
- **FR-IR-03.** Diff any two snapshots; the diff is rendered as a structural tree (added/removed/modified entities), not a text diff.
- **FR-IR-04.** Revert to any prior snapshot; the revert itself is a new snapshot (history is append-only).
- **FR-IR-05.** Validate IR integrity on every mutation: dangling references (e.g., a Rule pointing to a non-existent Backend) are rejected at the API boundary.

### 6.3 Wizard UI (FR-WZ)

- **FR-WZ-01.** Stepwise creation flow: Project Basics → Frontends → Backends → Servers → Rules → TLS / Health / Logging → Targets → Review.
- **FR-WZ-02.** Each step is independently navigable; the wizard does not block on incomplete steps but surfaces validation issues as a sticky panel.
- **FR-WZ-03.** Inline contextual help: each form field has a tooltip with the corresponding HAProxy and Nginx directive name and a link to upstream docs.
- **FR-WZ-04.** Form state binds to the IR via React Hook Form + Zod schemas generated from the same TypeScript types as the topology.
- **FR-WZ-05.** Field changes commit to the IR with a 300ms debounce; explicit Save button forces an immediate flush.

### 6.4 Topology Editor (FR-TP)

- **FR-TP-01.** Render the IR as a React Flow graph with custom node types per entity (Frontend, Backend, Server, Rule, TLS, RateLimit, Cache, Logger).
- **FR-TP-02.** Initial layout via dagre; user-positioned nodes persist in IR `view` metadata.
- **FR-TP-03.** Drag-to-connect: dragging from a Frontend's output handle to a Backend creates a default Rule (or attaches to `default_backend`).
- **FR-TP-04.** Drag-to-create: dragging from a node's plus-handle into empty space opens a context menu to create and connect a new entity.
- **FR-TP-05.** Right-click context menu per node: edit, duplicate, delete, "view as wizard" (jumps the wizard pane to that entity).
- **FR-TP-06.** Multi-select with rubber-band; bulk delete and bulk-attach operations.
- **FR-TP-07.** Mini-map and zoom-to-fit controls.
- **FR-TP-08.** **Two-way sync**: any IR mutation re-renders the topology within one render cycle; topology mutations dispatch the same mutation API as the wizard.
- **FR-TP-09.** Search box: highlight nodes matching name or address; auto-pan to first match.

### 6.5 Config Generation (FR-GN)

- **FR-GN-01.** Generate `haproxy.cfg` from the IR for any project that includes HAProxy as a target.
- **FR-GN-02.** Generate `nginx.conf` from the IR for any project that includes Nginx as a target.
- **FR-GN-03.** Generation is deterministic: same IR + same Mizan version = byte-identical output.
- **FR-GN-04.** Generation is idempotent: running it twice on unchanged IR produces no diff.
- **FR-GN-05.** Generated files include a header comment block with the project name, IR snapshot hash, generator version, and ISO-8601 timestamp.
- **FR-GN-06.** "Preview" mode: render generated output side-by-side in the WebUI with syntax highlighting (CodeMirror 6) without writing to disk.
- **FR-GN-07.** "Diff against last deploy": show a unified diff between currently-generated output and the bytes captured in the most recent successful deploy.

### 6.6 Validation (FR-VL)

- **FR-VL-01.** Run a **structural lint pass** on the IR before generation (rule references, port collisions, duplicate names, missing TLS certs for `:443` bindings).
- **FR-VL-02.** Invoke the local `haproxy -c -f <tmpfile>` binary on generated HAProxy output; surface stdout/stderr and exit code.
- **FR-VL-03.** Invoke the local `nginx -t -c <tmpfile>` binary on generated Nginx output; same surface.
- **FR-VL-04.** Map binary error output to IR entities by line-number lookup against the generator's source-map; highlight offending nodes on the topology with a red badge.
- **FR-VL-05.** Optional Docker-based dry-run: spin up a transient `haproxy:alpine` or `nginx:alpine` container, mount the generated file, observe the daemon's startup output for one second, then tear down. Off by default; opt-in via project setting.
- **FR-VL-06.** Validation results are cached per IR snapshot hash; re-validating an unchanged IR returns instantly.

### 6.7 Deployment (FR-DP)

- **FR-DP-01.** Deploy a generated config to a single Target via SSH (`golang.org/x/crypto/ssh` + `pkg/sftp`).
- **FR-DP-02.** Authentication options: SSH agent (preferred), private key file (with passphrase prompt), password (last resort).
- **FR-DP-03.** Deploy steps:
  1. Connect.
  2. Upload generated bytes to a temp path (`/tmp/mizan-<hash>.cfg`).
  3. Run `<target_binary> -c -f /tmp/mizan-<hash>.cfg` remotely; abort on failure.
  4. Backup current production config to `<config_path>.mizan-bak-<timestamp>`.
  5. Atomic move (SFTP rename) of temp file into production path.
  6. Issue reload command (configurable per Target; default `systemctl reload <unit>` or `kill -USR2 <pid>`).
  7. Wait `post_reload_grace` seconds (default 3).
  8. Run a post-reload health probe (HTTP GET to a configurable endpoint or `<target_binary> -c -f <production_path>` again).
  9. On failure, restore from backup and re-issue reload; mark deploy as **failed but rolled back**.
- **FR-DP-04.** Cluster deploy: orchestrate steps 1–9 across all Targets in a Cluster, with rolling strategy (1-by-1 by default; configurable `parallelism`).
- **FR-DP-05.** Cluster gating: if any Target's post-reload probe fails, halt remaining Targets, prompt for "rollback all" or "continue".
- **FR-DP-06.** Every deploy attempt writes one append-only record to `audit.jsonl` regardless of outcome.

### 6.8 Live Monitoring (FR-MN)

- **FR-MN-01.** Per-Target monitoring agent runs in the Mizan backend (no daemon installed on the LB host).
- **FR-MN-02.** **HAProxy collector**: connect to the Runtime API socket (UNIX or TCP) configured per Target; poll `show stat` and `show info` every 5 seconds (configurable `1s–60s`).
- **FR-MN-03.** **Nginx collector** (open source): HTTP GET to a configured `stub_status` endpoint; parse `Active connections`, `accepts`, `handled`, `requests`, `Reading`, `Writing`, `Waiting`.
- **FR-MN-04.** **Nginx Plus collector** (optional): HTTP GET to `/api/N/http/upstreams` (and related endpoints); parse JSON.
- **FR-MN-05.** Stream metric updates to the WebUI via SSE (`GET /api/projects/{id}/monitor/stream`).
- **FR-MN-06.** Topology canvas reflects health: Server nodes turn green/yellow/red based on the latest poll; Backend nodes show aggregate status.
- **FR-MN-07.** Charts panel: req/s, error rate (5xx), bytes in/out, active connections — per Backend, per Server, per Target. Last 1h / 6h / 24h windows.
- **FR-MN-08.** Time-series storage is in-memory ring buffer (default 24h × 5s = 17,280 points per series), capped per project. No external time-series DB.
- **FR-MN-09.** Optional Prometheus remote-write export of the same metrics for users who want long-term retention.

### 6.9 Versioning & Audit (FR-AD)

- **FR-AD-01.** Every IR mutation creates a snapshot; every deploy creates an audit log entry.
- **FR-AD-02.** Audit log is JSON Lines, one event per line, append-only, never edited.
- **FR-AD-03.** Audit event schema: `{event_id, project_id, timestamp, actor, action, ir_snapshot_hash, target_host, target_engine, validation_result, reload_result, rollback_performed, error_message}`.
- **FR-AD-04.** Audit viewer with filters by date range, actor, action, target, outcome.
- **FR-AD-05.** Snapshot diff viewer (side-by-side wizard view + side-by-side topology view + structural diff tree).
- **FR-AD-06.** Tag a snapshot (e.g., "release-2026-04-23"); tags survive snapshot pruning.

### 6.10 CLI (FR-CL)

The CLI mirrors the API surface so headless usage and CI/CD integration are first-class.

- **FR-CL-01.** `mizan project new|list|open|delete|export|import`
- **FR-CL-02.** `mizan frontend|backend|server|rule|tls|health add|edit|delete`
- **FR-CL-03.** `mizan generate --target {haproxy|nginx} [--out PATH]`
- **FR-CL-04.** `mizan validate [--target ...]`
- **FR-CL-05.** `mizan deploy --to HOST [--reload] [--dry-run]`
- **FR-CL-06.** `mizan cluster deploy --name CLUSTER`
- **FR-CL-07.** `mizan monitor --target HOST` (streaming console output, Ctrl-C to exit)
- **FR-CL-08.** `mizan audit show [--filter ...]`
- **FR-CL-09.** `mizan serve` to start the WebUI server.
- **FR-CL-10.** All commands accept `--project NAME_OR_ID`; default is the most recently-opened project (stored in `~/.mizan/state.json`).

### 6.11 Authentication & Authorization (FR-AU)

v1 is **single-operator** by default. Multi-user is in scope as a later phase but not v1.

- **FR-AU-01.** WebUI binds to `127.0.0.1:7890` by default. To expose on the network, the operator must pass `--bind 0.0.0.0:7890` and **must** also configure auth.
- **FR-AU-02.** Optional auth modes: HTTP Basic, bearer token, or OIDC (Google, GitHub, generic). Configured via `~/.mizan/auth.json`.
- **FR-AU-03.** When auth is enabled, every API call requires a valid session; the actor identity is recorded in audit events.
- **FR-AU-04.** SSH credentials are **never** stored in project files. They live only in `~/.mizan/secrets/<target-id>.json`, encrypted at rest with a key derived from a user-supplied passphrase (Argon2id KDF).

---

## 7. Non-Functional Requirements

### 7.1 Performance

- **NFR-P-01.** WebUI cold start < 1 second (binary size budget < 30 MB compressed, < 80 MB uncompressed).
- **NFR-P-02.** Wizard form mutation → topology re-render < 50 ms for projects with up to 500 entities.
- **NFR-P-03.** Validation pass on a 200-Frontend / 1000-Backend project < 2 seconds excluding subprocess time.
- **NFR-P-04.** Monitor SSE stream latency < 200 ms p95 from poll to UI render.

### 7.2 Reliability & Safety

- **NFR-R-01.** Every disk write is atomic: write-to-temp + fsync + rename.
- **NFR-R-02.** All concurrent project opens use OS-level advisory locks (`flock` on Unix, `LockFileEx` on Windows); a second process opening the same project gets a clear "locked by PID N" error.
- **NFR-R-03.** No deploy ever leaves a target with broken config; rollback is tested in every release.
- **NFR-R-04.** Crash recovery: if Mizan crashes mid-mutation, the last good snapshot is the latest readable state; no in-progress writes are visible.

### 7.3 Security

- **NFR-S-01.** SSH host keys are pinned per Target on first connect (TOFU); subsequent mismatches abort the deploy.
- **NFR-S-02.** Secrets at rest are encrypted with AES-256-GCM, key derived via Argon2id from a user passphrase (entered at `mizan serve` startup or `mizan deploy` time).
- **NFR-S-03.** No secrets are ever logged.
- **NFR-S-04.** Generated config files do not embed plaintext credentials in production paths; passwords/tokens reference environment variables or external files.

### 7.4 Portability

- **NFR-O-01.** Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64). FreeBSD best-effort.
- **NFR-O-02.** No runtime dependency beyond the binary itself for the WebUI/CLI. Local validation requires the target binary (`haproxy`, `nginx`) to be on `PATH`.

### 7.5 Observability of Mizan Itself

- **NFR-X-01.** Structured logs (JSON) on stdout; configurable level (`debug|info|warn|error`).
- **NFR-X-02.** Optional Prometheus `/metrics` endpoint for Mizan internals (request latencies, deploy counts, validation outcomes).
- **NFR-X-03.** Health endpoint `/healthz` for liveness; `/readyz` for readiness.

### 7.6 Internationalization

- **NFR-I-01.** UI strings externalized; English (`en`) and Turkish (`tr`) ship in v1.
- **NFR-I-02.** Date/time formatting respects browser locale; numbers respect locale.
- **NFR-I-03.** Right-to-left languages out of scope for v1.

### 7.7 Accessibility

- **NFR-A-01.** WCAG 2.1 AA color contrast for both themes.
- **NFR-A-02.** All interactive elements keyboard-reachable; topology canvas has a keyboard navigation mode.
- **NFR-A-03.** ARIA labels on all custom controls.

---

## 8. REST API Surface

All endpoints under `/api/v1`. JSON bodies, JSON responses. Optional `If-Match` header for optimistic concurrency on mutations (returns the current IR version ID).

### Projects

```
GET    /projects
POST   /projects
GET    /projects/{id}
PATCH  /projects/{id}
DELETE /projects/{id}
POST   /projects/{id}/duplicate
POST   /projects/{id}/import
GET    /projects/{id}/export
```

### IR

```
GET   /projects/{id}/ir
PATCH /projects/{id}/ir              ; typed mutation; If-Match required
GET   /projects/{id}/ir/snapshots
GET   /projects/{id}/ir/snapshots/{hash}
POST  /projects/{id}/ir/revert       ; body: {snapshot_hash}
POST  /projects/{id}/ir/diff         ; body: {from_hash, to_hash}
POST  /projects/{id}/ir/tag          ; body: {snapshot_hash, label}
```

### Generation & Validation

```
POST /projects/{id}/generate         ; body: {target: "haproxy"|"nginx"}
POST /projects/{id}/validate         ; body: {target?: "haproxy"|"nginx", dry_run?: bool}
```

### Deployment

```
POST /projects/{id}/targets
GET  /projects/{id}/targets
POST /projects/{id}/clusters
POST /projects/{id}/deploy           ; body: {target_id?, cluster_id?, generate_target}
GET  /projects/{id}/deploy/{deploy_id}
```

### Monitoring (SSE & WebSocket)

```
GET  /projects/{id}/monitor/snapshot          ; one-shot
GET  /projects/{id}/monitor/stream            ; text/event-stream
WS   /projects/{id}/monitor/ops               ; bidirectional control (optional)
```

### Audit

```
GET  /projects/{id}/audit?from=&to=&actor=&action=
```

### System

```
GET /healthz
GET /readyz
GET /metrics                ; Prometheus
GET /version
```

---

## 9. Storage Schema

### 9.1 Filesystem Layout

```
~/.mizan/
├── state.json                              # last-opened project, UI prefs
├── auth.json                               # optional auth config (mode, oidc settings)
├── secrets/
│   └── <target-id>.json                    # AES-256-GCM encrypted SSH creds
└── projects/
    └── <project-id>/
        ├── project.json                    # metadata
        ├── config.json                     # current IR (canonicalized)
        ├── targets.json                    # deploy targets & clusters
        ├── audit.jsonl                     # append-only audit log
        └── snapshots/
            ├── 2026-04-25T1407-<hash>.json
            ├── 2026-04-25T1352-<hash>.json
            └── ...
```

### 9.2 IR JSON Schema (excerpt)

```json
{
  "version": 1,
  "id": "01HXG8...",
  "name": "edge-prod",
  "engines": ["haproxy", "nginx"],
  "frontends": [
    {
      "id": "fe_web",
      "name": "web",
      "bind": ":443",
      "protocol": "http",
      "tls_id": "tls_default",
      "rules": ["r_api", "r_static"],
      "default_backend": "be_app",
      "view": { "x": 100, "y": 100 }
    }
  ],
  "backends": [
    {
      "id": "be_app",
      "name": "app-pool",
      "algorithm": "leastconn",
      "health_check_id": "hc_default",
      "servers": ["s_app_1", "s_app_2"],
      "view": { "x": 400, "y": 200 }
    }
  ],
  "servers": [
    {
      "id": "s_app_1",
      "address": "10.0.1.10",
      "port": 8080,
      "weight": 100,
      "max_conn": 1024
    }
  ],
  "rules": [
    {
      "id": "r_api",
      "predicate": { "type": "path_prefix", "value": "/api/" },
      "action": { "type": "use_backend", "backend_id": "be_api" }
    }
  ],
  "tls_profiles": [
    {
      "id": "tls_default",
      "cert_path": "/etc/mizan/certs/edge.pem",
      "key_path": "/etc/mizan/certs/edge.key",
      "ciphers": "TLS_AES_256_GCM_SHA384:...",
      "min_version": "TLSv1.2",
      "alpn": ["h2", "http/1.1"]
    }
  ],
  "health_checks": [
    {
      "id": "hc_default",
      "type": "http",
      "path": "/healthz",
      "expected_status": [200],
      "interval_ms": 2000,
      "timeout_ms": 1000,
      "rise": 2,
      "fall": 3
    }
  ],
  "rate_limits": [],
  "caches": [],
  "loggers": [],
  "view": {
    "zoom": 1.0,
    "pan": { "x": 0, "y": 0 }
  }
}
```

### 9.3 Canonicalization Rules

- Map keys are **sorted lexicographically**.
- Arrays preserve user-defined order (translation depends on it for ACL evaluation order).
- Numbers serialize without trailing zeros.
- Snapshot hash = SHA-256 of canonicalized UTF-8 bytes.

### 9.4 File Locking Protocol

- `flock(LOCK_EX)` on `~/.mizan/projects/<id>/.lock` for the duration of any write transaction.
- Read paths use `flock(LOCK_SH)` for consistency on slow filesystems.
- Lock acquisition timeout: 5 seconds; on timeout, return `409 Conflict` with the holding PID.

---

## 10. Glossary

| Term | Meaning |
|------|---------|
| **IR** | Intermediate Representation — the universal data model |
| **Translator** | Pure function that converts IR into target-specific config bytes |
| **Project** | A self-contained design unit with its own IR, targets, history |
| **Snapshot** | An immutable IR state, content-addressed by SHA-256 hash |
| **Target** | A single deployable host (HAProxy or Nginx server) |
| **Cluster** | A group of Targets that should receive the same generated config |
| **Frontend** | A listening endpoint in the IR (port + protocol + TLS) |
| **Backend** | A pool of upstream servers |
| **Server** | A single upstream member of a Backend |
| **Rule** | An ACL/route: predicate + action |
| **TOFU** | Trust On First Use — SSH host key pinning policy |
| **Reload** | Sending HAProxy/Nginx the signal to reread config without dropping connections |

---

*End of SPECIFICATION.md*

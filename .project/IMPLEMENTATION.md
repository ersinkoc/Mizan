# Mizan — IMPLEMENTATION

> Architecture, packages, key algorithms.
> Companion to `SPECIFICATION.md`.

---

## Table of Contents

1. [Tech Stack](#1-tech-stack)
2. [Repository Layout](#2-repository-layout)
3. [Backend Architecture](#3-backend-architecture)
4. [Frontend Architecture](#4-frontend-architecture)
5. [Universal IR & Translators](#5-universal-ir--translators)
6. [Wizard ↔ Topology Sync](#6-wizard--topology-sync)
7. [Storage with File Locking](#7-storage-with-file-locking)
8. [Validation Pipeline](#8-validation-pipeline)
9. [SSH Deployment](#9-ssh-deployment)
10. [Live Monitoring](#10-live-monitoring)
11. [Real-Time Updates (SSE)](#11-real-time-updates-sse)
12. [Build & Distribution](#12-build--distribution)
13. [Security](#13-security)
14. [Testing Strategy](#14-testing-strategy)

---

## 1. Tech Stack

### Backend (Go 1.23+)

**Standard library first.** Allowed external imports:

| Module | Purpose |
|--------|---------|
| `golang.org/x/crypto/ssh` | SSH client for deployments |
| `github.com/pkg/sftp` | SFTP file uploads |
| `golang.org/x/sys` | Cross-platform `flock` (also Windows file locks) |
| `golang.org/x/term` | Passphrase prompts in CLI |
| `gopkg.in/yaml.v3` | YAML import/export (optional, for CI scripts) |

Anything else gets pushback. No web framework — use `net/http` + `http.ServeMux` (Go 1.22+ mux supports method-routing). No ORM — JSON files. No logging library beyond `log/slog`.

### Frontend (React 19 + Vite + TypeScript)

```json
{
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "@xyflow/react": "^12.x",
    "tailwindcss": "^4.1.x",
    "@radix-ui/react-*": "latest stable",
    "zustand": "^5.x",
    "@tanstack/react-query": "^5.x",
    "react-hook-form": "^7.x",
    "@hookform/resolvers": "^3.x",
    "zod": "^3.x",
    "lucide-react": "^0.4xx.x",
    "recharts": "^2.x",
    "date-fns": "^4.x",
    "@codemirror/state": "^6.x",
    "@codemirror/view": "^6.x",
    "@codemirror/lang-css": "^6.x",
    "dagre": "^0.8.x",
    "next-themes": "^0.4.x",
    "sonner": "^1.x",
    "cmdk": "^1.x"
  },
  "devDependencies": {
    "vite": "^5.x",
    "@vitejs/plugin-react": "^4.x",
    "typescript": "^5.6.x",
    "vitest": "^2.x",
    "@playwright/test": "^1.x"
  }
}
```

**shadcn/ui is copy-paste components, not a dependency** — it lives in `webui/src/components/ui/` and uses Radix primitives + Tailwind classes. Versions pinned per shadcn install.

---

## 2. Repository Layout

```
mizan/
├── cmd/
│   └── mizan/
│       └── main.go                      # entrypoint, CLI dispatch, serve cmd
├── internal/
│   ├── api/                             # HTTP handlers (REST, SSE)
│   │   ├── projects.go
│   │   ├── ir.go
│   │   ├── generate.go
│   │   ├── validate.go
│   │   ├── deploy.go
│   │   ├── monitor.go
│   │   ├── audit.go
│   │   ├── auth.go
│   │   └── middleware.go
│   ├── ir/                              # universal model
│   │   ├── types.go                     # entity structs
│   │   ├── canonicalize.go              # deterministic JSON output
│   │   ├── hash.go                      # SHA-256 snapshot hashes
│   │   ├── lint.go                      # structural validation
│   │   ├── mutate.go                    # typed mutation API
│   │   └── parser/
│   │       ├── haproxy.go               # haproxy.cfg → IR
│   │       └── nginx.go                 # nginx.conf → IR
│   ├── translate/                       # IR → cfg
│   │   ├── haproxy/
│   │   │   ├── translator.go
│   │   │   ├── frontend.go
│   │   │   ├── backend.go
│   │   │   ├── tls.go
│   │   │   └── sourcemap.go             # IR entity → output line range
│   │   └── nginx/
│   │       ├── translator.go
│   │       ├── server.go
│   │       ├── upstream.go
│   │       └── sourcemap.go
│   ├── store/                           # JSON file storage
│   │   ├── store.go
│   │   ├── lock.go                      # flock-based locking
│   │   ├── atomic.go                    # write-temp + fsync + rename
│   │   ├── snapshot.go
│   │   └── audit.go                     # append-only JSONL
│   ├── deploy/                          # SSH delivery
│   │   ├── ssh.go
│   │   ├── upload.go
│   │   ├── reload.go
│   │   ├── rollback.go
│   │   ├── cluster.go
│   │   └── hostkey.go                   # TOFU pinning
│   ├── validate/                        # config validation
│   │   ├── lint.go                      # IR-level lint
│   │   ├── haproxy.go                   # haproxy -c invocation
│   │   ├── nginx.go                     # nginx -t invocation
│   │   └── docker.go                    # optional dry-run runner
│   ├── monitor/                         # live telemetry
│   │   ├── collector.go                 # interface
│   │   ├── haproxy_runtime.go           # Runtime API client
│   │   ├── nginx_status.go              # stub_status parser
│   │   ├── nginx_plus.go                # /api parser
│   │   ├── ringbuf.go                   # time-series ring buffer
│   │   └── stream.go                    # SSE broadcaster
│   ├── auth/                            # local auth
│   │   ├── basic.go
│   │   ├── token.go
│   │   ├── oidc.go
│   │   └── session.go
│   ├── secrets/                         # encrypted credential vault
│   │   ├── vault.go
│   │   └── kdf.go                       # Argon2id
│   ├── cli/                             # cobra-free CLI dispatch
│   │   ├── root.go
│   │   ├── project.go
│   │   ├── frontend.go
│   │   ├── backend.go
│   │   ├── server.go
│   │   ├── generate.go
│   │   ├── validate.go
│   │   ├── deploy.go
│   │   └── serve.go
│   ├── server/                          # HTTP server bootstrap
│   │   ├── server.go
│   │   ├── embed.go                     # //go:embed for webui assets
│   │   └── routes.go
│   └── version/                         # build-time version info
│       └── version.go
├── webui/                               # React 19 frontend
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── routes/
│   │   ├── components/
│   │   │   ├── ui/                      # shadcn primitives
│   │   │   ├── topology/                # React Flow canvas + node types
│   │   │   ├── wizard/                  # step components
│   │   │   ├── monitor/                 # charts, dashboards
│   │   │   └── audit/
│   │   ├── store/                       # Zustand slices
│   │   │   ├── ir.ts
│   │   │   ├── ui.ts
│   │   │   └── monitor.ts
│   │   ├── api/                         # TanStack Query hooks
│   │   ├── lib/
│   │   │   ├── ir-schema.ts             # Zod schemas (mirror of Go types)
│   │   │   ├── topology-layout.ts       # dagre wrapper
│   │   │   └── codemirror.ts
│   │   ├── i18n/
│   │   │   ├── en.json
│   │   │   └── tr.json
│   │   └── styles/
│   │       └── globals.css              # Tailwind v4 @theme block
│   └── public/
├── docs/
│   ├── README.md
│   ├── SPECIFICATION.md
│   ├── IMPLEMENTATION.md
│   ├── TASKS.md
│   └── BRANDING.md
├── testdata/                            # fixture configs for parser tests
│   ├── haproxy/
│   └── nginx/
├── .github/workflows/
│   ├── ci.yml
│   └── release.yml
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── README.md
```

---

## 3. Backend Architecture

The backend is a **layered monolith** with strict directional dependencies. Top to bottom:

```
cmd/mizan
   │
   ▼
internal/cli  ────────►  internal/server (HTTP) ──────► internal/api (handlers)
                                                            │
                                                            ▼
                                                ┌──────── core services ────────┐
                                                ▼                                ▼
                                       internal/ir, translate         internal/store, deploy,
                                                                       validate, monitor, secrets, auth
```

Rules:

- `internal/api` is the only package that touches HTTP types.
- `internal/ir` is pure: no I/O, no network, no globals. Everything else can depend on it; it depends on nothing from this project.
- `internal/store` is the only package that touches the filesystem for project data.
- `internal/deploy` is the only package that opens SSH connections.
- `internal/cli` and `internal/api` are siblings; the CLI invokes service packages directly (no in-process HTTP loopback).

### 3.1 Service Layer Pattern

Each domain has a service struct:

```go
// internal/store/store.go
type Store struct {
    root string  // ~/.mizan
    log  *slog.Logger
}

func New(root string, log *slog.Logger) *Store { ... }

func (s *Store) ListProjects(ctx context.Context) ([]ProjectMeta, error)
func (s *Store) GetIR(ctx context.Context, projectID string) (*ir.Model, string, error) // returns IR + version
func (s *Store) SaveIR(ctx context.Context, projectID string, m *ir.Model, ifMatch string) (newVersion string, err error)
```

Services accept `context.Context` first, return `(value, error)`. No singletons, no init-time DI containers; main.go wires dependencies explicitly.

### 3.2 HTTP Server

```go
// internal/server/server.go
func New(cfg Config, store *store.Store, ...) *http.Server {
    mux := http.NewServeMux()
    api.Register(mux, store, ...)
    mux.Handle("/", embeddedUI())  // serves SPA, falls through to index.html
    return &http.Server{Addr: cfg.Bind, Handler: mux}
}
```

`api.Register` uses Go 1.22+ method-routed mux:

```go
mux.HandleFunc("GET    /api/v1/projects",            h.listProjects)
mux.HandleFunc("POST   /api/v1/projects",            h.createProject)
mux.HandleFunc("PATCH  /api/v1/projects/{id}/ir",    h.patchIR)
mux.HandleFunc("GET    /api/v1/projects/{id}/monitor/stream", h.streamMonitor)
```

Middleware: request logging, panic recovery, auth check, CORS (off by default; on only when `--bind 0.0.0.0`), rate limit on mutation endpoints (token bucket, in-memory).

---

## 4. Frontend Architecture

### 4.1 State Layers

The frontend uses **three coexisting state layers**:

| Layer | Tool | Holds |
|-------|------|-------|
| Server cache | TanStack Query | Snapshots, audit, monitor data — anything fetched |
| Client state | Zustand | UI prefs, current view, transient form drafts |
| IR (canonical) | Zustand + immer | The active IR model — single source of truth for both wizard and topology |

The IR slice exposes typed mutators:

```ts
const useIR = create<IRStore>()(immer((set, get) => ({
  model: emptyIR(),
  version: '0',
  patch(mutation: IRMutation) {
    // optimistic local update
    set(s => applyMutation(s.model, mutation));
    // dispatch to backend (PATCH /api/v1/projects/:id/ir)
    return apiPatchIR(get().model.id, mutation, get().version);
  },
  setFromServer(model, version) {
    set(s => { s.model = model; s.version = version; });
  },
})));
```

### 4.2 Wizard

Wizard steps are sibling React components, each scoped to a slice of the IR:

```tsx
// FrontendStep.tsx
function FrontendStep({ frontendId }: { frontendId: string }) {
  const frontend = useIR(s => s.model.frontends.find(f => f.id === frontendId));
  const patch = useIR(s => s.patch);
  const form = useForm<FrontendInput>({
    resolver: zodResolver(FrontendSchema),
    values: frontend,                    // bind directly to IR slice
  });

  const onChange = useDebouncedCallback((data) => {
    patch({ type: 'frontend.update', id: frontendId, data });
  }, 300);

  return ( ... <form onChange={form.handleSubmit(onChange)}>...</form> ... );
}
```

### 4.3 Topology

React Flow nodes consume the same IR slice:

```tsx
// TopologyCanvas.tsx
function TopologyCanvas() {
  const model = useIR(s => s.model);
  const patch = useIR(s => s.patch);

  const { nodes, edges } = useMemo(() => buildGraph(model), [model]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      onNodesChange={(changes) => {
        // position changes → IR view metadata
        for (const c of changes) {
          if (c.type === 'position' && c.position) {
            patch({ type: 'view.move', entityId: c.id, x: c.position.x, y: c.position.y });
          }
        }
      }}
      onConnect={(params) => {
        // edge creation → ACL/default_backend mutation
        patch(buildConnectMutation(params, model));
      }}
    />
  );
}
```

`buildGraph(model)` is a pure function: same IR → same nodes/edges (apart from React Flow's internal IDs which are stable from IR entity IDs). This is the foundation of the bidirectional sync — there's no separate topology state, only a derived view.

---

## 5. Universal IR & Translators

### 5.1 Why an IR

Without an IR, every UI feature has to be aware of HAProxy and Nginx specifics — a combinatorial mess. With an IR:

- The wizard, topology, and validators only know **one model**.
- Translators are pure functions, easy to test exhaustively.
- New targets (Caddy, Traefik) are just new translators.
- Reverse parsers can normalize existing configs into one shape regardless of source.

### 5.2 IR vs Target Capabilities

Some features map cleanly across both targets (Frontend, Backend, Server, basic ACL). Some are target-specific:

| IR feature | HAProxy | Nginx |
|------------|---------|-------|
| Round-robin / least-conn | ✓ | ✓ |
| `source` IP hash | ✓ | ✓ (`ip_hash`) |
| `uri` hash | ✓ | ✓ (`hash $request_uri`) |
| Stick-tables | ✓ | ✗ → warning |
| Weight-based | ✓ | ✓ |
| Active health checks | ✓ | ✗ (OSS) / ✓ (Plus) |
| Passive health checks | ✓ | ✓ |
| Cache | ✗ → warning | ✓ |
| Brotli | ✗ → warning | ✓ |
| ACL on header | ✓ | ✓ (`map`) |
| ACL on path | ✓ | ✓ (`location`) |
| Rate limit per IP | ✓ (stick-table) | ✓ (`limit_req_zone`) |
| TLS termination | ✓ | ✓ |
| HTTP/2 | ✓ | ✓ |
| HTTP/3 | ✓ (2.9+) | ✓ (1.25+) |

When the user constructs an IR feature unsupported by a chosen target, the translator emits a `ir.Warning` that the UI surfaces both at the entity (red badge on the canvas node) and in a project-level diagnostic panel.

### 5.3 Source Maps

Each translator emits not just bytes but a **source map**: a slice of `{StartLine, EndLine, EntityID}` records.

```go
type SourceMap struct {
    Entries []SourceMapEntry
}
type SourceMapEntry struct {
    StartLine int
    EndLine   int
    EntityID  string
}
```

When `haproxy -c -f` reports an error on line 47, the validator looks up which IR entity owns lines 40–55 in the source map and surfaces `Backend "api-pool" → unknown directive 'roundrobin2'`.

### 5.4 Round-Trip via Reverse Parsers

The reverse parser tokenizes the source config (HAProxy uses indentation-aware section parsing; Nginx uses brace-delimited blocks). For each known directive, it constructs the corresponding IR entity. Unknown directives are preserved as `OpaqueBlock` IR entities — the IR retains them verbatim, and the forward translator emits them unchanged at the original location. This guarantees that a user who imports an exotic HAProxy config edits only what Mizan understands, and the rest survives untouched.

```go
type OpaqueBlock struct {
    ID      string
    Section string  // "frontend web", "global", "http", ...
    Lines   []string
    Anchor  string  // ID of the entity it should appear after
}
```

---

## 6. Wizard ↔ Topology Sync

### 6.1 Single Source of Truth

The IR is canonical. Both views are projections.

```
┌──────────────────┐      mutation       ┌──────────────────┐
│   Wizard form    │ ──────────────────► │                  │
│                  │ ◄────────────────── │  IR (Zustand)    │
└──────────────────┘    re-render        │                  │
                                          │                  │
┌──────────────────┐      mutation       │                  │
│ Topology canvas  │ ──────────────────► │                  │
│                  │ ◄────────────────── │                  │
└──────────────────┘    re-render        └────────┬─────────┘
                                                   │
                                       PATCH /ir   │   server pushes new
                                                   ▼   version on conflict
                                          ┌──────────────────┐
                                          │   Backend store  │
                                          └──────────────────┘
```

### 6.2 Mutation Types

A small set of typed mutations covers all edits. Defined in TypeScript:

```ts
type IRMutation =
  | { type: 'frontend.create';   data: Frontend }
  | { type: 'frontend.update';   id: string; data: Partial<Frontend> }
  | { type: 'frontend.delete';   id: string }
  | { type: 'backend.create';    data: Backend }
  | { type: 'backend.update';    id: string; data: Partial<Backend> }
  | { type: 'backend.delete';    id: string }
  | { type: 'server.create';     backendId: string; data: Server }
  | { type: 'server.update';     id: string; data: Partial<Server> }
  | { type: 'server.delete';     id: string }
  | { type: 'rule.create';       frontendId: string; data: Rule }
  | { type: 'rule.update';       id: string; data: Partial<Rule> }
  | { type: 'rule.delete';       id: string }
  | { type: 'connection.create'; from: string; to: string }
  | { type: 'connection.delete'; from: string; to: string }
  | { type: 'view.move';         entityId: string; x: number; y: number }
  | { type: 'view.zoom';         zoom: number }
  /* ... TLS, health, rate, cache, logger ... */
```

The Go side has the mirror types. Schema is generated from a single YAML source so the two cannot drift.

### 6.3 Optimistic Concurrency

PATCH requests carry `If-Match: <version>`. On conflict, the server returns `409` with the current IR; the frontend rebases the user's mutation if possible, or shows a "config changed elsewhere — reload?" dialog.

---

## 7. Storage with File Locking

### 7.1 Atomic Writes

```go
// internal/store/atomic.go
func atomicWrite(path string, data []byte) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil { return err }
    defer os.Remove(tmp.Name())  // best-effort cleanup if rename fails

    if _, err := tmp.Write(data); err != nil { tmp.Close(); return err }
    if err := tmp.Sync(); err != nil { tmp.Close(); return err }
    if err := tmp.Close(); err != nil { return err }

    return os.Rename(tmp.Name(), path)
}
```

### 7.2 File Locking (`flock`)

```go
// internal/store/lock.go (Unix)
//go:build !windows

import "golang.org/x/sys/unix"

type Lock struct{ fd int }

func Acquire(path string, timeout time.Duration) (*Lock, error) {
    fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT, 0o600)
    if err != nil { return nil, err }
    deadline := time.Now().Add(timeout)
    for {
        if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err == nil {
            return &Lock{fd: fd}, nil
        }
        if time.Now().After(deadline) {
            unix.Close(fd)
            return nil, ErrLocked
        }
        time.Sleep(50 * time.Millisecond)
    }
}

func (l *Lock) Release() error {
    if err := unix.Flock(l.fd, unix.LOCK_UN); err != nil { return err }
    return unix.Close(l.fd)
}
```

Windows uses `LockFileEx` from `golang.org/x/sys/windows` in a separate file with `//go:build windows`.

### 7.3 Snapshot Pruning

After every save, prune snapshots older than the most recent N (default 200) **and** older than M days (default 90), whichever is more restrictive — but never delete tagged snapshots. Pruning runs in a goroutine after the write commits.

---

## 8. Validation Pipeline

```go
type ValidateResult struct {
    Lint      []LintIssue        // IR structural issues
    Generated []byte             // emitted config bytes
    SourceMap SourceMap
    Native    *NativeResult      // result of haproxy -c / nginx -t
    DryRun    *DryRunResult      // optional: docker-based startup test
}
```

Pipeline:

1. **Lint pass** (pure Go, fast). Checks: dangling refs, port collisions, duplicate names, missing TLS for `:443`, empty backends, unreachable rules.
2. **Generate** to `[]byte` + source map.
3. **Native check**: write to temp file, invoke `haproxy -c -f <tmp>` or `nginx -t -c <tmp>` with timeout `5s`. Capture stdout/stderr. Map errors to entities via source map.
4. **Optional dry-run**: spin up `haproxy:alpine` or `nginx:alpine` with the temp file mounted; observe stderr for 1s of startup; tear down. Off by default; opt-in per project. Requires Docker on the operator's machine.

Error mapping example:

```
haproxy -c output:
  [ALERT] (12345) : config : parsing [<tmp>:47] : 'roundrobin2' is not recognized.

→ tmp file line 47 falls inside SourceMap entry { Start: 40, End: 55, EntityID: "be_app" }
→ surface as: { entityId: "be_app", severity: "error", message: "'roundrobin2' is not recognized" }
→ topology: red badge on backend node "app-pool"
→ wizard:   red border on the algorithm field of that backend
```

---

## 9. SSH Deployment

### 9.1 Connection Lifecycle

```go
// internal/deploy/ssh.go
type Client struct {
    target Target
    conn   *ssh.Client
    sftp   *sftp.Client
}

func Connect(ctx context.Context, t Target, vault *secrets.Vault) (*Client, error) {
    cfg, err := buildSSHConfig(t, vault)         // agent → key file → password
    if err != nil { return nil, err }
    cfg.HostKeyCallback = hostkey.TOFU(t.HostKeyFile)

    addr := net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
    conn, err := ssh.Dial("tcp", addr, cfg)
    if err != nil { return nil, err }
    sftpClient, err := sftp.NewClient(conn)
    ...
}
```

### 9.2 Deploy State Machine

```
                ┌──────────┐
                │   IDLE   │
                └────┬─────┘
                     │ start
                     ▼
              ┌──────────────┐  fail   ┌──────────┐
              │   CONNECT    │────────►│  FAILED  │
              └────┬─────────┘         └──────────┘
                   │ ok
                   ▼
              ┌──────────────┐  fail   ┌──────────┐
              │   UPLOAD     │────────►│  FAILED  │
              │   to /tmp    │         └──────────┘
              └────┬─────────┘
                   │ ok
                   ▼
              ┌──────────────┐  fail   ┌──────────┐
              │  REMOTE-CHK  │────────►│  FAILED  │ (no rollback needed; prod untouched)
              │  (haproxy -c)│         └──────────┘
              └────┬─────────┘
                   │ ok
                   ▼
              ┌──────────────┐
              │   BACKUP     │
              │   prod →     │
              │   .mizan-bak │
              └────┬─────────┘
                   │
                   ▼
              ┌──────────────┐  fail   ┌──────────────┐
              │  ATOMIC-MV   │────────►│  RESTORE     │
              │  tmp → prod  │         │  bak → prod  │
              └────┬─────────┘         └──────────────┘
                   │ ok
                   ▼
              ┌──────────────┐  fail   ┌──────────────┐
              │   RELOAD     │────────►│  RESTORE+    │
              │  systemctl   │         │  RE-RELOAD   │
              └────┬─────────┘         └──────────────┘
                   │ ok
                   ▼
              ┌──────────────┐  fail   ┌──────────────┐
              │  POST-PROBE  │────────►│  RESTORE+    │
              │  HTTP / -c   │         │  RE-RELOAD   │
              └────┬─────────┘         └──────────────┘
                   │ ok
                   ▼
              ┌──────────────┐
              │   SUCCESS    │
              └──────────────┘
```

### 9.3 Cluster Orchestration

A Cluster has `parallelism` (default 1) and `gate_on_failure` (default `halt`). The orchestrator runs the per-target state machine in batches of size `parallelism`. After each batch, if any target failed and `gate_on_failure == halt`, remaining batches are skipped and the audit entry records the partial state. Successful targets remain on the new config (this is intentional — they passed their probes).

---

## 10. Live Monitoring

### 10.1 Collector Interface

```go
type Collector interface {
    Name() string
    Poll(ctx context.Context) (Snapshot, error)
}

type Snapshot struct {
    TargetID  string
    Timestamp time.Time
    Backends  []BackendStat
    Servers   []ServerStat
    Frontends []FrontendStat
    Process   ProcessStat
}
```

### 10.2 HAProxy Runtime API

HAProxy exposes a control socket (UNIX or TCP) via the `stats socket` global directive. The collector:

1. Opens a socket connection (using `net.Dial("unix", ...)` for UNIX or `net.Dial("tcp", ...)` for TCP — TCP requires `expose-fd listeners` with care).
2. Sends `show stat\n`.
3. Reads a CSV stream until empty line.
4. Parses each row into the appropriate `BackendStat` / `ServerStat`.
5. Sends `show info\n` for process-level metrics.
6. Closes.

Polling is per-target with a single goroutine; jitter introduced on startup to avoid thundering herd across many targets.

### 10.3 Nginx OSS

`stub_status` returns a small text payload:

```
Active connections: 291
server accepts handled requests
 16630948 16630948 31070465
Reading: 6 Writing: 179 Waiting: 106
```

The collector parses this into `FrontendStat` aggregate metrics. Per-server stats are not available in OSS Nginx without third-party modules; Mizan surfaces this limitation in the UI.

### 10.4 Nginx Plus

`/api/9/http/upstreams` returns rich JSON: per-upstream peer health, requests, responses by class, sent/received bytes, server-side checks. The collector uses `encoding/json` directly into typed structs.

### 10.5 Ring Buffer

Per-`(target, metric, dimension)` time series stored in a fixed-size ring buffer:

```go
type Ring struct {
    mu     sync.RWMutex
    data   []Point        // pre-allocated
    head   int
    full   bool
    cap    int
}

type Point struct {
    T time.Time
    V float64
}
```

Default capacity 17,280 (24h × 5s). When the buffer fills, `head` wraps; reads return points in time order via two-segment iteration.

---

## 11. Real-Time Updates (SSE)

Two streams:

### 11.1 Monitor Stream

`GET /api/v1/projects/{id}/monitor/stream` returns `text/event-stream`. Server writes one event per polling cycle:

```
event: snapshot
data: {"target_id":"t-1","ts":"2026-04-25T18:42:11Z","backends":[...],...}

event: snapshot
data: {...}
```

Client uses native `EventSource`. On disconnect, `EventSource` reconnects; the server emits `id:` headers so the client can resume from `Last-Event-ID`.

### 11.2 Project Stream

`GET /api/v1/projects/{id}/events` streams IR change notifications, deploy progress, and audit entries. This is what enables a second open browser tab on the same project to see live updates from the first tab.

A simple in-process pub/sub (`map[string][]chan Event` with a mutex) backs both streams. No Redis, no NATS — single-process by design.

---

## 12. Build & Distribution

### 12.1 Single Binary with Embedded UI

```go
// internal/server/embed.go
import "embed"

//go:embed all:dist
var webuiFS embed.FS

func embeddedUI() http.Handler {
    sub, _ := fs.Sub(webuiFS, "dist")
    return spaHandler(http.FS(sub))  // serves files; SPA fallback to index.html
}
```

The `webui/dist` folder is built by `vite build` and copied to `internal/server/dist` before `go build`. The Makefile orchestrates:

```makefile
.PHONY: ui binary release

ui:
	cd webui && pnpm install && pnpm build
	rm -rf internal/server/dist
	cp -r webui/dist internal/server/dist

binary: ui
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/mizanproxy/mizan/internal/version.Version=$(VERSION) -X github.com/mizanproxy/mizan/internal/version.Commit=$(COMMIT) -X github.com/mizanproxy/mizan/internal/version.Date=$(DATE)" -o dist/mizan ./cmd/mizan

release:
	# matrix build: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
	./scripts/release.sh
```

`CGO_ENABLED=0` and `-trimpath` ensure reproducible, statically-linked binaries with no host paths leaking into debug info.

### 12.2 Reproducible Builds

`go.mod` and `go.sum` are committed; Go 1.23+ `GOFLAGS=-trimpath` is mandatory in CI. Two CI runners building the same commit must produce byte-identical binaries (modulo timestamps in `version.Date`, which the release pipeline pins).

---

## 13. Security

### 13.1 Threat Model

Operators are trusted. The threat surface is:

- **Network exposure of the WebUI** (mitigation: localhost bind by default, mandatory auth when bound externally).
- **Compromise of `~/.mizan/secrets/`** (mitigation: AES-256-GCM with Argon2id-derived key from a passphrase; the passphrase is never written to disk).
- **MITM on SSH deploys** (mitigation: TOFU host key pinning per Target).
- **Supply chain** (mitigation: stdlib-first; allowed external deps audited; `go.sum` enforced; CI uses `govulncheck`).

### 13.2 Argon2id Vault

```go
// internal/secrets/kdf.go
const (
    argonTime    = 3
    argonMemory  = 64 * 1024  // 64 MiB
    argonThreads = 4
    keyLen       = 32
    saltLen      = 16
)

func DeriveKey(passphrase, salt []byte) []byte {
    return argon2.IDKey(passphrase, salt, argonTime, argonMemory, argonThreads, keyLen)
}
```

Vault file layout: `salt(16) || nonce(12) || ciphertext`. Decrypt requires the passphrase entered at `mizan serve` startup or `mizan deploy` invocation; the derived key lives only in memory.

### 13.3 No Logged Secrets

A `slog` middleware redacts known sensitive keys (`password`, `token`, `passphrase`, `private_key`, etc.) from log records. Secret values never appear in HTTP request/response logs.

---

## 14. Testing Strategy

### 14.1 Backend

| Layer | Tool | Coverage Target |
|-------|------|-----------------|
| `internal/ir` | `testing` + table-driven | 95% — pure logic, easy to test |
| `internal/translate/{haproxy,nginx}` | golden file tests | 100% of supported directives |
| `internal/ir/parser/{haproxy,nginx}` | golden file tests with round-trip | 100% of supported directives |
| `internal/store` | filesystem tests with `t.TempDir()` | 90% |
| `internal/deploy` | mocked SSH server (`embedded sshd`) | 80% |
| `internal/api` | `httptest.Server` + table tests | 85% |

**Golden file tests** are key for the translator: `testdata/haproxy/<scenario>/in.json` (IR) + `out.cfg` (expected). Running `go test -update` regenerates expected output during development; CI fails if `git diff` is non-empty after tests.

### 14.2 Frontend

| Layer | Tool |
|-------|------|
| Pure functions (graph build, layout) | Vitest unit tests |
| Components (wizard steps, node renderers) | Vitest + React Testing Library |
| End-to-end (wizard ↔ topology sync, deploy flow) | Playwright |

E2E scenarios:

- New project → wizard creates frontend + backend → topology shows nodes → drag connection → wizard reflects ACL.
- Import existing config → both views populate identically.
- Generate → invoke validate (mocked) → see error highlights.
- Deploy (mocked SSH) → see audit entry → see live monitor stream (mocked).

### 14.3 Cross-Stack Contract Tests

A small Go test program exercises the REST API end-to-end: spin up the full server in-process, run a Playwright suite against it via `pnpm test:contract`. This catches schema drift between frontend Zod and backend Go types.

---

*End of IMPLEMENTATION.md*

# Mizan

> **Visual Config Architect for HAProxy & Nginx**
> Design once on a shared canvas. Validate locally. Deploy via SSH. Watch it run. Single Go binary, embedded WebUI.

[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8)]() [![License: MIT](https://img.shields.io/badge/License-MIT-blue)]() [![Single Binary](https://img.shields.io/badge/binary-single-success)]() [![Status: Planning](https://img.shields.io/badge/status-planning-orange)]()

---

## Why Mizan

Writing HAProxy or Nginx configurations by hand is fast for one server and miserable for fifty. Existing GUIs are either commercial-only (HAProxy Enterprise, Nginx Plus dashboard) or fragmented across one-off bash scripts and Ansible roles. Mizan replaces that mess with a single binary that runs anywhere, persists state as Git-friendly JSON, and renders the same load balancer topology two ways: as a **wizard form** and as an **interactive canvas** that stay in two-way sync.

Mizan does **not** run traffic. It is not a load balancer. It is the **architect's tool** that produces correct `haproxy.cfg` and `nginx.conf` files, validates them, ships them to your servers, reloads safely, and surfaces live runtime metrics so you know the change took effect.

---

## Features

**Universal IR (Intermediate Representation)** — One abstract model represents your topology. The same model emits valid HAProxy *or* Nginx output, with target-specific warnings for features that don't translate.

**Hybrid Wizard ↔ Topology** — Edit a frontend's listening port in a form field; watch the canvas update. Drag a backend onto a frontend in the canvas; watch the form fields fill in. Both views read from and write to the same in-memory IR.

**Live Validation** — Before you save, Mizan runs the actual `haproxy -c -f` or `nginx -t -c` binaries (sandboxed on a temp file) and surfaces line-numbered diagnostics directly on the canvas node that caused them.

**Git-Friendly Storage** — All projects are JSON files under `~/.mizan/`. Diff, commit, branch, merge. No SQLite, no proprietary blob.

**SSH Deployment** — Push to single hosts or clusters. Atomic file swap, automatic rollback if reload fails, gradual rollout for clusters with health gating.

**Live Monitoring** — Connect to HAProxy's Runtime API socket or Nginx's `stub_status` / `ngx_http_api_module`. Stream metrics over SSE to the WebUI, render time-series charts, surface unhealthy backends as red on the topology canvas.

**Versioning & Audit** — Every save snapshots the IR. Every deploy logs *who*, *when*, *to where*, *what changed*, *result*. Roll back to any prior version with one click.

**Single Binary** — Backend in Go, frontend embedded via `embed`. Run `./mizan` and open `http://localhost:7890`. No Docker, no Node runtime, no external database.

---

## Quick Start

```bash
# Install
go install github.com/mizanproxy/mizan/cmd/mizan@latest
# or grab a release binary from GitHub Releases

# Initialize a new project
mizan project new "edge-cluster" --target haproxy

# Launch the WebUI
mizan serve
# → open http://localhost:7890

# Or use the CLI for everything
mizan frontend add --name web --bind ':443' --tls
mizan backend  add --name api-pool --algorithm leastconn
mizan backend  attach api-pool --frontend web --acl 'path /api/*'
mizan server   add api-pool 10.0.1.10:8080 --weight 100 --check
mizan generate --target haproxy --out /etc/haproxy/haproxy.cfg
mizan validate --target haproxy
mizan deploy   --to lb-prod-1.example.com --reload
```

---

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│                  Mizan (single binary)                      │
│                                                             │
│  ┌──────────────┐    ┌──────────────────────────────────┐   │
│  │ Embedded     │    │  Go HTTP Server                  │   │
│  │ React 19     │◄──►│  • REST API   • SSE stream       │   │
│  │ + Tailwind 4 │    │  • Static UI  • WebSocket (ops)  │   │
│  │ + shadcn/ui  │    └────────────┬─────────────────────┘   │
│  │ + React Flow │                 │                         │
│  └──────────────┘                 ▼                         │
│                       ┌────────────────────────┐            │
│                       │  Universal IR (model)  │            │
│                       └─────┬──────────┬───────┘            │
│                             ▼          ▼                    │
│                    ┌────────────┐ ┌────────────┐            │
│                    │  HAProxy   │ │   Nginx    │            │
│                    │ Translator │ │ Translator │            │
│                    └────────────┘ └────────────┘            │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────┐   │
│  │ JSON File Store  │  │ SSH Deployer     │  │ Monitor  │   │
│  │ + flock + atomic │  │ + rollback       │  │ collector│   │
│  └──────────────────┘  └──────────────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌─────────┐  ┌─────────┐  ┌─────────┐
        │ HAProxy │  │ Nginx   │  │ HAProxy │
        │  node   │  │  node   │  │  node   │
        └─────────┘  └─────────┘  └─────────┘
```

---

## Tech Stack

**Backend**: Go 1.23+ · stdlib-first · `golang.org/x/crypto/ssh` · `pkg/sftp` · `embed` for UI assets

**Frontend**: React 19 · Vite 5+ · TypeScript strict · Tailwind CSS v4.1 · shadcn/ui (Radix-based) · `@xyflow/react` (React Flow) · Zustand · TanStack Query · React Hook Form + Zod · Lucide icons · Recharts

**Targets**: HAProxy 2.8+ · Nginx 1.24+ (open source) · Nginx Plus optional for richer telemetry

---

## Roadmap

12 phases across roughly 12 months. See [`TASKS.md`](./TASKS.md) for the full breakdown.

| Phase | Goal |
|-------|------|
| 1 | Foundation: repo, CI, base Go module, base React app |
| 2 | Universal IR data model |
| 3 | HAProxy translator (IR → cfg, cfg → IR) |
| 4 | Nginx translator (IR → conf, conf → IR) |
| 5 | JSON storage with file locking & atomic writes |
| 6 | REST API + WebUI skeleton (theme, routing, embed) |
| 7 | Wizard UI (all steps, form binding) |
| 8 | React Flow topology with two-way sync |
| 9 | Validation pipeline (lint + dry-run) |
| 10 | SSH deployment & cluster orchestration |
| 11 | Live monitoring & runtime telemetry |
| 12 | Audit, versioning, diff viewer, polish |

---

## Documentation

- [`SPECIFICATION.md`](./SPECIFICATION.md) — Functional and non-functional requirements
- [`IMPLEMENTATION.md`](./IMPLEMENTATION.md) — Architecture, packages, key algorithms
- [`TASKS.md`](./TASKS.md) — Phased task breakdown with hour estimates
- [`BRANDING.md`](./BRANDING.md) — Name, palette, typography, voice

---

## Status

**Planning phase.** Documentation complete, implementation has not begun. Star the repo to follow along.

---

## Philosophy

Mizan is part of the **#NOFORKANYMORE** family — built from scratch in pure Go, zero runtime dependencies on the deployment target, single binary, embedded WebUI, JSON-backed storage. If you can `scp` it, you can run it.

---

## License

MIT. Use it, fork it, ship it. Just don't claim you wrote it.

---

*Three Pans. One Beam. All Traffic.*

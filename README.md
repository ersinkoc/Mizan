# Mizan

Mizan is a single-binary visual config architect for HAProxy and Nginx. It stores projects as JSON under `~/.mizan`, serves an embedded React UI, and can generate basic HAProxy/Nginx configs from a shared IR.

This repository currently contains the working v0 foundation:

- Go CLI and HTTP server: `mizan serve`
- Project CRUD and IR persistence with snapshots
- HAProxy/Nginx import into IR for the core supported directives
- Snapshot listing, tagging, retrieval, and revert
- Structural IR linting and deterministic snapshot hashes
- HAProxy and Nginx config generation
- Native validation wrapper when `haproxy` or `nginx` exists on `PATH`
- React/Vite WebUI for project creation, IR editing, generation, and diagnostics

## Run

```sh
go run ./cmd/mizan serve
```

Open `http://127.0.0.1:7890`.

For frontend development:

```sh
cd webui
npm install
npm run dev
```

## CLI Examples

```sh
go run ./cmd/mizan project new --name edge-prod --engines haproxy,nginx
go run ./cmd/mizan project import ./haproxy.cfg --name imported-edge
go run ./cmd/mizan project list
go run ./cmd/mizan snapshot list --project <id>
go run ./cmd/mizan snapshot tag --project <id> --label release-1 <snapshot-ref>
go run ./cmd/mizan generate --project <id> --target haproxy
go run ./cmd/mizan validate --project <id> --target nginx
```

## Build

```sh
make ui
make binary
```

## Test and Coverage

```sh
make test
make coverage
```

Frontend core library coverage is gated at 100% statements, 100% functions, 100% lines, and 90% branches. Backend coverage is reported with `go tool cover`; the current backend goal is to keep raising package coverage without weakening the all-green test gate.

On Windows without `make`, run:

```powershell
cd webui
npm install
npm run build
cd ..
Remove-Item -Recurse -Force internal/server/dist
Copy-Item -Recurse webui/dist internal/server/dist
go build -o dist/mizan.exe ./cmd/mizan
```

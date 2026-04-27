VERSION ?= 0.1.0-dev
COMMIT ?= local
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: dev ui binary test coverage lint vuln e2e clean

dev:
	go run ./cmd/mizan serve

ui:
	cd webui && npm install && npm run build
	rm -rf internal/server/dist
	cp -r webui/dist internal/server/dist

binary: ui
	go build -trimpath -ldflags="-s -w -X github.com/mizanproxy/mizan/internal/version.Version=$(VERSION) -X github.com/mizanproxy/mizan/internal/version.Commit=$(COMMIT) -X github.com/mizanproxy/mizan/internal/version.Date=$(DATE)" -o dist/mizan ./cmd/mizan

test:
	go test ./...
	cd webui && npm test

coverage:
	mkdir -p dist
	go test -coverprofile dist/coverage.out ./...
	go tool cover -func dist/coverage.out
	cd webui && npm run test:coverage

lint:
	go test ./...
	cd webui && npm run lint

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	cd webui && npm audit --audit-level=low

e2e:
	cd webui && npm run test:e2e

clean:
	rm -rf dist webui/dist internal/server/dist/assets

FROM node:22-bookworm-slim AS webui

WORKDIR /src/webui
COPY webui/package.json webui/package-lock.json ./
RUN npm ci
COPY webui/ ./
RUN npm run build

FROM golang:1.23-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=webui /src/webui/dist ./internal/server/dist

ARG VERSION=0.1.0-dev
ARG COMMIT=container
ARG DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X github.com/mizanproxy/mizan/internal/version.Version=${VERSION} -X github.com/mizanproxy/mizan/internal/version.Commit=${COMMIT} -X github.com/mizanproxy/mizan/internal/version.Date=${DATE}" \
    -o /out/mizan ./cmd/mizan

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates openssh-client \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home --home-dir /var/lib/mizan --shell /usr/sbin/nologin mizan \
    && mkdir -p /var/lib/mizan \
    && chown -R mizan:mizan /var/lib/mizan

COPY --from=build /out/mizan /usr/local/bin/mizan

USER mizan
WORKDIR /var/lib/mizan
VOLUME ["/var/lib/mizan"]
EXPOSE 7890

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["mizan", "doctor", "--home", "/var/lib/mizan", "--json"]

ENTRYPOINT ["mizan"]
CMD ["serve", "--bind", "0.0.0.0:7890", "--home", "/var/lib/mizan"]

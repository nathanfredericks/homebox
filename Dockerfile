# syntax=docker/dockerfile:1

# ---------------------------------------
# Node dependencies stage
# ---------------------------------------
FROM public.ecr.aws/docker/library/node:22-alpine AS frontend-dependencies
WORKDIR /app

# Install pnpm 10 (latest stable, works reliably in Alpine)
RUN npm install -g pnpm@10

# Copy package.json and lockfile to leverage caching
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# ---------------------------------------
# Build Next.js (frontend) stage
# ---------------------------------------
FROM public.ecr.aws/docker/library/node:22-alpine AS frontend-builder
WORKDIR /app

# Install pnpm 10 (latest stable)
RUN npm install -g pnpm@10

# Copy over source files and node_modules from dependencies stage
COPY frontend .
COPY --from=frontend-dependencies /app/node_modules ./node_modules
# Produces .next/standalone (server.js + minimal node_modules), .next/static
# (hashed client assets) and public/ (static files). See next.config.ts
# `output: "standalone"`.
RUN pnpm build

# ---------------------------------------
# Go dependencies stage
# ---------------------------------------
FROM public.ecr.aws/docker/library/golang:alpine AS builder-dependencies
WORKDIR /go/src/app

# Copy go.mod and go.sum for better caching
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

# ---------------------------------------
# Build API stage
# ---------------------------------------
FROM public.ecr.aws/docker/library/golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TIME
ARG COMMIT
ARG VERSION

# Install necessary build tools
RUN apk update && \
    apk upgrade && \
    apk add --no-cache git build-base gcc g++ && \
    if [ "$TARGETARCH" != "arm" ] || [ "$TARGETARCH" != "riscv64" ]; then apk --no-cache add libwebp libavif libheif libjxl; fi

WORKDIR /go/src/app

# Copy Go modules (from dependencies stage) and source code
COPY --from=builder-dependencies /go/pkg/mod /go/pkg/mod
COPY ./backend .

# The frontend is no longer embedded in the Go binary; it is served by the
# Next.js standalone server. backend/app/api/static/public ships a small
# placeholder page (the API's notFoundHandler still embeds it via go:embed).

# Use cache for Go build artifacts
RUN --mount=type=cache,target=/root/.cache/go-build \
    if [ "$TARGETARCH" = "arm" ] || [ "$TARGETARCH" = "riscv64" ];  \
    then echo "nodynamic" $TARGETOS $TARGETARCH; CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
        -ldflags "-s -w -X main.commit=$COMMIT -X main.buildTime=$BUILD_TIME -X main.version=$VERSION" \
        -tags nodynamic -o /go/bin/api -v ./app/api/*.go; \
    else \
         echo $TARGETOS $TARGETARCH; CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
        -ldflags "-s -w -X main.commit=$COMMIT -X main.buildTime=$BUILD_TIME -X main.version=$VERSION" \
        -o /go/bin/api -v ./app/api/*.go; \
    fi

# ---------------------------------------
# Caddy binary stage
# ---------------------------------------
# Grab a static caddy binary from the official image rather than building it.
FROM public.ecr.aws/docker/library/caddy:2-alpine AS caddy

# ---------------------------------------
# Production stage
# ---------------------------------------
FROM public.ecr.aws/docker/library/node:22-alpine
ARG TARGETARCH
ENV HBOX_MODE=production
ENV HBOX_STORAGE_CONN_STRING=file:///?no_tmp_dir=true
ENV HBOX_STORAGE_PREFIX_PATH=data
ENV HBOX_DATABASE_SQLITE_PATH=/data/homebox.db?_pragma=busy_timeout=2000&_pragma=journal_mode=WAL&_fk=1&_time_format=sqlite

# Bind the Go API to loopback only; Caddy is the single public listener on 7745.
ENV HBOX_WEB_HOST=127.0.0.1
ENV HBOX_WEB_PORT=7746

# Next.js standalone server: loopback only, fronted by Caddy.
ENV PORT=3000
ENV HOSTNAME=127.0.0.1
ENV NODE_ENV=production
# SSR data fetches hit the Go API directly on loopback, bypassing Caddy. In dev
# this defaults to http://localhost:7745.
ENV API_BASE_URL=http://127.0.0.1:7746

# s6-overlay version pinned for reproducible builds.
ARG S6_OVERLAY_VERSION=3.2.0.2

# Install runtime dependencies. xz is needed to unpack the s6-overlay tarballs.
RUN apk --no-cache add ca-certificates wget mosquitto-clients xz && \
    if [ "$TARGETARCH" != "arm" ] || [ "$TARGETARCH" != "riscv64" ]; then apk --no-cache add libwebp libavif libheif libjxl; fi

# Install s6-overlay (PID 1 supervisor for caddy + go + node).
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz /tmp/
RUN tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz && rm -f /tmp/s6-overlay-noarch.tar.xz
# Pick the architecture-specific tarball matching the build platform.
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-x86_64.tar.xz /tmp/s6-overlay-x86_64.tar.xz
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-aarch64.tar.xz /tmp/s6-overlay-aarch64.tar.xz
RUN case "${TARGETARCH:-$(uname -m)}" in \
      amd64|x86_64)  tar -C / -Jxpf /tmp/s6-overlay-x86_64.tar.xz ;; \
      arm64|aarch64) tar -C / -Jxpf /tmp/s6-overlay-aarch64.tar.xz ;; \
      *) echo "unsupported arch: ${TARGETARCH:-$(uname -m)}" >&2; exit 1 ;; \
    esac && \
    rm -f /tmp/s6-overlay-*.tar.xz

# Caddy binary from the official image.
COPY --from=caddy /usr/bin/caddy /usr/bin/caddy

# Go API binary.
RUN mkdir -p /app /app/ui
COPY --from=builder /go/bin/api /app/api
RUN chmod +x /app/api

# Next.js standalone server. The standalone output already contains a minimal
# node_modules and server.js; .next/static and public/ must be copied in
# alongside (Next does not bundle them into standalone).
COPY --from=frontend-builder /app/.next/standalone /app/ui
COPY --from=frontend-builder /app/.next/static /app/ui/.next/static
COPY --from=frontend-builder /app/public /app/ui/public

# Caddy config and s6 service definitions.
COPY docker/Caddyfile /etc/caddy/Caddyfile
COPY docker/rootfs/ /

# Labels and configuration for the final image
LABEL Name=homebox Version=0.0.1
LABEL org.opencontainers.image.source="https://github.com/sysadminsmedia/homebox"

# Expose necessary ports for Homebox
EXPOSE 7745
WORKDIR /app

# Healthcheck exercises the full path Caddy -> Go API.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD [ "wget", "--no-verbose", "--tries=1", "-O", "-", "http://localhost:7745/api/v1/status" ]

# Persist volume
VOLUME [ "/data" ]

# s6-overlay is PID 1 and supervises caddy, the Go API and the Next server.
ENTRYPOINT [ "/init" ]

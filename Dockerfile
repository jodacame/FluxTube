# FluxTube — multi-stage build:
#   node builds the UI → go compiles & embeds it → runtime with ffmpeg, the
#   latest yt-dlp and the Deno JS runtime (required to resolve all formats).

# ---------- stage 1: build the React UI ----------
FROM node:22-bookworm-slim AS ui
WORKDIR /ui
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ---------- stage 2: compile the Go binary (UI embedded) ----------
FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /ui/dist ./web/dist
ARG TARGETOS TARGETARCH VERSION=docker
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X github.com/jodacame/fluxtube/internal/api.Version=${VERSION}" \
    -o /out/fluxtube ./cmd/fluxtube

# ---------- stage 3: runtime ----------
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg python3 ca-certificates curl unzip \
    && rm -rf /var/lib/apt/lists/*

# Latest yt-dlp at build time (kept current on every image rebuild).
RUN curl -fsSL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
      -o /usr/local/bin/yt-dlp && chmod +x /usr/local/bin/yt-dlp

# Deno JS runtime — yt-dlp needs it to solve YouTube's player challenges and
# expose all adaptive formats.
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh \
    && /usr/local/bin/deno --version

COPY --from=build /out/fluxtube /usr/local/bin/fluxtube

EXPOSE 7002
ENV FT_CONFIG_DIR=/config FT_CACHE_PATH=/cache FT_LISTEN_HOST=0.0.0.0 FT_LISTEN_PORT=7002 \
    PATH=/usr/local/bin:/usr/bin:/bin
VOLUME ["/config", "/cache"]
ENTRYPOINT ["/usr/local/bin/fluxtube"]

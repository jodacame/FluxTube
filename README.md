<div align="center">

# ◢ FluxTube

**A simple, efficient, self-hosted YouTube → HTTP streaming bridge with a web UI.**

Give it a YouTube video and FluxTube turns it into a **seekable HTTP stream** with
selectable **audio and subtitle tracks** for any modern player — plus a headless,
keyless **discovery API** to search and browse, and a built-in web client to watch.

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](#-quick-start-docker)

</div>

---

> Part of the **Flux** family, alongside
> [FluxTorrent](https://github.com/jodacame/FluxTorrent).

## Table of contents

- [What is it?](#what-is-it)
- [Features](#features)
- [Quick start (Docker)](#-quick-start-docker)
- [How it works](#how-it-works)
- [Configuration](#configuration)
- [API](#api)
- [Building from source](#building-from-source)
- [Tech stack](#tech-stack)
- [Security](#security)
- [License](#license)
- [Disclaimer](#disclaimer)

## What is it?

FluxTube does one job well: it turns a YouTube video into a **ready-to-consume HTTP
stream** on demand, with multiple audio and subtitle tracks the player can choose
from — like an MKV, but **seekable**. It also exposes a small **discovery API** so
clients can search, browse channels and get recommendations.

It is a **thin, mostly stateless bridge**: it stores no media, keeps only an
ephemeral segment cache, and runs comfortably on modest hardware (a NAS or a small
always-on box).

## Features

- 🎬 **Ready-to-consume streaming** — HLS (fragmented MP4) remuxed with stream copy (no re-encoding), so playback starts fast and stays light.
- 🔊 **Multi-track, like an MKV** — selectable audio languages and subtitle tracks (manual + a curated set of auto-captions as WebVTT).
- ⏩ **Seekable** — segmented HLS means fast-forward/seek works in real players.
- 🪶 **Efficient by design** — `-c copy` (no transcode), bounded ephemeral cache, idle sessions are torn down. Minimal upstream requests via aggressive caching and single-flight.
- 🔎 **Headless discovery API** — keyless search, trending, channels, playlists, related and a stateless recommended feed (client owns its history/follows).
- 🖥️ **Two web UIs in one** — a management dashboard (`/`) and a YouTube-like web client (`/app`) with an `hls.js` player and track selectors.
- 🧭 **Per-source rules** — match by channel / title / id → reject, cap quality, prefer audio/subtitle language, force cache/ephemeral, or mark as music.
- 🎵 **Music mode** — search the official **YouTube Music** catalog and play **audio-only** in a universal AAC/`m4a` format; songs are stored once (persistent, optimal — no re-download) and can be auto-detected and auto-saved.
- 🍪 **Optional cookies** — point to a cookies file to unlock restricted videos.
- 🪶 **Single binary, single container** — UI embedded via `go:embed`.

## 🚀 Quick start (Docker)

```bash
docker run -d --name fluxtube \
  -p 7002:7002 \
  -v "$PWD/config:/config" \
  --restart unless-stopped \
  ghcr.io/jodacame/fluxtube:latest
```

Open **http://localhost:7002/app** to browse and watch, or **http://localhost:7002/**
to manage the library, rules and settings.

### docker-compose

```yaml
services:
  fluxtube:
    image: ghcr.io/jodacame/fluxtube:latest   # or build: .
    container_name: fluxtube
    ports:
      - "7002:7002"
    volumes:
      - ./config:/config      # settings, rules, optional cookies
      - ./cache:/cache         # bounded ephemeral segment cache
    environment:
      - FT_LISTEN_PORT=7002
      # - FT_API_TOKEN=secret           # guard /api/* with a bearer token
      # - FT_COOKIES=/config/cookies.txt
    restart: unless-stopped
```

## How it works

1. **Resolve** — a video is resolved on demand and the result is cached (the cheap
   catalog metadata comes from a lightweight lookup; the full resolve is deferred
   until playback).
2. **Remux** — the player requests the HLS master; segments are produced on demand
   with `ffmpeg -c copy` (no re-encoding) and served from a bounded cache.
3. **Cleanup** — sessions are ephemeral: the cache is wiped when playback ends or
   after an idle timeout. The source is never fully downloaded — only what is played.

> **Note:** resolving the full set of qualities relies on a JavaScript runtime
> (bundled in the Docker image) and an up-to-date `yt-dlp` (installed fresh on every
> image build). Without them, only a basic progressive format may be available. For
> region- or sign-in-restricted videos, configure an optional cookies file.

## Music mode

FluxTube can act as a lightweight music service backed by YouTube:

- In the web client (`/app`), toggle **🎵 Music** next to the search box. Searches then
  hit the official **YouTube Music** catalog, so the top hit is the official track.
- Playing a music result streams **audio only** as **AAC in an `.m4a` container**
  (`-c copy` when the source is already AAC, otherwise a light transcode) with a
  front-loaded index, so it plays and seeks in **any player** — including a plain
  browser `<audio>` element, VLC, etc.: `GET /stream/<id>/audio.m4a`.
- The audio is written **once** to a persistent store and served directly afterwards,
  so a song is **never downloaded or processed twice**.
- **Auto-save** (Settings → Music, on by default): videos detected as music — by the
  YouTube *Music* category or auto-generated `- Topic`/Vevo channels — are saved as
  music automatically, no rule required. You can also mark sources as music with a
  **rule** (`action: music`).
- The persistent store path is configurable (Settings → Music, env `FT_MUSIC_DIR`,
  default `/config/music`). The status bar shows how much space the saved music uses
  and the free disk space.

## Configuration

Everything is editable in **Settings** and persisted to `/config`. A few values can
be seeded from environment variables:

| Env var | Default | Description |
|---|---|---|
| `FT_LISTEN_HOST` | `0.0.0.0` | Bind address |
| `FT_LISTEN_PORT` | `7002` | API + UI + stream port |
| `FT_CONFIG_DIR` | `/config` | Where the database lives |
| `FT_CACHE_PATH` | `/cache` | Ephemeral segment cache root |
| `FT_API_TOKEN` | _(empty)_ | Optional bearer token for `/api/*` |
| `FT_COOKIES` | _(empty)_ | Optional cookies file to unlock restricted videos |
| `FT_MUSIC_DIR` | `/config/music` | Persistent store for saved music (audio) |

From the UI you can also tune default quality, preferred audio/subtitle languages,
auto-caption languages, segment length, idle timeout, concurrency limits and rules.

## API

Base: `http://<host>:7002`. `/api/*` can be guarded by a bearer token; `/stream/*`
stays open so saved player URLs keep working.

```
POST   /api/videos              { "id": "<id>" | "url": "..." }
GET    /api/videos              → library + live state
GET    /api/videos/:id          → resolved info (qualities, audio, subtitle tracks)
POST   /api/videos/:id/stop     → stop the live session
DELETE /api/videos/:id          → remove + stop + clear cache
GET    /stream/:id/master.m3u8  → HLS master (?q=1080 to cap quality)   ← players
GET    /stream/:id/progressive  → progressive fallback
GET    /stream/:id/audio.m4a    → audio-only (AAC), persistent           ← music
GET    /api/discover/search?q=&music=1  → official YouTube Music results
GET/PUT /api/settings
GET/PUT /api/rules
WS     /api/events              → live state
GET    /api/health

# Headless discovery (keyless)
GET    /api/discover/search?q=&limit=
GET    /api/discover/trending
GET    /api/discover/channel/:channelId
GET    /api/discover/channel/:channelId/videos?page=
GET    /api/discover/playlist/:playlistId?page=
GET    /api/discover/video/:id
GET    /api/discover/related/:id
POST   /api/discover/recommended   { "channels": [...], "seeds": [...] }
```

## Building from source

Requires Go 1.23+, Node 18+, plus `ffmpeg`, `yt-dlp` and a JavaScript runtime
(Deno) on the host.

```bash
# 1. build the embedded UI
cd web && npm install && npm run build && cd ..

# 2. build the single binary (UI baked in)
go build -o fluxtube ./cmd/fluxtube

# 3. run
FT_CONFIG_DIR=./config FT_CACHE_PATH=./cache ./fluxtube   # → http://localhost:7002
```

Or just `docker build -t fluxtube .`.

## Tech stack

Go single binary with the web UI embedded via `go:embed` · `yt-dlp` + Deno for
resolution · `ffmpeg` for remuxing · React + Vite + TypeScript + Tailwind UI with
`hls.js` · Docker.

## Security

FluxTube is a **self-hosted service meant for a trusted network**. By default it has
**no authentication**, permissive CORS, and binds `0.0.0.0`.

- Set **`FT_API_TOKEN`** to require a bearer token on `/api/*`.
- `/stream` stays open by design so players work without credentials.
- To expose it publicly, keep it behind a **VPN or reverse proxy** (TLS + auth).

See [SECURITY.md](SECURITY.md) for how to report vulnerabilities.

## License

[Apache-2.0](LICENSE). Provided **"as is", without warranty of any kind**.

## Disclaimer

FluxTube is a **general-purpose streaming bridge**. It ships with **no content** and
does **not host, index or provide** any media. It uses third-party tools to access
publicly reachable resources on your behalf.

**You are solely responsible** for how you use it and for complying with all
applicable laws and with the Terms of Service of any platform you access. Accessing
or downloading content without authorization may be illegal in your jurisdiction, and
automated access may violate a platform's terms. Use FluxTube only with content you
have the right to access. By using the software you accept full responsibility for
your usage.

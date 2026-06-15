<div align="center">

# ◢ FluxTube

**A simple, efficient, self-hosted YouTube → HTTP streaming bridge with a web UI.**

Give it a YouTube video and FluxTube turns it into a seekable HTTP stream with
selectable audio and subtitle tracks for any modern player — plus a headless,
keyless discovery API to search and browse.

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](#quick-start)

</div>

---

> Part of the **Flux** family, alongside
> [FluxTorrent](https://github.com/jodacame/FluxTorrent).

## What is it?

FluxTube does one job well: it turns a YouTube video into a **ready-to-consume
HTTP stream** on demand, with multiple audio and subtitle tracks the player can
choose from, and exposes a small **discovery API** so clients can search and
browse. It is a thin, stateless streaming bridge — it stores no media and runs
comfortably on modest hardware.

## Quick start

```bash
docker run -d --name fluxtube \
  -p 7002:7002 \
  -v "$PWD/config:/config" \
  --restart unless-stopped \
  ghcr.io/jodacame/fluxtube:latest
```

Open **http://localhost:7002**.

## Tech stack

Go single binary with the web UI embedded via `go:embed`, `yt-dlp` + `ffmpeg`
for resolution and remuxing, React + Vite + TypeScript UI. Docker-ready.

## License

[Apache-2.0](LICENSE). Provided "as is", without warranty of any kind.

## Disclaimer

FluxTube is a general-purpose streaming bridge. It ships with no content and
does not host or index anything. You are solely responsible for how you use it
and for complying with all applicable laws and the terms of any service you
access.

# Security Policy

## Supported versions

FluxTube is under active development; security fixes target the latest `main` and
the most recent release.

## Reporting a vulnerability

Please report security issues privately rather than opening a public issue:

- Use GitHub's **"Report a vulnerability"** (Security → Advisories) on this
  repository, or
- Contact the maintainer through the address listed on the GitHub profile.

Please include enough detail to reproduce the issue. We aim to acknowledge reports
promptly and will coordinate a fix and disclosure timeline with you.

## Hardening notes

FluxTube is designed for a **trusted network**. By default it has no authentication,
permissive CORS, and binds all interfaces.

- Set `FT_API_TOKEN` to require a bearer token on `/api/*`.
- Do not expose the port directly to the internet; place it behind a VPN or a
  reverse proxy that terminates TLS and adds authentication.
- Treat any configured cookies file as a secret: it grants access to the associated
  account. Mount it read-only and keep it out of version control.
